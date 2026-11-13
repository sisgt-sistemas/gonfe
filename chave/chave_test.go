package chave_test

import (
	"math/rand/v2"
	"strconv"
	"strings"
	"testing"

	"github.com/sisgt-sistemas/gonfe/chave"
)

// chaveExemplo é montada campo a campo e tem o dígito verificador conferido à
// mão no teste abaixo. Serve de base para os demais testes deste arquivo.
const chaveExemplo = "35260312345678000199550010000001231000099997"

func TestDigitoVerificadorVetorConferidoAMao(t *testing.T) {
	// Composição: cUF=35 AAMM=2603 CNPJ=12345678000199 mod=55 série=001
	//             nNF=000000123 tpEmis=1 cNF=00009999
	base := chaveExemplo[:43]
	const esperado = "3526031234567800019955001000000123100009999"
	if base != esperado {
		t.Fatalf("base do exemplo = %s, queria %s", base, esperado)
	}

	// Módulo 11 com pesos 2..9 da direita para a esquerda: a soma ponderada
	// dos 43 dígitos é 653. 653 = 11×59 + 4, logo o dígito é 11 − 4 = 7.
	if soma := somaPonderada(base); soma != 653 {
		t.Fatalf("soma ponderada = %d, queria 653", soma)
	}
	if dv := chave.DigitoVerificador(base); dv != 7 {
		t.Errorf("DigitoVerificador = %d, queria 7", dv)
	}
	if err := chave.Validar(chaveExemplo); err != nil {
		t.Errorf("Validar(%s): %v", chaveExemplo, err)
	}
}

func TestDigitoVerificadorCasosDeBorda(t *testing.T) {
	casos := []struct {
		base string
		dv   int
		nota string
	}{
		{strings.Repeat("0", 43), 0, "soma zero, resto 0 → dígito 0"},
		{strings.Repeat("0", 42) + "6", 0, "soma 12, resto 1 → dígito 0, não 10"},
		{strings.Repeat("0", 42) + "1", 9, "soma 2, resto 2 → dígito 9"},
		{"1" + strings.Repeat("0", 42), 7, "dígito mais à esquerda tem peso 4"},
		{strings.Repeat("9", 43), 7, "soma 2061, resto 4 → dígito 7"},
	}
	for _, c := range casos {
		if dv := chave.DigitoVerificador(c.base); dv != c.dv {
			t.Errorf("%s: DigitoVerificador = %d, queria %d", c.nota, dv, c.dv)
		}
	}
}

// somaPonderada reimplementa a soma do módulo 11 de forma independente da
// biblioteca, indexando da esquerda para a direita, para servir de contraprova.
func somaPonderada(base string) int {
	soma := 0
	n := len(base)
	for i, r := range base {
		posicaoDaDireita := n - i // 1 para o último dígito
		peso := 2 + (posicaoDaDireita-1)%8
		soma += int(r-'0') * peso
	}
	return soma
}

func TestDigitoVerificadorBateComImplementacaoIndependente(t *testing.T) {
	r := rand.New(rand.NewPCG(1, 2))
	for range 5000 {
		var b strings.Builder
		for range 43 {
			b.WriteByte(byte('0' + r.IntN(10)))
		}
		base := b.String()

		esperado := 11 - somaPonderada(base)%11
		if esperado >= 10 {
			esperado = 0
		}
		if dv := chave.DigitoVerificador(base); dv != esperado {
			t.Fatalf("base %s: DigitoVerificador = %d, contraprova = %d", base, dv, esperado)
		}
	}
}

func TestNovaMontaChaveCompleta(t *testing.T) {
	c := chave.Chave{
		CUF:    43, // RS
		Ano:    26,
		Mes:    3,
		CNPJ:   "12345678000199",
		Modelo: 55,
		Serie:  1,
		Numero: 1234,
		TpEmis: 1,
		CNF:    "87654321",
	}
	s, err := chave.Nova(c)
	if err != nil {
		t.Fatalf("Nova: %v", err)
	}
	if len(s) != chave.Tamanho {
		t.Fatalf("chave tem %d dígitos: %s", len(s), s)
	}
	const esperadoSemDV = "4326031234567800019955001000001234187654321"
	if s[:43] != esperadoSemDV {
		t.Errorf("base = %s, queria %s", s[:43], esperadoSemDV)
	}
	if err := chave.Validar(s); err != nil {
		t.Errorf("a chave montada não valida: %v", err)
	}

	volta, err := chave.Parse(s)
	if err != nil {
		t.Fatalf("Parse: %v", err)
	}
	if volta != c {
		t.Errorf("ida e volta = %+v, queria %+v", volta, c)
	}
}

func TestNovaIdaEVoltaEmTodasAsUFs(t *testing.T) {
	for codigo := range 100 {
		c := chave.Chave{
			CUF: codigo, Ano: 26, Mes: 12, CNPJ: "99999999000191",
			Modelo: 65, Serie: 999, Numero: 999999999, TpEmis: 9, CNF: "00000001",
		}
		s, err := chave.Nova(c)
		if err != nil {
			continue // código de UF inexistente: rejeitado, como esperado
		}
		volta, err := chave.Parse(s)
		if err != nil {
			t.Errorf("cUF %d: Parse(%s): %v", codigo, s, err)
			continue
		}
		if volta != c {
			t.Errorf("cUF %d: ida e volta = %+v", codigo, volta)
		}
	}
}

func TestNovaRejeitaCamposForaDaFaixa(t *testing.T) {
	valida := chave.Chave{
		CUF: 35, Ano: 26, Mes: 3, CNPJ: "12345678000199",
		Modelo: 55, Serie: 1, Numero: 1234, TpEmis: 1, CNF: "87654321",
	}
	casos := map[string]func(*chave.Chave){
		"cUF inexistente":     func(c *chave.Chave) { c.CUF = 99 },
		"ano negativo":        func(c *chave.Chave) { c.Ano = -1 },
		"mês zero":            func(c *chave.Chave) { c.Mes = 0 },
		"mês treze":           func(c *chave.Chave) { c.Mes = 13 },
		"CNPJ curto":          func(c *chave.Chave) { c.CNPJ = "1234" },
		"CNPJ não numérico":   func(c *chave.Chave) { c.CNPJ = "1234567800019X" },
		"série negativa":      func(c *chave.Chave) { c.Serie = -1 },
		"série acima de 999":  func(c *chave.Chave) { c.Serie = 1000 },
		"número zero":         func(c *chave.Chave) { c.Numero = 0 },
		"número muito grande": func(c *chave.Chave) { c.Numero = 1000000000 },
		"tpEmis zero":         func(c *chave.Chave) { c.TpEmis = 0 },
		"cNF com 7 dígitos":   func(c *chave.Chave) { c.CNF = "1234567" },
		"cNF igual ao nNF":    func(c *chave.Chave) { c.CNF = "00001234" },
	}
	for nome, quebrar := range casos {
		c := valida
		quebrar(&c)
		if s, err := chave.Nova(c); err == nil {
			t.Errorf("%s: Nova devolveu %s, queria erro", nome, s)
		}
	}
}

func TestValidarRejeitaChavesRuins(t *testing.T) {
	dvErrado := chaveExemplo[:43] + strconv.Itoa((int(chaveExemplo[43]-'0')+1)%10)
	casos := map[string]string{
		"vazia":        "",
		"curta":        chaveExemplo[:43],
		"longa":        chaveExemplo + "0",
		"não numérica": chaveExemplo[:43] + "X",
		"DV trocado":   dvErrado,
		"só pontuação": "....",
	}
	for nome, s := range casos {
		if err := chave.Validar(s); err == nil {
			t.Errorf("%s: Validar(%q) deveria falhar", nome, s)
		}
		if chave.Valida(s) {
			t.Errorf("%s: Valida(%q) deveria ser falso", nome, s)
		}
	}
}

func TestParseAceitaPrefixoEFormatacao(t *testing.T) {
	entradas := []string{
		chaveExemplo,
		"NFe" + chaveExemplo,
		chave.Formatar(chaveExemplo),
		"  " + chave.Formatar(chaveExemplo) + "  ",
	}
	for _, e := range entradas {
		c, err := chave.Parse(e)
		if err != nil {
			t.Errorf("Parse(%q): %v", e, err)
			continue
		}
		if c.CUF != 35 || c.Ano != 26 || c.Mes != 3 || c.Modelo != 55 ||
			c.Serie != 1 || c.Numero != 123 || c.TpEmis != 1 ||
			c.CNPJ != "12345678000199" || c.CNF != "00009999" {
			t.Errorf("Parse(%q) = %+v", e, c)
		}
	}
}

func TestFormatar(t *testing.T) {
	got := chave.Formatar(chaveExemplo)
	esperado := "3526 0312 3456 7800 0199 5500 1000 0001 2310 0009 9997"
	if got != esperado {
		t.Errorf("Formatar = %q, queria %q", got, esperado)
	}
	if chave.Limpar(got) != chaveExemplo {
		t.Error("a formatação alterou os dígitos")
	}
	if chave.Formatar("123") != "123" {
		t.Error("chave incompleta deveria ser devolvida sem formatação")
	}
}

func TestGerarCodigoNumerico(t *testing.T) {
	vistos := make(map[string]bool)
	for range 200 {
		c, err := chave.GerarCodigoNumerico(42)
		if err != nil {
			t.Fatalf("GerarCodigoNumerico: %v", err)
		}
		if len(c) != 8 {
			t.Fatalf("código %q não tem 8 dígitos", c)
		}
		if c == "00000042" {
			t.Fatal("o código não pode ser igual ao número do documento")
		}
		vistos[c] = true
	}
	if len(vistos) < 190 {
		t.Errorf("apenas %d códigos distintos em 200 sorteios: entropia insuficiente", len(vistos))
	}
}

func TestMustNovaEntraEmPanico(t *testing.T) {
	defer func() {
		if recover() == nil {
			t.Error("queria pânico com chave inválida")
		}
	}()
	chave.MustNova(chave.Chave{})
}

// dvIndependente refaz o dígito da chave pela descrição da NT 2025.001, sem usar o pacote: valor ASCII menos 48,
// pesos de 2 a 9 da direita para a esquerda, resto 0 ou 1 dá zero.
func dvIndependente(base string) int {
	soma := 0
	for i := range len(base) {
		peso := 2 + (len(base)-1-i)%8
		soma += (int(base[i]) - 48) * peso
	}
	if r := soma % 11; r > 1 {
		return 11 - r
	}
	return 0
}

func TestChaveComCNPJAlfanumerico(t *testing.T) {
	// O CNPJ do exemplo da NT 2025.001: 12.ABC.345/01DE-35.
	const cnpj = "12ABC34501DE35"
	c := chave.Chave{CUF: 43, Ano: 26, Mes: 7, CNPJ: cnpj, Modelo: 57, Serie: 1, Numero: 1234, TpEmis: 1, CNF: "11223344"}
	s, err := chave.Nova(c)
	if err != nil {
		t.Fatalf("Nova: %v", err)
	}
	if s[6:20] != cnpj {
		t.Errorf("CNPJ na chave = %s, queria %s", s[6:20], cnpj)
	}
	if got, want := int(s[43]-'0'), dvIndependente(s[:43]); got != want {
		t.Errorf("dígito = %d, a contraprova dá %d", got, want)
	}
	if err := chave.Validar(s); err != nil {
		t.Errorf("Validar: %v", err)
	}
	volta, err := chave.Parse("CTe" + s)
	if err != nil || volta != c {
		t.Errorf("Parse(CTe…) = %+v, %v; queria %+v", volta, err, c)
	}
	if got := chave.Limpar(strings.ToLower(chave.Formatar(s))); got != s {
		t.Errorf("Limpar(formatada, minúsculas) = %s, queria %s", got, s)
	}
}

func TestDigitoDaChaveNumericaNaoMuda(t *testing.T) {
	// A regra nova não muda o dígito de chave só de dígitos.
	if got, want := chave.DigitoVerificador(chaveExemplo[:43]), dvIndependente(chaveExemplo[:43]); got != want {
		t.Errorf("dígito = %d, a contraprova dá %d", got, want)
	}
}

func TestChaveRecusaLetraForaDoCNPJ(t *testing.T) {
	s := chave.MustNova(chave.Chave{CUF: 43, Ano: 26, Mes: 7, CNPJ: "12ABC34501DE35", Modelo: 57, Serie: 1,
		Numero: 1234, TpEmis: 1, CNF: "11223344"})
	for _, ruim := range []string{
		"4A" + s[2:],          // letra na UF
		s[:18] + "A" + s[19:], // letra no dígito do CNPJ
		s[:30] + "A" + s[31:], // letra no número
	} {
		if chave.Valida(ruim) {
			t.Errorf("Validar aceitou %s", ruim)
		}
	}
	if _, err := chave.Nova(chave.Chave{CUF: 43, Ano: 26, Mes: 7, CNPJ: "12ABC34501DEAB", Modelo: 57, Serie: 1,
		Numero: 1, TpEmis: 1, CNF: "11223344"}); err == nil {
		t.Error("Nova aceitou CNPJ com letra nos dígitos verificadores")
	}
}

func TestLimparTiraRotuloAntesDaChave(t *testing.T) {
	s := chave.MustNova(chave.Chave{CUF: 43, Ano: 26, Mes: 7, CNPJ: "12ABC34501DE35", Modelo: 57, Serie: 1,
		Numero: 1234, TpEmis: 1, CNF: "11223344"})
	for _, entrada := range []string{"Chave: " + s, "chNFe=" + s, "chave de acesso " + chave.Formatar(s)} {
		if got := chave.Limpar(entrada); got != s {
			t.Errorf("Limpar(%q) = %s, queria %s", entrada, got, s)
		}
	}
}
