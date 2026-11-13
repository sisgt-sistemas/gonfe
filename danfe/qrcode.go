package danfe

import (
	"fmt"

	"rsc.io/qr"

	"github.com/sisgt-sistemas/gonfe/internal/pdf"
)

// matrizQR codifica o texto do QR Code de consulta (qrCodCTe, qrCodMDFe) na matriz que a página desenha. A margem
// clara em volta fica por conta de quem posiciona o desenho.
func matrizQR(texto string) (pdf.MatrizQR, error) {
	if texto == "" {
		return nil, fmt.Errorf("danfe: o documento não traz o texto do QR Code")
	}
	// Nível M de correção: o manual não fixa o nível, e o M é o usual em documento impresso.
	c, err := qr.Encode(texto, qr.M)
	if err != nil {
		return nil, fmt.Errorf("danfe: codificar o QR Code: %w", err)
	}
	m := make(pdf.MatrizQR, c.Size)
	for y := range m {
		m[y] = make([]bool, c.Size)
		for x := range m[y] {
			m[y][x] = c.Black(x, y)
		}
	}
	return m, nil
}
