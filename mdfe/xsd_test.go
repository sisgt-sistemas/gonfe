package mdfe_test

import (
	"errors"
	"testing"

	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/mdfe"
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

// O esquema da NT 2025.001 v1.04 aceita o processo de emissão 4 (Provedor de Assinatura e Autorização); o anterior
// só aceitava o 0.
func TestManifestoPorPAAPassaNoEsquema(t *testing.T) {
	m := manifestoExemplo()
	m.InfMDFe.Ide.ProcEmi = mdfe.EmissaoPAA
	assinado, err := m.AssinarCom(certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora}))
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	err = xsd.ValidarMDFe(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("o MDF-e emitido por PAA não passa no esquema: %v", err)
	}
}
