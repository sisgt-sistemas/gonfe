package cte_test

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

// Um CT-e de emitente com CNPJ alfanumérico, transportando a NF-e de outro emitente alfanumérico: as letras chegam
// ao XML (o Preparar não as apaga), a chave segue o TChDFe e o esquema oficial aceita o documento.
func TestConhecimentoComCNPJAlfanumerico(t *testing.T) {
	c := conhecimentoExemplo()
	c.InfCte.Emit.CNPJ = "12.abc.345/01de-35" // máscara e minúsculas, como pode vir do cadastro
	nfeAlfa := chave.MustNova(chave.Chave{CUF: 35, Ano: 26, Mes: 7, CNPJ: cnpjAlfa, Modelo: 55, Serie: 1,
		Numero: 77, TpEmis: 1, CNF: "10203040"})
	c.InfCte.InfCTeNorm.InfDoc.InfNFe[0].Chave = nfeAlfa

	assinado, err := c.AssinarCom(certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjAlfa}))
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := c.Validar(); err != nil {
		t.Errorf("Validar recusou o CNPJ alfanumérico: %v", err)
	}
	if c.Chave()[6:20] != cnpjAlfa {
		t.Errorf("CNPJ na chave = %s, queria %s", c.Chave()[6:20], cnpjAlfa)
	}
	for _, trecho := range []string{"<CNPJ>" + cnpjAlfa + "</CNPJ>", "<chave>" + nfeAlfa + "</chave>"} {
		if !bytes.Contains(assinado, []byte(trecho)) {
			t.Errorf("o XML não traz %s", trecho)
		}
	}
	err = xsd.ValidarCTe(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("o CT-e com CNPJ alfanumérico não passa no esquema: %v", err)
	}
}
