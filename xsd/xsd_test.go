package xsd_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sisgt-sistemas/gonfe/xsd"
)

func skipSemXmllint(t *testing.T, err error) {
	t.Helper()
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
}

// Documento que não segue o esquema é recusado com a saída do xmllint; esquema que não existe é erro.
func TestValidarRecusa(t *testing.T) {
	err := xsd.Validar([]byte(`<CTe xmlns="http://www.portalfiscal.inf.br/cte"><nada/></CTe>`), xsd.CTe)
	skipSemXmllint(t, err)
	if !errors.Is(err, xsd.ErrInvalido) || !strings.Contains(err.Error(), "nada") {
		t.Fatalf("um CT-e sem infCte passou: %v", err)
	}
	if err := xsd.Validar([]byte(`<x/>`), xsd.Esquema("cte/inexistente.xsd")); err == nil || errors.Is(err, xsd.ErrInvalido) {
		t.Fatalf("esquema desconhecido: %v", err)
	}
}

// O modal sai do infModal com o namespace do documento, como o esquema do modal pede.
func TestModalGanhaONamespace(t *testing.T) {
	doc := []byte(`<CTe xmlns="http://www.portalfiscal.inf.br/cte"><infCte><infModal versaoModal="4.00"><rodo><RNTRC>12345678</RNTRC></rodo></infModal></infCte></CTe>`)
	modal, err := xsd.Modal(doc, "http://www.portalfiscal.inf.br/cte")
	if err != nil || string(modal) != `<rodo xmlns="http://www.portalfiscal.inf.br/cte"><RNTRC>12345678</RNTRC></rodo>` {
		t.Fatalf("modal: %s %v", modal, err)
	}
	err = xsd.Validar(modal, xsd.CTeModalRodoviario)
	skipSemXmllint(t, err)
	if err != nil {
		t.Fatalf("o modal rodoviário mínimo não passa: %v", err)
	}
	if _, err := xsd.Modal([]byte(`<CTe/>`), "x"); !errors.Is(err, xsd.ErrInvalido) {
		t.Fatalf("sem infModal: %v", err)
	}
}

// Os esquemas de documento carregam (os includes e imports resolvem dentro do pacote): um documento vazio é recusado
// pelo conteúdo, não por esquema quebrado.
func TestOsEsquemasCarregam(t *testing.T) {
	for esquema, raiz := range map[xsd.Esquema]string{
		xsd.CTe:     `<CTe xmlns="http://www.portalfiscal.inf.br/cte"><x/></CTe>`,
		xsd.CTeSimp: `<CTeSimp xmlns="http://www.portalfiscal.inf.br/cte"><x/></CTeSimp>`,
		xsd.MDFe:    `<MDFe xmlns="http://www.portalfiscal.inf.br/mdfe"><x/></MDFe>`,
	} {
		err := xsd.Validar([]byte(raiz), esquema)
		skipSemXmllint(t, err)
		if !errors.Is(err, xsd.ErrInvalido) || !strings.Contains(err.Error(), "Element '{http") || strings.Contains(err.Error(), "failed to compile") {
			t.Errorf("%s: %v", esquema, err)
		}
	}
}
