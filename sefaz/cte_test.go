package sefaz_test

import (
	"bytes"
	"compress/gzip"
	"context"
	"encoding/base64"
	"errors"
	"io"
	"net/http"
	"strings"
	"testing"
	"time"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/sefaz"
	"github.com/sisgt-sistemas/gonfe/uf"
)

// enveloparCTe embrulha a resposta no envelope que os serviços de CT-e
// devolvem.
func enveloparCTe(conteudo string) string {
	return `<?xml version="1.0" encoding="utf-8"?>` +
		`<soap:Envelope xmlns:soap="http://www.w3.org/2003/05/soap-envelope"><soap:Body>` +
		`<cteResultMsg xmlns="http://www.portalfiscal.inf.br/cte/wsdl/CTeRecepcaoEventoV4">` +
		conteudo +
		`</cteResultMsg></soap:Body></soap:Envelope>`
}

func clienteCTeApontando(t *testing.T, endereco string) *sefaz.ClienteCTe {
	t.Helper()
	endpoints := make(map[sefaz.ServicoCTe]string)
	for _, s := range sefaz.ServicosCTe() {
		endpoints[s] = endereco
	}
	c, err := sefaz.NovoClienteCTe(sefaz.ConfigCTe{
		UF:        uf.RS,
		Ambiente:  cte.Homologacao,
		Endpoints: endpoints,
		HTTP:      &http.Client{Timeout: 5 * time.Second},
	})
	if err != nil {
		t.Fatalf("NovoClienteCTe: %v", err)
	}
	return c
}

func TestCTeEnviarEvento(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparCTe(
			`<retEventoCTe versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte"><infEvento>` +
				`<tpAmb>2</tpAmb><verAplic>RS20260304</verAplic><cOrgao>43</cOrgao>` +
				`<cStat>135</cStat><xMotivo>Evento registrado e vinculado ao CT-e</xMotivo>` +
				`<chCTe>43260312345678000195570010000009871122334411</chCTe>` +
				`<tpEvento>110111</tpEvento><nSeqEvento>1</nSeqEvento>` +
				`<dhRegEvento>2026-03-04T10:00:30-03:00</dhRegEvento><nProt>143260000077777</nProt>` +
				`</infEvento></retEventoCTe>`)
	})

	c := clienteCTeApontando(t, s.URL)
	resposta, err := c.EnviarEvento(context.Background(),
		[]byte(`<eventoCTe versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte"><infEvento/></eventoCTe>`))
	if err != nil {
		t.Fatalf("EnviarEvento: %v", err)
	}
	if !resposta.Registrado() {
		t.Errorf("cStat = %d; o evento deveria ter sido aceito", resposta.InfEvento.CStat)
	}
	if resposta.InfEvento.NProt != "143260000077777" {
		t.Errorf("nProt = %q", resposta.InfEvento.NProt)
	}
}

func TestCTeEventoRejeitado(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparCTe(
			`<retEventoCTe versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte"><infEvento>` +
				`<tpAmb>2</tpAmb><verAplic>RS20260304</verAplic><cOrgao>43</cOrgao>` +
				`<cStat>573</cStat><xMotivo>Rejeicao: Duplicidade de evento</xMotivo>` +
				`</infEvento></retEventoCTe>`)
	})

	c := clienteCTeApontando(t, s.URL)
	resposta, err := c.EnviarEvento(context.Background(),
		[]byte(`<eventoCTe versao="4.00"><infEvento/></eventoCTe>`))
	if err == nil {
		t.Fatal("esperava erro com a rejeição")
	}
	if resposta == nil || resposta.InfEvento.CStat != 573 {
		t.Errorf("resposta = %+v", resposta)
	}
}

func TestCTeEnviarEventoRejeitaConteudoErrado(t *testing.T) {
	c := clienteCTeApontando(t, "http://127.0.0.1:1")
	// Um CT-e não é um evento, e "<CTe" nem sequer é prefixo de "<eventoCTe".
	if _, err := c.EnviarEvento(context.Background(), []byte("<CTe/>")); err == nil {
		t.Error("esperava erro ao enviar um documento que não é evento")
	}
}

// O CT-e Simplificado vai para a recepção dele (CTeRecepcaoSimpV4, método cteRecepcaoSimp, NT 2024.002), com o
// documento comprimido em gzip e base64, e o retCTeSimp volta como resposta com o protocolo. "<CTe" é prefixo de
// "<CTeSimp": sem o caso próprio, o documento iria para a recepção do CT-e.
func TestCTeAutorizarOCTeSimplificado(t *testing.T) {
	const documento = `<CTeSimp xmlns="http://www.portalfiscal.inf.br/cte"><infCte versao="4.00" Id="CTe1"/></CTeSimp>`
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparCTe(`<retCTeSimp versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte">` +
			`<tpAmb>2</tpAmb><cUF>43</cUF><verAplic>RS20261113</verAplic><cStat>100</cStat>` +
			`<xMotivo>Autorizado o uso do CT-e</xMotivo>` +
			`<protCTe versao="4.00"><infProt><tpAmb>2</tpAmb><verAplic>RS20261113</verAplic>` +
			`<chCTe>43261112345678000195570010000012341112233445</chCTe><dhRecbto>2026-11-13T10:00:00-03:00</dhRecbto>` +
			`<nProt>143260000099999</nProt><cStat>100</cStat><xMotivo>Autorizado o uso do CT-e</xMotivo></infProt></protCTe>` +
			`</retCTeSimp>`)
	})
	c := clienteCTeApontando(t, s.URL)
	resposta, err := c.Autorizar(context.Background(), []byte(documento))
	if err != nil {
		t.Fatalf("Autorizar(CTeSimp): %v", err)
	}
	if resposta.Versao != "4.00" || resposta.TpAmb != cte.Homologacao || resposta.CUF != 43 ||
		resposta.VerAplic != "RS20261113" || resposta.CStat != 100 || resposta.XMotivo != "Autorizado o uso do CT-e" ||
		resposta.ProtCTe == nil || resposta.ProtCTe.InfProt.NProt != "143260000099999" {
		t.Errorf("resposta = %+v", resposta)
	}

	enviado, _ := s.ultimoCorpo.Load().(string)
	if !strings.Contains(enviado, `<cteDadosMsg xmlns="http://www.portalfiscal.inf.br/cte/wsdl/CTeRecepcaoSimpV4">`) {
		t.Errorf("a mensagem não foi para o CTeRecepcaoSimpV4:\n%s", enviado)
	}
	if tipo, _ := s.ultimoTipoConteudo.Load().(string); !strings.Contains(tipo, "CTeRecepcaoSimpV4/cteRecepcaoSimp") {
		t.Errorf("ação SOAP = %q, queria a do cteRecepcaoSimp", tipo)
	}
	inicio := strings.Index(enviado, "CTeRecepcaoSimpV4\">") + len("CTeRecepcaoSimpV4\">")
	fim := strings.Index(enviado, "</cteDadosMsg>")
	comprimido, err := base64.StdEncoding.DecodeString(enviado[inicio:fim])
	if err != nil {
		t.Fatalf("a mensagem não está em base64: %v", err)
	}
	r, err := gzip.NewReader(bytes.NewReader(comprimido))
	if err != nil {
		t.Fatalf("a mensagem não está em gzip: %v", err)
	}
	if conteudo, _ := io.ReadAll(r); string(conteudo) != documento {
		t.Errorf("conteúdo enviado = %s", conteudo)
	}
}

// Uma rejeição do CT-e Simplificado vira *sefaz.ErroSefaz com o cStat e o motivo, e a resposta volta junto.
func TestCTeAutorizarOCTeSimplificadoRejeitado(t *testing.T) {
	s := novoServidor(t, func(int32, string) (int, string) {
		return 200, enveloparCTe(`<retCTeSimp versao="4.00" xmlns="http://www.portalfiscal.inf.br/cte">` +
			`<tpAmb>2</tpAmb><cUF>43</cUF><verAplic>RS20261113</verAplic><cStat>215</cStat>` +
			`<xMotivo>Rejeicao: Falha no schema XML</xMotivo></retCTeSimp>`)
	})
	c := clienteCTeApontando(t, s.URL)
	resposta, err := c.Autorizar(context.Background(), []byte(`<CTeSimp xmlns="http://www.portalfiscal.inf.br/cte"><infCte/></CTeSimp>`))
	var rejeicao *sefaz.ErroSefaz
	if !errors.As(err, &rejeicao) || rejeicao.CStat != 215 || !strings.Contains(rejeicao.XMotivo, "schema") {
		t.Fatalf("erro = %v, queria ErroSefaz com cStat 215", err)
	}
	if resposta == nil || resposta.CStat != 215 || resposta.ProtCTe != nil {
		t.Errorf("resposta = %+v; queria a rejeição, sem protocolo", resposta)
	}
}

// Os endereços do CTeRecepcaoSimpV4 são os das relações de serviços do Portal do CT-e, nos 8 autorizadores e nos dois
// ambientes; o MT o publica em outra aplicação que a da recepção síncrona.
func TestEnderecosDoCTeSimplificado(t *testing.T) {
	const (
		svrsP = "https://cte.svrs.rs.gov.br/ws/CTeRecepcaoSimpV4/CTeRecepcaoSimpV4.asmx"
		svrsH = "https://cte-homologacao.svrs.rs.gov.br/ws/CTeRecepcaoSimpV4/CTeRecepcaoSimpV4.asmx"
		spP   = "https://nfe.fazenda.sp.gov.br/CTeWS/WS/CTeRecepcaoSimpV4.asmx"
		spH   = "https://homologacao.nfe.fazenda.sp.gov.br/CTeWS/WS/CTeRecepcaoSimpV4.asmx"
	)
	esperado := map[sefaz.AutorizadorCTe][2]string{
		sefaz.AutorizadorCTeSVRS:  {svrsP, svrsH},
		sefaz.AutorizadorCTeSVCRS: {svrsP, svrsH},
		sefaz.AutorizadorCTeSP:    {spP, spH},
		sefaz.AutorizadorCTeSVCSP: {spP, spH},
		sefaz.AutorizadorCTeMG:    {"https://cte.fazenda.mg.gov.br/cte/services/CTeRecepcaoSimpV4", "https://hcte.fazenda.mg.gov.br/cte/services/CTeRecepcaoSimpV4"},
		sefaz.AutorizadorCTeMS:    {"https://producao.cte.ms.gov.br/ws/CTeRecepcaoSimpV4", "https://homologacao.cte.ms.gov.br/ws/CTeRecepcaoSimpV4"},
		sefaz.AutorizadorCTeMT:    {"https://cte.sefaz.mt.gov.br/cte-ws/services/CTeRecepcaoSimpV4", "https://homologacao.sefaz.mt.gov.br/cte-ws/services/CTeRecepcaoSimpV4"},
		sefaz.AutorizadorCTePR:    {"https://cte.fazenda.pr.gov.br/cte4/CTeRecepcaoSimpV4", "https://homologacao.cte.fazenda.pr.gov.br/cte4/CTeRecepcaoSimpV4"},
	}
	for autorizador, urls := range esperado {
		for i, ambiente := range []cte.Ambiente{cte.Producao, cte.Homologacao} {
			got, err := sefaz.URLCTe(autorizador, ambiente, sefaz.ServicoCTeRecepcaoSimp)
			if err != nil || got != urls[i] {
				t.Errorf("%s/%s = %q, %v; queria %q", autorizador, ambiente, got, err, urls[i])
			}
		}
	}
}
