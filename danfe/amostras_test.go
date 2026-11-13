package danfe_test

import (
	"flag"
	"os"
	"path/filepath"
	"testing"

	"github.com/sisgt-sistemas/gonfe/danfe"
)

// amostras é a pasta onde TestGerarAmostras grava os PDFs dos exemplos: `make amostras`. Vazia, o teste é pulado.
var amostras = flag.String("amostras", "", "pasta onde gravar os PDFs de amostra do DACTE, do DACTE OS e do DAMDFE")

// TestGerarAmostras grava os documentos auxiliares dos exemplos, curtos e longos, para comparar com o Manual de
// Orientação. Não confere nada: é a vitrine que se olha antes de mexer no leiaute.
func TestGerarAmostras(t *testing.T) {
	if *amostras == "" {
		t.Skip("rode com -amostras=<pasta> (make amostras)")
	}
	if err := os.MkdirAll(*amostras, 0o755); err != nil { //nolint:gosec // Pasta de amostras, de quem roda o teste.
		t.Fatal(err)
	}
	gerar := func(nome string, pdf []byte, err error) {
		t.Helper()
		if err != nil {
			t.Fatalf("%s: %v", nome, err)
		}
		if err := os.WriteFile(filepath.Join(*amostras, nome), pdf, 0o644); err != nil { //nolint:gosec // Amostra.
			t.Fatal(err)
		}
	}
	for nome, documentos := range map[string]int{"dacte.pdf": 1, "dacte-muitos-documentos.pdf": 40} {
		pdf, err := danfe.GerarDACTE(procCTeDe(t, conhecimentoExemplo(documentos)), danfe.Opcoes{})
		gerar(nome, pdf, err)
	}
	for nome, documentos := range map[string]int{"dacte-meia-folha.pdf": 4, "dacte-meia-folha-muitos-documentos.pdf": 40} {
		pdf, err := danfe.GerarDACTE(procCTeDe(t, conhecimentoExemplo(documentos)), danfe.Opcoes{Papel: danfe.PapelA5})
		gerar(nome, pdf, err)
	}
	{
		var procs [][]byte
		for _, documentos := range []int{4, 40, 1} {
			procs = append(procs, procCTeDe(t, conhecimentoExemplo(documentos)))
		}
		pdf, err := danfe.GerarDACTEsEmA4(procs, danfe.Opcoes{})
		gerar("dacte-meia-folha-duas-por-a4.pdf", pdf, err)
	}
	for nome, referenciados := range map[string]int{"dacte-os.pdf": 1, "dacte-os-muitos-documentos.pdf": 40} {
		pdf, err := danfe.GerarDACTEOS(procOSDe(t, fretamentoExemplo(referenciados)), danfe.Opcoes{})
		gerar(nome, pdf, err)
	}
	for nome, entregas := range map[string]int{"dacte-simplificado.pdf": 3, "dacte-simplificado-muitas-entregas.pdf": 40} {
		pdf, err := danfe.GerarDACTESimp(procSimpDe(t, simplificadoExemplo(entregas)), danfe.Opcoes{})
		gerar(nome, pdf, err)
	}
	for nome, notas := range map[string]int{"damdfe.pdf": 2, "damdfe-muitas-notas.pdf": 60} {
		pdf, err := danfe.GerarDAMDFE(procMDFeDe(t, manifestoExemplo(notas)), danfe.Opcoes{})
		gerar(nome, pdf, err)
	}
}
