package ctesimp

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"encoding/xml"
	"errors"
	"fmt"
	"strings"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/internal/norm"
	"github.com/sisgt-sistemas/gonfe/tipos"
	"github.com/sisgt-sistemas/gonfe/uf"
	"github.com/sisgt-sistemas/gonfe/xmldsig"
)

// Novo devolve um CT-e Simplificado com os campos de estrutura preenchidos com
// os padrões do leiaute: versão 4.00, modelo 57, finalidade 5, modal
// rodoviário, serviço normal, emissão normal por aplicativo do contribuinte.
func Novo() *CTeSimp {
	return &CTeSimp{
		InfCte: InfCte{
			Versao: Versao,
			Ide: Ide{
				Mod:     cte.ModeloCTe,
				Modal:   cte.ModalRodoviario,
				TpEmis:  cte.EmissaoNormal,
				ProcEmi: cte.EmissaoAplicativoContribuinte,
				VerProc: "gonfe",
				TpCTe:   Simplificado,
				TpImp:   cte.Retrato,
				TpServ:  cte.ServicoNormal,
				Retira:  cte.RetiraNao,
			},
			InfModal: cte.InfModal{VersaoModal: VersaoModal},
		},
	}
}

// Chave devolve a chave de acesso de 44 dígitos, extraída do atributo Id.
func (c *CTeSimp) Chave() string { return strings.TrimPrefix(c.InfCte.Id, "CTe") }

// OpcoesPreparo ajusta o comportamento de [CTeSimp.Preparar].
type OpcoesPreparo struct {
	// SemCalculoDeTotais preserva os valores de cada det e o grupo total como
	// preenchidos pelo chamador.
	SemCalculoDeTotais bool
}

// Preparar deixa o conhecimento pronto para serialização: normaliza os campos
// de texto, numera os itens sem número, calcula os valores a partir dos
// componentes e monta a chave de acesso.
//
// Preparar é idempotente.
func (c *CTeSimp) Preparar(opcoes ...OpcoesPreparo) error {
	opc := OpcoesPreparo{}
	if len(opcoes) > 0 {
		opc = opcoes[0]
	}

	c.InfCte.Versao = Versao
	if c.InfCte.InfModal.VersaoModal == "" {
		c.InfCte.InfModal.VersaoModal = VersaoModal
	}
	for i := range c.InfCte.Det {
		if c.InfCte.Det[i].NItem == 0 {
			c.InfCte.Det[i].NItem = i + 1
		}
	}
	norm.Normalizar(c)

	if !opc.SemCalculoDeTotais {
		c.CalcularTotais()
	}
	norm.Normalizar(c)

	return c.gerarChave()
}

// CalcularTotais soma os componentes de cada prestação no vPrest dela (e no
// vRec, se vazio) e as prestações no grupo total.
func (c *CTeSimp) CalcularTotais() {
	totalPrest := tipos.NovoDecimal(0, 2)
	totalRec := tipos.NovoDecimal(0, 2)
	for i := range c.InfCte.Det {
		det := &c.InfCte.Det[i]
		if len(det.Comp) > 0 {
			soma := tipos.NovoDecimal(0, 2)
			for _, comp := range det.Comp {
				soma = soma.Somar(comp.VComp)
			}
			det.VPrest = soma.ComCasas(2)
		}
		if det.VRec.EhZero() {
			det.VRec = det.VPrest.ComCasas(2)
		}
		totalPrest = totalPrest.Somar(det.VPrest)
		totalRec = totalRec.Somar(det.VRec)
	}
	c.InfCte.Total.VTPrest = totalPrest.ComCasas(2)
	c.InfCte.Total.VTRec = totalRec.ComCasas(2)
}

func (c *CTeSimp) gerarChave() error {
	ide := &c.InfCte.Ide

	documento := c.InfCte.Emit.CNPJ
	if documento == "" {
		documento = c.InfCte.Emit.CPF
	}
	if documento == "" {
		return fmt.Errorf("ctesimp: o emitente precisa de CNPJ ou CPF para compor a chave de acesso")
	}
	if len(documento) < 14 {
		documento = strings.Repeat("0", 14-len(documento)) + documento
	}

	if ide.DhEmi.Vazia() {
		return fmt.Errorf("ctesimp: dhEmi precisa estar preenchida para compor a chave de acesso")
	}
	ano, mes := tipos.AnoMes(ide.DhEmi.Time)

	if ide.CCT == "" {
		codigo, err := chave.GerarCodigoNumerico(ide.NCT)
		if err != nil {
			return err
		}
		ide.CCT = codigo
	}

	unidade, err := uf.PorSigla(c.InfCte.Emit.EnderEmit.UF)
	if err != nil {
		return fmt.Errorf("ctesimp: UF do emitente: %w", err)
	}
	if ide.CUF == 0 {
		ide.CUF = unidade.Codigo()
	}

	completa, err := chave.Nova(chave.Chave{
		CUF:    ide.CUF,
		Ano:    ano,
		Mes:    mes,
		CNPJ:   documento,
		Modelo: ide.Mod.Numero(),
		Serie:  ide.Serie,
		Numero: ide.NCT,
		TpEmis: ide.TpEmis.Numero(),
		CNF:    ide.CCT,
	})
	if err != nil {
		return err
	}

	ide.CDV = int(completa[43] - '0')
	c.InfCte.Id = "CTe" + completa
	return nil
}

// XML serializa o conhecimento, sem declaração XML e sem espaços supérfluos.
func (c *CTeSimp) XML() ([]byte, error) {
	dados, err := xml.Marshal(c)
	if err != nil {
		return nil, fmt.Errorf("ctesimp: falha ao serializar: %w", err)
	}
	return dados, nil
}

// AssinarCom prepara, serializa e assina o conhecimento em uma única chamada. A
// assinatura cobre o grupo infCte, como no CT-e.
func (c *CTeSimp) AssinarCom(assinante xmldsig.Assinante) ([]byte, error) {
	if err := c.Preparar(); err != nil {
		return nil, err
	}
	documento, err := c.XML()
	if err != nil {
		return nil, err
	}
	return xmldsig.Assinar(documento, "infCte", assinante)
}

// Ler interpreta o XML de um CT-e Simplificado. Aceita o elemento <CTeSimp>
// isolado ou dentro de um <cteSimpProc>.
func Ler(dados []byte) (*CTeSimp, error) {
	recorte, err := recortar(dados, "CTeSimp")
	if err != nil {
		return nil, err
	}
	var c CTeSimp
	if err := xml.Unmarshal(recorte, &c); err != nil {
		return nil, fmt.Errorf("ctesimp: falha ao interpretar o XML: %w", err)
	}
	return &c, nil
}

// LerCTeSimpProc separa o conhecimento e o protocolo de um arquivo de
// distribuição.
func LerCTeSimpProc(dados []byte) (*CTeSimp, *cte.ProtCTe, error) {
	c, err := Ler(dados)
	if err != nil {
		return nil, nil, err
	}
	recorte, err := recortar(dados, "protCTe")
	if errors.Is(err, errAusente) {
		return c, nil, nil // Sem protocolo: o documento ainda não foi autorizado.
	}
	if err != nil {
		return c, nil, err
	}
	var prot cte.ProtCTe
	if err := xml.Unmarshal(recorte, &prot); err != nil {
		return c, nil, fmt.Errorf("ctesimp: falha ao interpretar o protocolo: %w", err)
	}
	return c, &prot, nil
}

// MontarCTeSimpProc envelopa o conhecimento assinado com o protocolo de
// autorização (cteSimpProc), preservando os bytes assinados.
func MontarCTeSimpProc(assinado []byte, prot *cte.ProtCTe) ([]byte, error) {
	if prot == nil {
		return nil, fmt.Errorf("ctesimp: protocolo ausente")
	}
	recorte, err := recortar(assinado, "CTeSimp")
	if err != nil {
		return nil, err
	}
	if prot.Versao == "" {
		prot.Versao = Versao
	}
	protXML, err := xml.Marshal(prot)
	if err != nil {
		return nil, fmt.Errorf("ctesimp: falha ao serializar o protocolo: %w", err)
	}

	var b bytes.Buffer
	b.WriteString(`<cteSimpProc xmlns="` + Espaco + `" versao="` + Versao + `">`)
	b.Write(recorte)
	b.Write(protXML)
	b.WriteString(`</cteSimpProc>`)
	return b.Bytes(), nil
}

// MontarEnvioSincrono prepara a mensagem de envio no formato do CT-e: o
// documento comprimido em gzip e codificado em base64, sem alterar os bytes
// assinados. O serviço que a recebe é o do CT-e Simplificado, não o do CT-e.
func MontarEnvioSincrono(assinado []byte) ([]byte, error) {
	recorte, err := recortar(assinado, "CTeSimp")
	if err != nil {
		return nil, err
	}
	var comprimido bytes.Buffer
	w := gzip.NewWriter(&comprimido)
	if _, err := w.Write(recorte); err != nil {
		return nil, fmt.Errorf("ctesimp: falha ao comprimir: %w", err)
	}
	if err := w.Close(); err != nil {
		return nil, fmt.Errorf("ctesimp: falha ao comprimir: %w", err)
	}
	return []byte(base64.StdEncoding.EncodeToString(comprimido.Bytes())), nil
}

// XMLDeclarado antepõe a declaração XML em UTF-8 ao documento.
func XMLDeclarado(documento []byte) []byte { return cte.XMLDeclarado(documento) }

// errAusente é o elemento que o documento não traz; elemento aberto e não fechado é outro erro.
var errAusente = errors.New("elemento ausente")

// recortar isola o primeiro elemento com o nome informado, preservando os bytes
// originais. Os marcadores incluem o caractere que segue o nome, então "<CTe "
// nunca casa com "<CTeSimp ".
func recortar(dados []byte, nome string) ([]byte, error) {
	s := string(dados)
	abertura := -1
	for _, marcador := range []string{"<" + nome + " ", "<" + nome + ">"} {
		if i := strings.Index(s, marcador); i >= 0 && (abertura < 0 || i < abertura) {
			abertura = i
		}
	}
	if abertura < 0 {
		return nil, fmt.Errorf("ctesimp: o documento não contém um elemento <%s>: %w", nome, errAusente)
	}
	fechamento := "</" + nome + ">"
	fim := strings.LastIndex(s, fechamento)
	if fim < abertura {
		return nil, fmt.Errorf("ctesimp: o elemento <%s> não está fechado", nome)
	}
	return dados[abertura : fim+len(fechamento)], nil
}
