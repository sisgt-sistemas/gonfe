package cteos_test

import (
	"errors"
	"testing"

	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/xsd"
)

// O CT-e OS de exemplo, assinado, passa no esquema oficial (cteOS_v4.00) e o modal rodoviário de outros serviços no
// esquema dele.
func TestFretamentoPassaNoEsquema(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	assinado, err := fretamentoExemplo().AssinarCom(cert)
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	err = xsd.ValidarCTeOS(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("o CT-e OS de exemplo não passa no esquema: %v", err)
	}
}
