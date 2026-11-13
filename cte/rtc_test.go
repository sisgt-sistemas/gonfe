package cte_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/internal/rtctest"
	"github.com/sisgt-sistemas/gonfe/tipos"
	"github.com/sisgt-sistemas/gonfe/xsd"
)

// reformaExemplo é o CT-e de exemplo com todos os grupos da reforma tributária que o esquema prevê para ele.
func reformaExemplo() *cte.CTe {
	c := conhecimentoExemplo()
	ide := &c.InfCte.Ide
	ide.GCompraGov = rtctest.CompraGov()
	ide.TpPagAnt = "1"
	ide.GPagAntecipado = &cte.PagAntecipado{ChDFePagAnt: rtctest.ChaveDFe(601)}
	ide.GMultaJuros = &cte.MultaJuros{ChCTeRef: rtctest.ChaveDFe(602)}
	c.InfCte.VPrest.VTPrestLiq = tipos.Ptr(tipos.D("2475.00"))
	c.InfCte.Imp.IBSCBS = rtctest.IBSCBS("2500.00")
	c.InfCte.Imp.VTotDFe = tipos.Ptr(tipos.D("2525.00"))
	c.InfCte.PgtoVinc = rtctest.PgtoVinc()
	c.InfCte.InfSolicNFF = rtctest.NFF()
	c.InfCte.InfPAA = rtctest.PAA()
	return c
}

// Os grupos da reforma, na ordem e nas casas do esquema (NT 2026.004 RTC): o CT-e com todos eles passa no esquema
// oficial e no Validar.
func TestConhecimentoComReformaPassaNoEsquema(t *testing.T) {
	c := reformaExemplo()
	assinado, err := c.AssinarCom(certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora}))
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := c.Validar(); err != nil {
		t.Errorf("Validar recusou o exemplo com a reforma: %v", err)
	}
	for _, trecho := range []string{"<pIBSUF>0.1000</pIBSUF>", "<vIBS>2.50</vIBS>", "<vCBS>22.50</vCBS>", "<gCompraGov>"} {
		if !strings.Contains(string(assinado), trecho) {
			t.Errorf("o XML não traz %s", trecho)
		}
	}
	err = xsd.ValidarCTe(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("o CT-e com a reforma não passa no esquema: %v", err)
	}
}

func TestValidarReforma(t *testing.T) {
	casos := []struct {
		nome, campo string
		mexer       func(c *cte.CTe)
	}{
		{"vIBS diferente da soma", "imp.IBSCBS.gIBSCBS.vIBS", func(c *cte.CTe) { c.InfCte.Imp.IBSCBS.GIBSCBS.VIBS = tipos.D("9.99") }},
		{"CST com dois dígitos", "imp.IBSCBS.CST", func(c *cte.CTe) { c.InfCte.Imp.IBSCBS.CST = "01" }},
		{"classificação curta", "imp.IBSCBS.cClassTrib", func(c *cte.CTe) { c.InfCte.Imp.IBSCBS.CClassTrib = "123" }},
		{"doação 2", "imp.IBSCBS.indDoacao", func(c *cte.CTe) { c.InfCte.Imp.IBSCBS.IndDoacao = "2" }},
		{"ente 7", "ide.gCompraGov.tpEnteGov", func(c *cte.CTe) { c.InfCte.Ide.GCompraGov.TpEnteGov = "7" }},
		{"pagamento antecipado 2", "ide.tpPagAnt", func(c *cte.CTe) { c.InfCte.Ide.TpPagAnt = "2" }},
		{"chave do pagamento antecipado", "ide.gPagAntecipado.chDFePagAnt", func(c *cte.CTe) { c.InfCte.Ide.GPagAntecipado.ChDFePagAnt = "123" }},
		{"pagamentos vazios", "pgtoVinc.pgto", func(c *cte.CTe) { c.InfCte.PgtoVinc.Pgto = nil }},
		{"pagamento sem número", "pgtoVinc.pgto[0]@nPag", func(c *cte.CTe) { c.InfCte.PgtoVinc.Pgto[0].NPag = 0 }},
		{"raiz do PSP curta", "pgtoVinc.pgto[0].CNPJBasePSP", func(c *cte.CTe) { c.InfCte.PgtoVinc.Pgto[0].CNPJBasePSP = "1234" }},
		{"assinatura do PAA fora de base64", "infPAA.PAASignature.SignatureValue", func(c *cte.CTe) { c.InfCte.InfPAA.PAASignature.SignatureValue = "não é base64!" }},
		{"solicitação NFF curta", "infSolicNFF.xSolic", func(c *cte.CTe) { c.InfCte.InfSolicNFF.XSolic = "A" }},
		{"transação de um caractere", "pgtoVinc.pgto[0]@idTransacao", func(c *cte.CTe) { c.InfCte.PgtoVinc.Pgto[0].IdTransacao = "X" }},
		{"base negativa", "imp.IBSCBS.gIBSCBS.vBC", func(c *cte.CTe) { c.InfCte.Imp.IBSCBS.GIBSCBS.VBC = tipos.D("-1.00") }},
		{"alíquota de mil", "imp.IBSCBS.gIBSCBS.gCBS.pCBS", func(c *cte.CTe) { c.InfCte.Imp.IBSCBS.GIBSCBS.GCBS.PCBS = tipos.D("1000.0000") }},
		{"redutor de mil", "ide.gCompraGov.pRedutor", func(c *cte.CTe) { c.InfCte.Ide.GCompraGov.PRedutor = tipos.D("1000.0000") }},
		{"emissão 5", "ide.tpEmis", func(c *cte.CTe) { c.InfCte.Ide.TpEmis = cte.TipoEmissao("5") }},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			c := reformaExemplo()
			if err := c.Preparar(); err != nil {
				t.Fatal(err)
			}
			caso.mexer(c)
			var erros cte.Erros
			if err := c.Validar(); !errors.As(err, &erros) {
				t.Fatalf("Validar aceitou; queria erro em %s", caso.campo)
			}
			for _, e := range erros {
				if e.Campo == caso.campo {
					return
				}
			}
			t.Errorf("queria erro em %s; veio: %v", caso.campo, c.Validar())
		})
	}
}

// O 3 é o Regime Especial da Nota Fiscal Fácil, não contingência: não pede data nem justificativa. O 5 saiu do
// leiaute 4.00.
func TestFormasDeEmissao(t *testing.T) {
	if cte.EmissaoRegimeEspecialNFF.Contingencia() {
		t.Error("o Regime Especial NFF (3) foi tratado como contingência")
	}
	for _, valido := range []cte.TipoEmissao{"1", "2", "3", "4", "7", "8"} {
		if !valido.Valida() {
			t.Errorf("tpEmis %s recusado", valido)
		}
	}
	for _, invalido := range []cte.TipoEmissao{"", "5", "6", "9"} {
		if invalido.Valida() {
			t.Errorf("tpEmis %q aceito", invalido)
		}
	}
	c := conhecimentoExemplo()
	c.InfCte.Ide.TpEmis = cte.EmissaoRegimeEspecialNFF
	if err := c.Preparar(); err != nil {
		t.Fatal(err)
	}
	if err := c.Validar(); err != nil {
		t.Errorf("o CT-e no Regime Especial NFF foi recusado: %v", err)
	}
}
