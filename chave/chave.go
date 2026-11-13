// Package chave monta, valida e decompõe a chave de acesso de 44 posições dos
// documentos fiscais eletrônicos.
//
// A composição segue o Manual de Orientação do Contribuinte da NF-e:
//
//	cUF(2) AAMM(4) CNPJ(14) mod(2) série(3) nNF(9) tpEmis(1) cNF(8) cDV(1)
//
// A mesma estrutura é usada por NF-e (modelo 55), NFC-e (modelo 65), CT-e
// (57/67) e MDF-e (58), o que torna este pacote independente do documento.
//
// # CNPJ alfanumérico
//
// Pela Nota Técnica Conjunta 2025.001, as doze primeiras posições do CNPJ podem
// ter letras maiúsculas, e a chave passa a seguir o padrão
// [0-9]{6}[A-Z0-9]{12}[0-9]{26}. O dígito verificador continua no módulo 11,
// com cada caractere valendo o seu código ASCII menos 48 (A vale 17): uma chave
// só de dígitos tem o mesmo dígito de antes.
package chave

import (
	"crypto/rand"
	"errors"
	"fmt"
	"math/big"
	"strconv"
	"strings"

	"github.com/sisgt-sistemas/gonfe/uf"
)

// Tamanho é a quantidade de dígitos de uma chave de acesso completa.
const Tamanho = 44

// ErrChaveInvalida é a base dos erros devolvidos por [Parse] e [Validar].
var ErrChaveInvalida = errors.New("chave: chave de acesso inválida")

// Chave é a decomposição de uma chave de acesso em seus campos.
type Chave struct {
	// CUF é o código do IBGE da UF do emitente.
	CUF int
	// Ano são os dois últimos dígitos do ano de emissão.
	Ano int
	// Mes é o mês de emissão, de 1 a 12.
	Mes int
	// CNPJ é o CNPJ do emitente com 14 posições: as doze primeiras podem ter
	// letras maiúsculas (CNPJ alfanumérico), as duas últimas são dígitos.
	// Emitentes pessoa física usam o CPF alinhado à direita com zeros à
	// esquerda.
	CNPJ string
	// Modelo é o modelo do documento: 55 (NF-e), 65 (NFC-e), 57/67 (CT-e),
	// 58 (MDF-e).
	Modelo int
	// Serie é a série do documento, de 0 a 999.
	Serie int
	// Numero é o número do documento, de 1 a 999999999.
	Numero int
	// TpEmis é a forma de emissão: 1 normal, 4 EPEC, 9 offline (NFC-e) etc.
	TpEmis int
	// CNF é o código numérico de oito dígitos que compõe a chave.
	CNF string
}

// Nova monta uma chave de acesso a partir dos campos informados, calculando o
// dígito verificador. Devolve erro se algum campo estiver fora da faixa
// permitida pelo leiaute.
func Nova(c Chave) (string, error) {
	if err := c.validarCampos(); err != nil {
		return "", err
	}
	base := c.base43()
	return base + strconv.Itoa(DigitoVerificador(base)), nil
}

// MustNova é como [Nova], mas entra em pânico em caso de erro. Destina-se a
// literais em testes.
func MustNova(c Chave) string {
	s, err := Nova(c)
	if err != nil {
		panic(err)
	}
	return s
}

func (c Chave) base43() string {
	var b strings.Builder
	b.Grow(43)
	fmt.Fprintf(&b, "%02d", c.CUF)
	fmt.Fprintf(&b, "%02d%02d", c.Ano, c.Mes)
	b.WriteString(c.CNPJ)
	fmt.Fprintf(&b, "%02d", c.Modelo)
	fmt.Fprintf(&b, "%03d", c.Serie)
	fmt.Fprintf(&b, "%09d", c.Numero)
	fmt.Fprintf(&b, "%d", c.TpEmis)
	b.WriteString(c.CNF)
	return b.String()
}

func (c Chave) validarCampos() error {
	if _, err := uf.PorCodigo(c.CUF); err != nil {
		return fmt.Errorf("%w: %w", ErrChaveInvalida, err)
	}
	if c.Ano < 0 || c.Ano > 99 {
		return fmt.Errorf("%w: ano %d fora da faixa 00–99", ErrChaveInvalida, c.Ano)
	}
	if c.Mes < 1 || c.Mes > 12 {
		return fmt.Errorf("%w: mês %d fora da faixa 1–12", ErrChaveInvalida, c.Mes)
	}
	if !CNPJValido(c.CNPJ) {
		return fmt.Errorf("%w: CNPJ %q fora do padrão [A-Z0-9]{12}[0-9]{2}", ErrChaveInvalida, c.CNPJ)
	}
	if c.Modelo < 1 || c.Modelo > 99 {
		return fmt.Errorf("%w: modelo %d fora da faixa 01–99", ErrChaveInvalida, c.Modelo)
	}
	if c.Serie < 0 || c.Serie > 999 {
		return fmt.Errorf("%w: série %d fora da faixa 0–999", ErrChaveInvalida, c.Serie)
	}
	if c.Numero < 1 || c.Numero > 999999999 {
		return fmt.Errorf("%w: número %d fora da faixa 1–999999999", ErrChaveInvalida, c.Numero)
	}
	if c.TpEmis < 1 || c.TpEmis > 9 {
		return fmt.Errorf("%w: tpEmis %d fora da faixa 1–9", ErrChaveInvalida, c.TpEmis)
	}
	if len(c.CNF) != 8 || !apenasDigitos(c.CNF) {
		return fmt.Errorf("%w: cNF %q deve ter 8 dígitos", ErrChaveInvalida, c.CNF)
	}
	// Regra de validação B03-10 do MOC: o código numérico não pode ser igual
	// ao número do documento.
	if n, err := strconv.Atoi(c.CNF); err == nil && n == c.Numero {
		return fmt.Errorf("%w: cNF não pode ser igual ao nNF (%d)", ErrChaveInvalida, c.Numero)
	}
	return nil
}

// Parse decompõe uma chave de acesso de 44 posições, conferindo o dígito
// verificador. Aceita pontuação e espaços, que são descartados, e o prefixo
// "NFe" usado no atributo Id do XML.
func Parse(s string) (Chave, error) {
	limpa := Limpar(s)
	if err := Validar(limpa); err != nil {
		return Chave{}, err
	}
	atoi := func(inicio, fim int) int {
		n, _ := strconv.Atoi(limpa[inicio:fim])
		return n
	}
	return Chave{
		CUF:    atoi(0, 2),
		Ano:    atoi(2, 4),
		Mes:    atoi(4, 6),
		CNPJ:   limpa[6:20],
		Modelo: atoi(20, 22),
		Serie:  atoi(22, 25),
		Numero: atoi(25, 34),
		TpEmis: atoi(34, 35),
		CNF:    limpa[35:43],
	}, nil
}

// Validar confere o comprimento, o padrão (letras só nas posições do CNPJ) e o
// dígito verificador da chave.
func Validar(s string) error {
	limpa := Limpar(s)
	if len(limpa) != Tamanho {
		return fmt.Errorf("%w: %d caracteres, esperados %d", ErrChaveInvalida, len(limpa), Tamanho)
	}
	if !apenasDigitos(limpa[:6]) || !CNPJValido(limpa[6:20]) || !apenasDigitos(limpa[20:]) {
		return fmt.Errorf("%w: fora do padrão [0-9]{6}[A-Z0-9]{12}[0-9]{26}", ErrChaveInvalida)
	}
	esperado := DigitoVerificador(limpa[:43])
	informado := int(limpa[43] - '0')
	if esperado != informado {
		return fmt.Errorf("%w: dígito verificador %d, esperado %d", ErrChaveInvalida, informado, esperado)
	}
	return nil
}

// Valida é a forma booleana de [Validar].
func Valida(s string) bool { return Validar(s) == nil }

// prefixosDeId são os prefixos do atributo Id que antecedem a chave no XML.
var prefixosDeId = []string{"NFE", "CTE", "MDFE", "NFCE", "NF3E", "NFCOM", "BPE", "GTVE"}

// Limpar devolve a chave em maiúsculas, só com letras e dígitos, permitindo
// recebê-la formatada em grupos, copiada do DANFE, com o prefixo do atributo Id
// ("NFe", "CTe", "MDFe") ou com um rótulo de letras antes ("Chave: …"), que são
// descartados. As letras do CNPJ alfanumérico são preservadas.
func Limpar(s string) string {
	var b strings.Builder
	b.Grow(len(s))
	for _, r := range strings.ToUpper(s) {
		if (r >= '0' && r <= '9') || (r >= 'A' && r <= 'Z') {
			b.WriteRune(r)
		}
	}
	limpa := b.String()
	if len(limpa) > Tamanho {
		for _, prefixo := range prefixosDeId {
			if len(limpa) == Tamanho+len(prefixo) && strings.HasPrefix(limpa, prefixo) {
				return limpa[len(prefixo):]
			}
		}
		// Um rótulo antes da chave ("Chave:", "chNFe="): a chave começa por dígitos, então as letras do início caem.
		if i := strings.IndexFunc(limpa, func(r rune) bool { return r >= '0' && r <= '9' }); i > 0 && len(limpa)-i == Tamanho {
			return limpa[i:]
		}
	}
	return limpa
}

// CNPJValido confere o padrão do CNPJ na chave e nos leiautes (TCnpj):
// [A-Z0-9]{12}[0-9]{2}. Não confere os dígitos verificadores; para isso, use o
// pacote validacao.
func CNPJValido(cnpj string) bool {
	if len(cnpj) != 14 || !apenasDigitos(cnpj[12:]) {
		return false
	}
	for i := 0; i < 12; i++ {
		if (cnpj[i] < '0' || cnpj[i] > '9') && (cnpj[i] < 'A' || cnpj[i] > 'Z') {
			return false
		}
	}
	return true
}

// DigitoVerificador calcula o dígito verificador da chave pelo módulo 11 com
// pesos de 2 a 9 aplicados da direita para a esquerda, conforme o MOC. Espera
// os 43 primeiros caracteres da chave. Letras maiúsculas valem o código ASCII
// menos 48 (NT 2025.001); qualquer outro caractere vale zero.
func DigitoVerificador(base string) int {
	soma, peso := 0, 2
	for i := len(base) - 1; i >= 0; i-- {
		// Valor do caractere pela NT 2025.001: código ASCII menos 48. Dígitos
		// valem o que já valiam; A vale 17, Z vale 42.
		d := int(base[i]) - '0'
		if (base[i] < '0' || base[i] > '9') && (base[i] < 'A' || base[i] > 'Z') {
			d = 0
		}
		soma += d * peso
		if peso++; peso > 9 {
			peso = 2
		}
	}
	resto := soma % 11
	if resto == 0 || resto == 1 {
		return 0
	}
	return 11 - resto
}

// Formatar apresenta a chave em onze grupos de quatro dígitos separados por
// espaço, como impresso no DANFE.
func Formatar(s string) string {
	limpa := Limpar(s)
	if len(limpa) != Tamanho {
		return limpa
	}
	var b strings.Builder
	b.Grow(Tamanho + 10)
	for i := 0; i < Tamanho; i += 4 {
		if i > 0 {
			b.WriteByte(' ')
		}
		b.WriteString(limpa[i : i+4])
	}
	return b.String()
}

// GerarCodigoNumerico sorteia um código numérico de oito dígitos para compor a
// chave, garantindo que ele seja diferente do número do documento, como exige a
// regra de validação B03-10.
//
// O sorteio usa [crypto/rand]: dois documentos emitidos em sequência não
// recebem códigos previsíveis, o que dificulta a adivinhação de chaves de
// acesso alheias.
func GerarCodigoNumerico(numeroDocumento int) (string, error) {
	for range 16 {
		n, err := rand.Int(rand.Reader, big.NewInt(100000000))
		if err != nil {
			return "", fmt.Errorf("chave: não foi possível sortear o código numérico: %w", err)
		}
		v := int(n.Int64())
		if v != numeroDocumento {
			return fmt.Sprintf("%08d", v), nil
		}
	}
	return "", errors.New("chave: não foi possível sortear um código numérico diferente do número do documento")
}

func apenasDigitos(s string) bool {
	for i := 0; i < len(s); i++ {
		if s[i] < '0' || s[i] > '9' {
			return false
		}
	}
	return true
}
