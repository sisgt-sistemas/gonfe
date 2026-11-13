package mdfe_test

import (
	"bytes"
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

// O produto predominante completo, com GTIN, NCM e carga lotação, sai na ordem do esquema.
func TestProdutoPredominantePassaNoEsquema(t *testing.T) {
	m := manifestoExemplo()
	m.InfMDFe.ProdPred = &mdfe.ProdPred{
		TpCarga: "05", XProd: "BEBIDAS EM GERAL", CEAN: "SEM GTIN", NCM: "2202.10.00",
		InfLotacao: &mdfe.InfLotacao{
			InfLocalCarrega:    mdfe.InfLocal{CEP: "01310-100"},
			InfLocalDescarrega: mdfe.InfLocal{CEP: "20040-020"},
		},
	}
	assinado, err := m.AssinarCom(certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora}))
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if !bytes.Contains(assinado, []byte("<cEAN>SEM GTIN</cEAN><NCM>22021000</NCM><infLotacao>")) {
		t.Errorf("GTIN e NCM fora do lugar ou sem normalizar:\n%s", assinado)
	}
	err = xsd.ValidarMDFe(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("o produto predominante não passa no esquema: %v", err)
	}
}
