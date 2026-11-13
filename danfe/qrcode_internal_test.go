package danfe

import (
	"testing"
)

// A matriz tem as três marcas de posição do QR Code nos cantos (quadrado escuro de 7 módulos com anel claro e centro
// escuro de 3): prova que a codificação saiu no formato, sem precisar de um leitor.
func TestMatrizQRTemAsMarcasDePosicao(t *testing.T) {
	const texto = "https://dfe-portal.svrs.rs.gov.br/cte/qrCode?chCTe=43260312345678000195570010000009871112233442&tpAmb=2"
	m, err := matrizQR(texto)
	if err != nil {
		t.Fatal(err)
	}
	n := len(m)
	if n < 21 || (n-21)%4 != 0 {
		t.Fatalf("lado %d não é de uma versão de QR Code (21 + 4k)", n)
	}
	marca := func(x0, y0 int) bool {
		for y := 0; y < 7; y++ {
			for x := 0; x < 7; x++ {
				borda := x == 0 || y == 0 || x == 6 || y == 6
				centro := x >= 2 && x <= 4 && y >= 2 && y <= 4
				if m[y0+y][x0+x] != (borda || centro) {
					return false
				}
			}
		}
		return true
	}
	for _, canto := range [][2]int{{0, 0}, {n - 7, 0}, {0, n - 7}} {
		if !marca(canto[0], canto[1]) {
			t.Errorf("sem marca de posição em %v", canto)
		}
	}
	if _, err := matrizQR(""); err == nil {
		t.Error("texto vazio foi aceito")
	}
}
