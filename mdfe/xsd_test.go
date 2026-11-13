package mdfe_test

import (
	"errors"
	"testing"

	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/xsd"
)

// O MDF-e de exemplo, assinado, passa no esquema oficial (PL_MDFe_300b) e o modal rodoviário no esquema dele.
func TestManifestoPassaNoEsquema(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	assinado, err := manifestoExemplo().AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	err = xsd.ValidarMDFe(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("o MDF-e de exemplo não passa no esquema: %v", err)
	}
}
