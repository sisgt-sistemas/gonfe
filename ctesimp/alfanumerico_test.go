package ctesimp_test

import (
	"bytes"
	"errors"
	"testing"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/xsd"
)

// cnpjAlfa é o CNPJ alfanumérico do exemplo da NT 2025.001 (12.ABC.345/01DE-35).
const cnpjAlfa = "12ABC34501DE35"

// Um CT-e Simplificado de emitente e tomador com CNPJ alfanumérico, com uma NF-e de emitente alfanumérico: as letras
// chegam ao XML e o esquema oficial aceita o documento.
func TestCTeSimpComCNPJAlfanumerico(t *testing.T) {
	c := entregasExemplo()
	c.InfCte.Emit.CNPJ = "12.abc.345/01de-35"
	c.InfCte.Toma.CNPJ = cnpjAlfa
	nfeAlfa := chave.MustNova(chave.Chave{CUF: 35, Ano: 26, Mes: 7, CNPJ: cnpjAlfa, Modelo: 55, Serie: 1,
		Numero: 79, TpEmis: 1, CNF: "10203042"})
	c.InfCte.Det[1].InfNFe[0].ChNFe = nfeAlfa

	assinado, err := c.AssinarCom(certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjAlfa}))
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := c.Validar(); err != nil {
		t.Errorf("Validar recusou o CNPJ alfanumérico: %v", err)
	}
	for _, trecho := range []string{"<CNPJ>" + cnpjAlfa + "</CNPJ>", "<chNFe>" + nfeAlfa + "</chNFe>"} {
		if !bytes.Contains(assinado, []byte(trecho)) {
			t.Errorf("o XML não traz %s", trecho)
		}
	}
	err = xsd.ValidarCTeSimp(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("o CT-e Simplificado com CNPJ alfanumérico não passa no esquema: %v", err)
	}
}
