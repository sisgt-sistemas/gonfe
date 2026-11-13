package mdfe_test

import (
	"bytes"
	"errors"
	"strings"
	"testing"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/xsd"
)

// cnpjAlfa é o CNPJ alfanumérico do exemplo da NT 2025.001 (12.ABC.345/01DE-35).
const cnpjAlfa = "12ABC34501DE35"

// Um MDF-e que leva a NF-e de um emitente com CNPJ alfanumérico: a chave com letras chega ao XML e o esquema oficial
// a aceita (TChNFe).
func TestManifestoLevaNFeDeCNPJAlfanumerico(t *testing.T) {
	m := manifestoExemplo()
	nfeAlfa := chave.MustNova(chave.Chave{CUF: 35, Ano: 26, Mes: 7, CNPJ: cnpjAlfa, Modelo: 55, Serie: 1,
		Numero: 78, TpEmis: 1, CNF: "10203041"})
	m.InfMDFe.InfDoc.InfMunDescarga[0].InfNFe[0].ChNFe = nfeAlfa

	assinado, err := m.AssinarCom(certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora}))
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := m.Validar(); err != nil {
		t.Errorf("Validar recusou a chave alfanumérica: %v", err)
	}
	if !bytes.Contains(assinado, []byte("<chNFe>"+nfeAlfa+"</chNFe>")) {
		t.Errorf("o XML não traz a chave %s", nfeAlfa)
	}
	err = xsd.ValidarMDFe(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("o MDF-e com chave alfanumérica não passa no esquema: %v", err)
	}
}

// O esquema embutido (PL_MDFe_300b, NT 2025.001 v1.04, o mais novo em 2026-11-13) ainda restringe o Id do infMDFe a MDFe[0-9]{44}: um MDF-e de
// emitente com CNPJ alfanumérico é recusado, embora a biblioteca o monte. Quando o pacote de esquemas for atualizado
// e este teste falhar, o emitente alfanumérico passa a valer também no MDF-e: troque o teste pelo positivo.
func TestEsquemaDoMDFeAindaRecusaEmitenteAlfanumerico(t *testing.T) {
	m := manifestoExemplo()
	m.InfMDFe.Emit.CNPJ = "12.abc.345/01de-35"
	assinado, err := m.AssinarCom(certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjAlfa}))
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if !strings.Contains(m.InfMDFe.Id, cnpjAlfa) {
		t.Errorf("Id = %s, queria o CNPJ %s", m.InfMDFe.Id, cnpjAlfa)
	}
	err = xsd.ValidarMDFe(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if !errors.Is(err, xsd.ErrInvalido) || !strings.Contains(err.Error(), "'Id'") {
		t.Fatalf("o esquema do MDF-e mudou quanto ao Id alfanumérico: %v", err)
	}
}
