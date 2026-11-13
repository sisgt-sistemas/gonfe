package danfe

import (
	"fmt"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/internal/pdf"
)

// Duas meias folhas cabem num A4 em retrato: 2 × 148 mm de altura, e o A4 tem 297. Cada uma vai em tamanho real, no
// meio da sua metade, com uma borda em volta (o DACTE menor que a folha sai delimitado) e uma linha tracejada de corte
// entre as duas.
const (
	metadeA4 = pdf.AlturaA4 / 2
	// recuoBorda é a distância da borda até a beira da meia folha: dentro da margem dela, longe do desenho.
	recuoBorda = 4.2
)

// CTeAutorizado é um conhecimento com o protocolo de autorização, que pode faltar.
type CTeAutorizado struct {
	CTe  *cte.CTe
	Prot *cte.ProtCTe
}

// GerarDACTEsEmA4 é como [DACTEsEmA4], a partir dos XML de distribuição, os cteProc.
func GerarDACTEsEmA4(procCTes [][]byte, opc Opcoes) ([]byte, error) {
	conhecimentos := make([]CTeAutorizado, 0, len(procCTes))
	for i, proc := range procCTes {
		c, prot, err := cte.LerCTeProc(proc)
		if err != nil {
			return nil, fmt.Errorf("dacte %d: %w", i+1, err)
		}
		conhecimentos = append(conhecimentos, CTeAutorizado{CTe: c, Prot: prot})
	}
	return DACTEsEmA4(conhecimentos, opc)
}

// DACTEsEmA4 gera os DACTEs em meia folha, duas por folha A4 em retrato, em tamanho real e na ordem dada. Uma linha
// tracejada marca o corte, e cada meia folha sai com uma borda. Um DACTE de mais de uma folha ocupa as metades
// seguintes; para duas vias do mesmo conhecimento, passe-o duas vezes.
//
// [Opcoes.Papel] é ignorado: as folhas são sempre as da meia folha. [Opcoes.CanhotoNaMeiaFolha] vale como em
// [DACTE].
func DACTEsEmA4(conhecimentos []CTeAutorizado, opc Opcoes) ([]byte, error) {
	if len(conhecimentos) == 0 {
		return nil, fmt.Errorf("dacte: nenhum conhecimento")
	}
	var meias []*pdf.Pagina
	for i, k := range conhecimentos {
		if k.CTe == nil {
			return nil, fmt.Errorf("dacte %d: conhecimento ausente", i+1)
		}
		documento, err := meiasFolhasDACTE(k.CTe, k.Prot, opc)
		if err != nil {
			return nil, fmt.Errorf("dacte %d: %w", i+1, err)
		}
		for p := range documento.Paginas() {
			meias = append(meias, documento.Pagina(p))
		}
	}

	saida := pdf.Novo()
	var folha *pdf.Pagina
	for i, meia := range meias {
		metade := i % 2
		if metade == 0 {
			folha = saida.NovaPaginaA4()
		}
		topo := float64(metade)*metadeA4 + (metadeA4-alturaA5)/2
		x := (pdf.LarguraA4 - larguraA5) / 2
		folha.Incluir(meia, x, topo)
		folha.Retangulo(x+recuoBorda, topo+recuoBorda, larguraA5-2*recuoBorda, alturaA5-2*recuoBorda, 0.2)
		if metade == 1 {
			linhaDeCorte(folha)
		}
	}
	return saida.Bytes()
}

// linhaDeCorte traceja o meio da folha A4, onde as duas meias folhas se separam.
func linhaDeCorte(folha *pdf.Pagina) {
	for x := 0.0; x < pdf.LarguraA4; x += 4 {
		folha.Linha(x, metadeA4, min(x+2, pdf.LarguraA4), metadeA4, 0.15)
	}
}
