package danfe_test

import (
	"bytes"
	"testing"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/ctesimp"
	"github.com/sisgt-sistemas/gonfe/danfe"
	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/tipos"
)

func chaveNFeSimp(numero int) string {
	return chave.MustNova(chave.Chave{CUF: 43, Ano: 26, Mes: 3, CNPJ: "11222333000181", Modelo: 55, Serie: 1,
		Numero: numero, TpEmis: 1, CNF: "12345678"})
}

// simplificadoExemplo monta um CT-e Simplificado com as entregas dadas; cada uma leva duas NF-e.
func simplificadoExemplo(entregas int) *ctesimp.CTeSimp {
	c := ctesimp.Novo()
	ide := &c.InfCte.Ide
	ide.CFOP, ide.NatOp = "5353", "PRESTACAO DE SERVICO DE TRANSPORTE"
	ide.Serie, ide.NCT, ide.CCT = 1, 1234, "11223344"
	ide.DhEmi = tipos.DH("2026-03-04T08:00:00-03:00")
	ide.TpAmb = cte.Homologacao
	ide.CMunEnv, ide.XMunEnv, ide.UFEnv = 4314902, "PORTO ALEGRE", "RS"
	ide.UFIni, ide.UFFim = "RS", "RS"
	c.InfCte.Emit = ctesimp.Emit{
		CNPJ: cnpjExemplo, IE: "0961234567", XNome: "TRANSPORTES EXEMPLO LTDA", CRT: "3",
		EnderEmit: cte.EnderecoEmitente{XLgr: "AVENIDA DAS INDUSTRIAS", Nro: "2000", XBairro: "DISTRITO INDUSTRIAL",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "91150000", UF: "RS", Fone: "5133330000"},
	}
	c.InfCte.Toma = ctesimp.Toma{
		Toma: cte.TomadorRemetente, IndIEToma: cte.ContribuinteICMS,
		CNPJ: "11222333000181", IE: "1234567890", XNome: "INDUSTRIA REMETENTE SA",
		EnderToma: cte.Endereco{XLgr: "RUA DA FABRICA", Nro: "500", XBairro: "INDUSTRIAL", CMun: 4314902,
			XMun: "PORTO ALEGRE", CEP: "90000000", UF: "RS", XPais: "BRASIL"},
	}
	c.InfCte.InfCarga = cte.InfCarga{VCarga: tipos.Ptr(tipos.D("85000.00")), ProPred: "PECAS AUTOMOTIVAS",
		InfQ: []cte.InfQ{{CUnid: cte.UnidadeKG, TpMed: "PESO BRUTO", QCarga: tipos.D("4200")}}}
	destinos := []string{"CAXIAS DO SUL", "SANTA MARIA", "PELOTAS", "NOVO HAMBURGO"}
	for i := range entregas {
		c.InfCte.Det = append(c.InfCte.Det, ctesimp.Det{
			CMunIni: 4314902, XMunIni: "PORTO ALEGRE", CMunFim: 4305108, XMunFim: destinos[i%len(destinos)],
			Comp:   []cte.Componente{{XNome: "FRETE PESO", VComp: tipos.D("500.00")}},
			InfNFe: []ctesimp.InfNFe{{ChNFe: chaveNFeSimp(100 + 2*i)}, {ChNFe: chaveNFeSimp(101 + 2*i)}},
		})
	}
	c.InfCte.InfModal.Rodo = &cte.Rodo{RNTRC: "12345678"}
	c.InfCte.Imp.ICMS.ICMS00 = &cte.ICMS00{CST: "00", VBC: tipos.D("1000.00"), PICMS: tipos.D("12.00"), VICMS: tipos.D("120.00")}
	c.InfCTeSupl = &cte.InfCTeSupl{QrCodCTe: "https://dfe-portal.svrs.rs.gov.br/cte/qrCode?chCTe=43260312345678000195570010000012341112233442&tpAmb=2"}
	return c
}

func procSimpDe(t *testing.T, c *ctesimp.CTeSimp) []byte {
	t.Helper()
	assinado, err := c.AssinarCom(certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjExemplo}))
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	prot := &cte.ProtCTe{InfProt: cte.InfProt{ChCTe: c.Chave(), NProt: "143260000012345", CStat: 100,
		DhRecbto: tipos.DH("2026-03-04T08:00:30-03:00")}}
	proc, err := ctesimp.MontarCTeSimpProc(assinado, prot)
	if err != nil {
		t.Fatal(err)
	}
	return proc
}

// O DACTE do Simplificado traz o cabeçalho do manual e a tabela das entregas, com os documentos de cada uma.
func TestDACTESimp(t *testing.T) {
	dados, err := danfe.GerarDACTESimp(procSimpDe(t, simplificadoExemplo(2)), danfe.Opcoes{})
	if err != nil {
		t.Fatal(err)
	}
	for _, trecho := range []string{"DACTE", "Simplificado", "5 - SIMPLIFICADO", "ENTREGAS", "CAXIAS DO SUL",
		"SANTA MARIA", "INDUSTRIA REMETENTE SA", "EMITIDO EM AMBIENTE DE HOMOLOGA", "1.000,00"} {
		if !bytes.Contains(dados, []byte(trecho)) {
			t.Errorf("falta %q", trecho)
		}
	}
	// Quatro NF-e, duas por entrega.
	if n := bytes.Count(dados, []byte("NF-e")); n < 4 {
		t.Errorf("%d NF-e na tabela, queria 4", n)
	}
}

func TestDACTESimpPaginaAsEntregas(t *testing.T) {
	dados, err := danfe.GerarDACTESimp(procSimpDe(t, simplificadoExemplo(60)), danfe.Opcoes{})
	if err != nil {
		t.Fatal(err)
	}
	n := paginasDe(dados)
	if n < 2 {
		t.Fatalf("120 documentos em %d folha(s); a tabela deveria continuar", n)
	}
	conferirFolhas(t, "Simplificado com 60 entregas", dados)
	// A última entrega aparece: nada se perdeu entre as folhas.
	if !bytes.Contains(dados, []byte(chaveFormatadaDe(chaveNFeSimp(101+2*59)))) {
		t.Error("a última NF-e da última entrega não aparece")
	}
}

func chaveFormatadaDe(c string) string { return chave.Formatar(c) }
