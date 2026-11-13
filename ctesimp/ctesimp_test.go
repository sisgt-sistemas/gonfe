package ctesimp_test

import (
	"bytes"
	"compress/gzip"
	"encoding/base64"
	"errors"
	"io"
	"strings"
	"testing"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/ctesimp"
	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/internal/rtctest"
	"github.com/sisgt-sistemas/gonfe/tipos"
	"github.com/sisgt-sistemas/gonfe/xmldsig"
)

const cnpjTransportadora = "12345678000195"

// chaveNFe monta uma chave de NF-e válida (modelo 55) com o número dado.
func chaveNFe(numero int) string {
	return chave.MustNova(chave.Chave{CUF: 43, Ano: 26, Mes: 3, CNPJ: "11222333000181", Modelo: 55, Serie: 1,
		Numero: numero, TpEmis: 1, CNF: "12345678"})
}

// chaveCTe monta uma chave de CT-e válida (modelo 57) de outra transportadora.
func chaveCTe(numero int) string {
	return chave.MustNova(chave.Chave{CUF: 43, Ano: 26, Mes: 2, CNPJ: "99999999000191", Modelo: 57, Serie: 1,
		Numero: numero, TpEmis: 1, CNF: "87654321"})
}

// entregasExemplo monta um CT-e Simplificado com duas entregas do mesmo tomador: a primeira com uma NF-e num
// contêiner lacrado sobre um reboque, a segunda com duas NF-e e componentes do frete.
func entregasExemplo() *ctesimp.CTeSimp {
	c := ctesimp.Novo()

	ide := &c.InfCte.Ide
	ide.CFOP = "5353"
	ide.NatOp = "PRESTACAO DE SERVICO DE TRANSPORTE"
	ide.Serie = 1
	ide.NCT = 1234
	ide.CCT = "11223344"
	ide.DhEmi = tipos.DH("2026-03-04T08:00:00-03:00")
	ide.TpAmb = cte.Homologacao
	ide.CMunEnv, ide.XMunEnv, ide.UFEnv = 4314902, "PORTO ALEGRE", "RS"
	ide.UFIni, ide.UFFim = "RS", "RS"

	c.InfCte.Emit = ctesimp.Emit{
		CNPJ:  cnpjTransportadora,
		IE:    "0961234567",
		XNome: "TRANSPORTADORA EXEMPLO LTDA",
		CRT:   "3",
		EnderEmit: cte.EnderecoEmitente{
			XLgr: "AVENIDA DAS INDUSTRIAS", Nro: "2000", XBairro: "DISTRITO INDUSTRIAL",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "91150000", UF: "RS", Fone: "5133330000",
		},
	}
	c.InfCte.Toma = ctesimp.Toma{
		Toma: cte.TomadorRemetente, IndIEToma: cte.ContribuinteICMS,
		CNPJ: "11222333000181", IE: "1234567890", XNome: "INDUSTRIA REMETENTE SA",
		EnderToma: cte.Endereco{
			XLgr: "RUA DA FABRICA", Nro: "500", XBairro: "INDUSTRIAL",
			CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "90000000", UF: "RS",
		},
	}
	c.InfCte.InfCarga = cte.InfCarga{
		VCarga:  tipos.Ptr(tipos.D("85000.00")),
		ProPred: "PECAS AUTOMOTIVAS",
		InfQ: []cte.InfQ{
			{CUnid: cte.UnidadeKG, TpMed: "03", QCarga: tipos.D("4200.5")},
			{CUnid: cte.UnidadeM3, TpMed: "01", QCarga: tipos.D("18.75")},
		},
	}
	c.InfCte.Det = []ctesimp.Det{
		{
			CMunIni: 4314902, XMunIni: "PORTO ALEGRE", CMunFim: 4305108, XMunFim: "CAXIAS DO SUL",
			VPrest: tipos.D("1500.00"),
			InfNFe: []ctesimp.InfNFe{{
				ChNFe: chaveNFe(101),
				InfUnidTransp: []ctesimp.UnidadeTransporte{{
					TpUnidTransp: "2", IdUnidTransp: "ABC1D23",
					InfUnidCarga: []ctesimp.UnidadeCarga{{
						TpUnidCarga: "1", IdUnidCarga: "MSCU1234567",
						LacUnidCarga: []ctesimp.Lacre{{NLacre: "L0001"}},
						QtdRat:       tipos.Ptr(tipos.D("1")),
					}},
				}},
			}},
		},
		{
			CMunIni: 4314902, XMunIni: "PORTO ALEGRE", CMunFim: 4316907, XMunFim: "SANTA MARIA",
			Comp: []cte.Componente{
				{XNome: "FRETE PESO", VComp: tipos.D("900.00")},
				{XNome: "PEDAGIO", VComp: tipos.D("100.00")},
			},
			InfNFe: []ctesimp.InfNFe{{ChNFe: chaveNFe(102)}, {ChNFe: chaveNFe(103), PIN: "123456"}},
		},
	}
	c.InfCte.InfModal.Rodo = &cte.Rodo{RNTRC: "12345678"}

	base := tipos.D("2500.00")
	c.InfCte.Imp.ICMS.ICMS00 = &cte.ICMS00{
		CST: "00", VBC: base, PICMS: tipos.D("12.00"), VICMS: base.Percentual(tipos.D("12.00"), 2),
	}
	return c
}

// redespachoExemplo é o mesmo documento num redespacho: a prestação aponta o CT-e anterior, com uma NF-e parcial.
func redespachoExemplo() *ctesimp.CTeSimp {
	c := entregasExemplo()
	c.InfCte.Ide.TpServ = cte.ServicoRedespacho
	c.InfCte.Det = c.InfCte.Det[:1]
	c.InfCte.Det[0].InfNFe = nil
	c.InfCte.Det[0].InfDocAnt = []ctesimp.InfDocAnt{{
		ChCTe: chaveCTe(77), TpPrest: ctesimp.PrestacaoParcial,
		InfNFeTranspParcial: []ctesimp.NFeParcial{{ChNFe: chaveNFe(104)}},
	}}
	return c
}

// completoExemplo preenche todos os opcionais que o pacote implementa, numa substituição emitida em contingência
// offline: é o exemplo que prova contra o esquema os campos que os outros dois deixam vazios.
func completoExemplo() *ctesimp.CTeSimp {
	c := entregasExemplo()
	ide := &c.InfCte.Ide
	ide.TpCTe = ctesimp.SubstituicaoSimplificado
	ide.TpEmis = cte.EmissaoContingenciaOffline
	ide.DhCont = tipos.Ptr(tipos.DH("2026-03-04T07:45:00-03:00"))
	ide.XJust = "SEFAZ FORA DO AR NO MOMENTO DA EMISSAO"
	ide.Retira = cte.RetiraSim
	ide.XDetRetira = "RETIRA NA FILIAL DE CAXIAS DO SUL"

	c.InfCte.Compl = &ctesimp.Compl{
		XCaracAd: "ENTREGA AGEND", XCaracSer: "COLETA E ENTREGA",
		Fluxo:    &cte.Fluxo{XOrig: "POA", Pass: []cte.Passagem{{XPass: "CXS"}}, XDest: "SMA", XRota: "BR116"},
		XObs:     "MERCADORIA FRAGIL",
		ObsCont:  []cte.ObsCampo{{XCampo: "PEDIDO", XTexto: "PED-778899"}},
		ObsFisco: []cte.ObsCampo{{XCampo: "REGIME", XTexto: "ICMS NORMAL"}},
	}
	c.InfCte.Emit.IEST = "1234567890"
	c.InfCte.Emit.XFant = "EXEMPLO CARGAS"
	c.InfCte.Emit.ISUFEmit = "123456789"
	c.InfCte.Toma.ISUF = "12345678"
	c.InfCte.Toma.Fone = "5133331111"
	c.InfCte.Toma.Email = "fiscal@remetente.com.br"
	c.InfCte.InfCarga.XOutCat = "CAIXAS"
	c.InfCte.InfCarga.VCargaAverb = tipos.Ptr(tipos.D("85000.00"))

	nfe := &c.InfCte.Det[0].InfNFe[0]
	nfe.PIN = "123456"
	nfe.DPrev = tipos.Ptr(tipos.Data{Time: tipos.DH("2026-03-06T00:00:00-03:00").Time})
	transp := &nfe.InfUnidTransp[0]
	transp.LacUnidTransp = []ctesimp.Lacre{{NLacre: "T0001"}}
	transp.QtdRat = tipos.Ptr(tipos.D("0.333"))
	transp.InfUnidCarga[0].QtdRat = tipos.Ptr(tipos.D("0.5"))
	// A segunda entrega leva as unidades de carga direto na NF-e, sem unidade de transporte.
	c.InfCte.Det[1].InfNFe[0].InfUnidCarga = []ctesimp.UnidadeCarga{{TpUnidCarga: "3", IdUnidCarga: "PALLET01"}}

	c.InfCte.Cobr = &cte.Cobr{
		Fat: &cte.Fat{NFat: "FAT-1", VOrig: tipos.Ptr(tipos.D("2500.00")), VLiq: tipos.Ptr(tipos.D("2500.00"))},
		Dup: []cte.Dup{{NDup: "001", DVenc: tipos.Ptr(tipos.Data{Time: tipos.DH("2026-04-04T00:00:00-03:00").Time}), VDup: tipos.D("2500.00")}},
	}
	c.InfCte.InfCteSub = &ctesimp.InfCteSub{ChCte: chaveCTe(55), IndAlteraToma: "1"}
	c.InfCte.Imp.ICMSUFFim = &cte.ICMSUFFim{
		VBCUFFim: tipos.D("2500.00"), PFCPUFFim: tipos.D("2.00"), PICMSUFFim: tipos.D("17.00"), PICMSInter: tipos.D("12.00"),
		VFCPUFFim: tipos.D("50.00"), VICMSUFFim: tipos.D("125.00"), VICMSUFIni: tipos.D("0.00"),
	}
	c.InfCte.AutXML = []cte.AutXML{{CNPJ: "11222333000181"}, {CPF: "12345678909"}}
	// Os grupos da reforma tributária: IBS e CBS por prestação (o Preparar soma o IBSCBSTot), e os do ide e do fim.
	ide.GCompraGov = rtctest.CompraGov()
	ide.TpPagAnt = "1"
	ide.GPagAntecipado = &cte.PagAntecipado{ChDFePagAnt: rtctest.ChaveDFe(801)}
	ide.GMultaJuros = &cte.MultaJuros{ChCTeRef: rtctest.ChaveDFe(802)}
	c.InfCte.Det[0].VPrestLiq = tipos.Ptr(tipos.D("1485.00"))
	c.InfCte.Det[1].VPrestLiq = tipos.Ptr(tipos.D("990.00"))
	c.InfCte.Det[0].IBSCBS = rtctest.IBSCBS("1500.00")
	c.InfCte.Det[1].IBSCBS = rtctest.IBSCBS("1000.00")
	// O vTPrestLiq e o IBSCBSTot saem do Preparar, somados das prestações.
	c.InfCte.Total.VTotDFe = tipos.Ptr(tipos.D("2525.00"))
	c.InfCte.PgtoVinc = rtctest.PgtoVinc()
	c.InfCte.InfSolicNFF = rtctest.NFF()
	c.InfCte.InfPAA = rtctest.PAA()
	c.InfCte.InfRespTec = &cte.InfRespTec{CNPJ: "11222333000181", XContato: "SUPORTE FISCAL", Email: "suporte@exemplo.com.br", Fone: "5133332222"}
	return c
}

func preparado(t *testing.T, c *ctesimp.CTeSimp) *ctesimp.CTeSimp {
	t.Helper()
	if err := c.Preparar(); err != nil {
		t.Fatalf("Preparar: %v", err)
	}
	return c
}

func TestPrepararCalculaOsValoresENumeraOsItens(t *testing.T) {
	c := preparado(t, entregasExemplo())

	if c.InfCte.Det[0].NItem != 1 || c.InfCte.Det[1].NItem != 2 {
		t.Errorf("nItem = %d e %d, queria 1 e 2", c.InfCte.Det[0].NItem, c.InfCte.Det[1].NItem)
	}
	// O segundo item soma os componentes (900 + 100); o vRec vazio vira o vPrest.
	if got := c.InfCte.Det[1].VPrest.String(); got != "1000.00" {
		t.Errorf("det[1].vPrest = %s, queria 1000.00", got)
	}
	if got := c.InfCte.Det[0].VRec.String(); got != "1500.00" {
		t.Errorf("det[0].vRec = %s, queria 1500.00", got)
	}
	if got := c.InfCte.Total.VTPrest.String(); got != "2500.00" {
		t.Errorf("total.vTPrest = %s, queria 2500.00", got)
	}
	if got := c.InfCte.Total.VTRec.String(); got != "2500.00" {
		t.Errorf("total.vTRec = %s, queria 2500.00", got)
	}
	if err := c.Validar(); err != nil {
		t.Errorf("o exemplo preparado não passa no Validar: %v", err)
	}
}

func TestCompletoValida(t *testing.T) {
	c := preparado(t, completoExemplo())
	if err := c.Validar(); err != nil {
		t.Fatalf("o exemplo completo não passa no Validar: %v", err)
	}
	// O rateio guarda as três casas que o TDec_0302_0303 aceita.
	if got := c.InfCte.Det[0].InfNFe[0].InfUnidTransp[0].QtdRat.String(); got != "0.333" {
		t.Errorf("qtdRat = %s, queria 0.333", got)
	}
	// A contingência entra na chave pelo tpEmis.
	if c.Chave()[34] != '2' {
		t.Errorf("tpEmis na chave = %c, queria 2", c.Chave()[34])
	}
}

// O Preparar soma o IBS e a CBS das prestações no IBSCBSTot; um total que não bate é recusado.
func TestTotalDeIBSCBS(t *testing.T) {
	c := completoExemplo()
	// Valores distintos por esfera e por prestação, para a soma não passar por acaso.
	for i, det := range c.InfCte.Det {
		g := det.IBSCBS.GIBSCBS
		n := tipos.D([]string{"1.00", "2.00"}[i])
		g.GIBSUF.GDif.VDif = n
		g.GIBSUF.GDevTrib.VDevTrib = n.Somar(tipos.D("10.00"))
		g.GIBSMun.GDif.VDif = n.Somar(tipos.D("20.00"))
		g.GIBSMun.GDevTrib.VDevTrib = n.Somar(tipos.D("30.00"))
		g.GIBSMun.VIBSMun = n.Somar(tipos.D("40.00"))
		g.VIBS = g.GIBSUF.VIBSUF.Somar(g.GIBSMun.VIBSMun)
		g.GCBS.GDif.VDif = n.Somar(tipos.D("50.00"))
		g.GCBS.GDevTrib.VDevTrib = n.Somar(tipos.D("60.00"))
		det.IBSCBS.GEstornoCred.VIBSEstCred = n.Somar(tipos.D("70.00"))
		det.IBSCBS.GEstornoCred.VCBSEstCred = n.Somar(tipos.D("80.00"))
	}
	c = preparado(t, c)
	if err := c.Validar(); err != nil {
		t.Fatalf("Validar: %v", err)
	}
	if got := c.InfCte.Total.VTPrestLiq; got == nil || got.String() != "2475.00" {
		t.Errorf("vTPrestLiq = %v, queria 2475.00", got)
	}
	tot := c.InfCte.Total.IBSCBSTot
	if tot == nil {
		t.Fatal("o Preparar não montou o IBSCBSTot")
	}
	// Bases 1500 e 1000; IBS da UF de 0,1% (1,50 + 1,00) e CBS de 0,9% (13,50 + 9,00).
	for _, par := range [][2]string{
		{tot.VBCIBSCBS.String(), "2500.00"}, {tot.GIBS.GIBSUF.VIBSUF.String(), "2.50"},
		{tot.GIBS.GIBSUF.VDif.String(), "3.00"}, {tot.GIBS.GIBSUF.VDevTrib.String(), "23.00"},
		{tot.GIBS.GIBSMun.VDif.String(), "43.00"}, {tot.GIBS.GIBSMun.VDevTrib.String(), "63.00"},
		{tot.GIBS.GIBSMun.VIBSMun.String(), "83.00"}, {tot.GIBS.VIBS.String(), "85.50"},
		{tot.GCBS.VDif.String(), "103.00"}, {tot.GCBS.VDevTrib.String(), "123.00"}, {tot.GCBS.VCBS.String(), "22.50"},
		{tot.GEstornoCred.VIBSEstCred.String(), "143.00"}, {tot.GEstornoCred.VCBSEstCred.String(), "163.00"},
	} {
		if par[0] != par[1] {
			t.Errorf("total = %s, queria %s", par[0], par[1])
		}
	}
	c.InfCte.Total.IBSCBSTot.GCBS.VCBS = tipos.D("99.99")
	if err := c.Validar(); err == nil || !strings.Contains(err.Error(), "total.IBSCBSTot") {
		t.Errorf("total de CBS errado não foi recusado: %v", err)
	}
	c.InfCte.Total.IBSCBSTot = nil
	if err := c.Validar(); err == nil || !strings.Contains(err.Error(), "falta o total") {
		t.Errorf("falta do IBSCBSTot não foi recusada: %v", err)
	}
	c.InfCte.Total.VTPrestLiq = tipos.Ptr(tipos.D("1.00"))
	if err := c.Validar(); err == nil || !strings.Contains(err.Error(), "total.vTPrestLiq") {
		t.Errorf("vTPrestLiq errado não foi recusado: %v", err)
	}
	semImposto := preparado(t, entregasExemplo())
	semImposto.InfCte.Total.IBSCBSTot = tot
	if err := semImposto.Validar(); err == nil || !strings.Contains(err.Error(), "nenhuma prestação tem IBS") {
		t.Errorf("total de IBS e CBS sem prestação com eles não foi recusado: %v", err)
	}
	if got := ctesimp.SomarIBSCBS(entregasExemplo().InfCte.Det); got != nil {
		t.Errorf("sem IBS e CBS nas prestações, o total deveria ser nil; veio %+v", got)
	}
}

func TestPrepararDuasVezesDaOMesmoXML(t *testing.T) {
	c := preparado(t, completoExemplo())
	primeiro, err := c.XML()
	if err != nil {
		t.Fatal(err)
	}
	segundo, err := preparado(t, c).XML()
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(primeiro, segundo) {
		t.Error("Preparar não é idempotente: o segundo XML difere do primeiro")
	}
}

func TestSemCalculoDeTotaisPreservaOsValores(t *testing.T) {
	c := entregasExemplo()
	c.InfCte.Det[0].VRec = tipos.D("1500.00")
	c.InfCte.Det[1].VPrest, c.InfCte.Det[1].VRec = tipos.D("1000.00"), tipos.D("1000.00")
	c.InfCte.Total = ctesimp.Total{VTPrest: tipos.D("9.99"), VTRec: tipos.D("9.99")}
	if err := c.Preparar(ctesimp.OpcoesPreparo{SemCalculoDeTotais: true}); err != nil {
		t.Fatal(err)
	}
	if got := c.InfCte.Total.VTPrest.String(); got != "9.99" {
		t.Errorf("total.vTPrest = %s; o Preparar recalculou mesmo com SemCalculoDeTotais", got)
	}
	var erros ctesimp.Erros
	if err := c.Validar(); !errors.As(err, &erros) || !strings.Contains(err.Error(), "total.vTPrest") {
		t.Errorf("Validar não acusou o total preservado errado: %v", err)
	}
}

func TestPrepararGeraChaveDoModelo57(t *testing.T) {
	c := preparado(t, entregasExemplo())
	const esperado = "43" + "2603" + cnpjTransportadora + "57" + "001" + "000001234" + "1" + "11223344"
	if c.Chave()[:43] != esperado {
		t.Errorf("chave = %s, queria começar com %s", c.Chave(), esperado)
	}
	if err := chave.Validar(c.Chave()); err != nil {
		t.Errorf("chave inválida: %v", err)
	}
	if !strings.HasPrefix(c.InfCte.Id, "CTe") {
		t.Errorf("Id = %q, queria o prefixo CTe", c.InfCte.Id)
	}
}

func TestXMLSegueAOrdemDoEsquema(t *testing.T) {
	c := preparado(t, entregasExemplo())
	dados, err := c.XML()
	if err != nil {
		t.Fatal(err)
	}
	s := string(dados)
	if !strings.HasPrefix(s, `<CTeSimp xmlns="http://www.portalfiscal.inf.br/cte"><infCte versao="4.00" Id="CTe`) {
		t.Errorf("início do XML: %.120s", s)
	}
	ordem := []string{"<ide>", "<emit>", "<toma>", "<infCarga>", `<det nItem="1">`, `<det nItem="2">`,
		"<infModal", "<imp>", "<total>"}
	ultimo := -1
	for _, marca := range ordem {
		i := strings.Index(s, marca)
		if i < 0 {
			t.Fatalf("o XML não tem %s", marca)
		}
		if i < ultimo {
			t.Errorf("%s fora de ordem", marca)
		}
		ultimo = i
	}
	// No det, os municípios vêm antes dos valores, e os valores antes dos documentos.
	det := s[strings.Index(s, `<det nItem="2">`):]
	if strings.Index(det, "<cMunIni>") > strings.Index(det, "<vPrest>") ||
		strings.Index(det, "<vRec>") > strings.Index(det, "<Comp>") ||
		strings.Index(det, "<Comp>") > strings.Index(det, "<infNFe>") {
		t.Errorf("ordem do det: %.300s", det)
	}
}

func TestAssinarLerEMontarOProc(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	original := entregasExemplo()
	assinado, err := original.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := xmldsig.Verificar(assinado); err != nil {
		t.Fatalf("a assinatura não confere: %v", err)
	}

	prot := &cte.ProtCTe{InfProt: cte.InfProt{ChCTe: original.Chave(), NProt: "143260000012345", CStat: 100}}
	proc, err := ctesimp.MontarCTeSimpProc(assinado, prot)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Contains(proc, assinado) {
		t.Error("o cteSimpProc não preserva os bytes assinados")
	}

	lido, protLido, err := ctesimp.LerCTeSimpProc(proc)
	if err != nil {
		t.Fatal(err)
	}
	if lido.Chave() != original.Chave() || len(lido.InfCte.Det) != 2 {
		t.Errorf("lido: chave %s, %d itens", lido.Chave(), len(lido.InfCte.Det))
	}
	if protLido == nil || protLido.InfProt.NProt != "143260000012345" {
		t.Errorf("protocolo lido: %+v", protLido)
	}
	if got := lido.InfCte.Det[0].InfNFe[0].InfUnidTransp[0].InfUnidCarga[0].IdUnidCarga; got != "MSCU1234567" {
		t.Errorf("unidade de carga lida: %q", got)
	}

	// Sem protocolo, o documento ainda não foi autorizado; protocolo cortado é erro.
	if _, p, err := ctesimp.LerCTeSimpProc(assinado); err != nil || p != nil {
		t.Errorf("sem protocolo: prot %v, erro %v", p, err)
	}
	cortado := proc[:bytes.LastIndex(proc, []byte("</protCTe>"))]
	if _, _, err := ctesimp.LerCTeSimpProc(cortado); err == nil {
		t.Error("protocolo aberto e não fechado foi aceito")
	}
}

func TestEnvioSincronoComprimeOsBytesAssinados(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	assinado, err := entregasExemplo().AssinarCom(cert)
	if err != nil {
		t.Fatal(err)
	}
	envio, err := ctesimp.MontarEnvioSincrono(assinado)
	if err != nil {
		t.Fatal(err)
	}
	comprimido, err := base64.StdEncoding.DecodeString(string(envio))
	if err != nil {
		t.Fatal(err)
	}
	r, err := gzip.NewReader(bytes.NewReader(comprimido))
	if err != nil {
		t.Fatal(err)
	}
	conteudo, err := io.ReadAll(r)
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(conteudo, assinado) {
		t.Error("o envio não traz os bytes assinados")
	}
}

func TestLerRecusaOutroDocumento(t *testing.T) {
	if _, err := ctesimp.Ler([]byte(`<CTe xmlns="http://www.portalfiscal.inf.br/cte"><infCte/></CTe>`)); err == nil {
		t.Error("um <CTe> foi lido como <CTeSimp>")
	}
}

func TestValidarRecusa(t *testing.T) {
	casos := []struct {
		nome   string
		campo  string
		mexer  func(c *ctesimp.CTeSimp)
		manual bool // true: monta os valores à mão, sem o cálculo do Preparar
	}{
		{"finalidade do CT-e comum", "ide.tpCTe", func(c *ctesimp.CTeSimp) { c.InfCte.Ide.TpCTe = "0" }, false},
		{"serviço fora da faixa", "ide.tpServ", func(c *ctesimp.CTeSimp) { c.InfCte.Ide.TpServ = cte.ServicoVinculadoMultimodal }, false},
		{"modal aéreo", "ide.modal", func(c *ctesimp.CTeSimp) { c.InfCte.Ide.Modal = cte.ModalAereo }, false},
		{"emissão 5", "ide.tpEmis", func(c *ctesimp.CTeSimp) { c.InfCte.Ide.TpEmis = cte.TipoEmissao("5") }, false},
		{"natureza longa", "ide.natOp", func(c *ctesimp.CTeSimp) { c.InfCte.Ide.NatOp = strings.Repeat("A", 61) }, false},
		{"retira inválido", "ide.retira", func(c *ctesimp.CTeSimp) { c.InfCte.Ide.Retira = "2" }, false},
		{"sem CRT", "emit.CRT", func(c *ctesimp.CTeSimp) { c.InfCte.Emit.CRT = "" }, false},
		{"tomador sem documento", "toma", func(c *ctesimp.CTeSimp) { c.InfCte.Toma.CNPJ = "" }, false},
		{"tomador contribuinte sem IE", "toma.IE", func(c *ctesimp.CTeSimp) { c.InfCte.Toma.IE = "" }, false},
		{"papel do tomador", "toma.toma", func(c *ctesimp.CTeSimp) { c.InfCte.Toma.Toma = "5" }, false},
		{"sem valor da carga", "infCarga.vCarga", func(c *ctesimp.CTeSimp) { c.InfCte.InfCarga.VCarga = nil }, false},
		{"sem quantidade", "infCarga.infQ", func(c *ctesimp.CTeSimp) { c.InfCte.InfCarga.InfQ = nil }, false},
		{"tipo de medida 16", "infCarga.infQ[0].tpMed", func(c *ctesimp.CTeSimp) { c.InfCte.InfCarga.InfQ[0].TpMed = "16" }, false},
		{"sem itens", "det", func(c *ctesimp.CTeSimp) { c.InfCte.Det = nil }, false},
		{"item fora de ordem", "det[1]@nItem", func(c *ctesimp.CTeSimp) { c.InfCte.Det[1].NItem = 5 }, false},
		{"componente longo", "det[1].Comp[0].xNome", func(c *ctesimp.CTeSimp) { c.InfCte.Det[1].Comp[0].XNome = "FRETE PESO CUBADO" }, false},
		{"NF-e e documento anterior", "det[0]", func(c *ctesimp.CTeSimp) {
			c.InfCte.Det[0].InfDocAnt = []ctesimp.InfDocAnt{{ChCTe: chaveCTe(1), TpPrest: ctesimp.PrestacaoTotal}}
		}, false},
		{"item sem documento", "det[0]", func(c *ctesimp.CTeSimp) { c.InfCte.Det[0].InfNFe = nil }, false},
		{"chave da NF-e errada", "det[1].infNFe[0].chNFe", func(c *ctesimp.CTeSimp) {
			c.InfCte.Det[1].InfNFe[0].ChNFe = chaveNFe(102)[:43] + "0"
		}, false},
		{"contêiner com minúscula", "det[0].infNFe[0].infUnidTransp[0].infUnidCarga[0].idUnidCarga", func(c *ctesimp.CTeSimp) {
			c.InfCte.Det[0].InfNFe[0].InfUnidTransp[0].InfUnidCarga[0].IdUnidCarga = "mscu1"
		}, false},
		{"contêiner com hífen", "det[0].infNFe[0].infUnidTransp[0].infUnidCarga[0].idUnidCarga", func(c *ctesimp.CTeSimp) {
			c.InfCte.Det[0].InfNFe[0].InfUnidTransp[0].InfUnidCarga[0].IdUnidCarga = "MSCU-1"
		}, false},
		{"CFOP de entrada", "ide.CFOP", func(c *ctesimp.CTeSimp) { c.InfCte.Ide.CFOP = "4353" }, false},
		{"PIN com zero à esquerda", "det[1].infNFe[1].PIN", func(c *ctesimp.CTeSimp) { c.InfCte.Det[1].InfNFe[1].PIN = "0123" }, false},
		{"CEP curto", "toma.enderToma.CEP", func(c *ctesimp.CTeSimp) { c.InfCte.Toma.EnderToma.CEP = "9000000" }, false},
		{"IE do tomador com letra", "toma.IE", func(c *ctesimp.CTeSimp) { c.InfCte.Toma.IE = "12AB" }, false},
		{"e-mail sem domínio", "toma.email", func(c *ctesimp.CTeSimp) { c.InfCte.Toma.Email = "fiscal@semponto" }, false},
		{"emitente em EX", "emit.enderEmit.UF", func(c *ctesimp.CTeSimp) { c.InfCte.Emit.EnderEmit.UF = "EX" }, false},
		{"observações demais", "compl.ObsCont", func(c *ctesimp.CTeSimp) {
			c.InfCte.Compl = &ctesimp.Compl{ObsCont: make([]cte.ObsCampo, 11)}
			for i := range c.InfCte.Compl.ObsCont {
				c.InfCte.Compl.ObsCont[i] = cte.ObsCampo{XCampo: "C", XTexto: "T"}
			}
		}, false},
		{"característica longa", "compl.xCaracAd", func(c *ctesimp.CTeSimp) {
			c.InfCte.Compl = &ctesimp.Compl{XCaracAd: strings.Repeat("A", 16)}
		}, false},
		{"autorizados demais", "autXML", func(c *ctesimp.CTeSimp) {
			c.InfCte.AutXML = make([]cte.AutXML, 11)
			for i := range c.InfCte.AutXML {
				c.InfCte.AutXML[i] = cte.AutXML{CNPJ: "11222333000181"}
			}
		}, false},
		{"unidade de transporte 8", "det[0].infNFe[0].infUnidTransp[0].tpUnidTransp", func(c *ctesimp.CTeSimp) {
			c.InfCte.Det[0].InfNFe[0].InfUnidTransp[0].TpUnidTransp = "8"
		}, false},
		{"sem RNTRC", "infModal.rodo.RNTRC", func(c *ctesimp.CTeSimp) { c.InfCte.InfModal.Rodo.RNTRC = "" }, false},
		{"sem ICMS", "imp.ICMS", func(c *ctesimp.CTeSimp) { c.InfCte.Imp.ICMS.ICMS00 = nil }, false},
		{"total diferente da soma", "total.vTPrest", func(c *ctesimp.CTeSimp) { c.InfCte.Total.VTPrest = tipos.D("2600.00") }, true},
		{"componentes diferentes do valor", "det[1].vPrest", func(c *ctesimp.CTeSimp) { c.InfCte.Det[1].VPrest = tipos.D("999.00") }, true},
		{"substituição sem o substituído", "infCteSub", func(c *ctesimp.CTeSimp) { c.InfCte.Ide.TpCTe = ctesimp.SubstituicaoSimplificado }, false},
		{"grupo de substituição no normal", "infCteSub", func(c *ctesimp.CTeSimp) {
			c.InfCte.InfCteSub = &ctesimp.InfCteSub{ChCte: chaveCTe(9)}
		}, false},
	}
	for _, caso := range casos {
		t.Run(caso.nome, func(t *testing.T) {
			c := preparado(t, entregasExemplo())
			caso.mexer(c)
			if !caso.manual {
				c.CalcularTotais()
			}
			err := c.Validar()
			var erros ctesimp.Erros
			if !errors.As(err, &erros) {
				t.Fatalf("Validar aceitou; queria erro em %s", caso.campo)
			}
			for _, e := range erros {
				if e.Campo == caso.campo {
					return
				}
			}
			t.Errorf("queria erro em %s; veio: %v", caso.campo, err)
		})
	}
}

func TestUFDoExteriorNoTrajeto(t *testing.T) {
	c := preparado(t, entregasExemplo())
	c.InfCte.Ide.UFFim = "EX" // TUf aceita EX; o emitente (TUF_sem_EX) não
	if err := c.Validar(); err != nil {
		t.Errorf("UFFim EX foi recusada: %v", err)
	}
}

func TestRedespachoComDocumentoAnteriorValida(t *testing.T) {
	if err := preparado(t, redespachoExemplo()).Validar(); err != nil {
		t.Errorf("o redespacho de exemplo não passa no Validar: %v", err)
	}
}
