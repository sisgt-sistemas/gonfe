package cte_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/xsd"
)

// O CT-e de exemplo, assinado, passa no esquema oficial (PL_CTe_400) e o modal rodoviário no esquema dele: o que o
// pacote monta é o que a SEFAZ aceita. Um nome além dos 60 caracteres do xNome é recusado pelo esquema.
func TestConhecimentoPassaNoEsquema(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	assinado, err := conhecimentoExemplo().AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	err = xsd.ValidarCTe(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("o CT-e de exemplo não passa no esquema: %v", err)
	}

	longo := conhecimentoExemplo()
	longo.InfCte.Emit.XNome = strings.Repeat("A", 61)
	assinado, err = longo.AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := xsd.ValidarCTe(assinado); !errors.Is(err, xsd.ErrInvalido) || !strings.Contains(err.Error(), "xNome") {
		t.Fatalf("um xNome de 61 caracteres passou no esquema: %v", err)
	}
}
