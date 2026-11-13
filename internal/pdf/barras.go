package pdf

import (
	"fmt"
	"strconv"
)

// padroesCode128 traz, para cada valor do Code 128, as larguras dos seis
// elementos que o compõem — barra, espaço, barra, espaço, barra, espaço —, em
// módulos. O último padrão, de parada, tem sete elementos.
//
// Todo padrão de dado soma exatamente onze módulos; o de parada soma treze. O
// teste do pacote confere essas duas invariantes em toda a tabela, o que
// transforma um erro de transcrição em falha de teste em vez de código de
// barras ilegível.
var padroesCode128 = [107]string{
	"212222", "222122", "222221", "121223", "121322", "131222", "122213", "122312",
	"132212", "221213", "221312", "231212", "112232", "122132", "122231", "113222",
	"123122", "123221", "223211", "221132", "221231", "213212", "223112", "312131",
	"311222", "321122", "321221", "312212", "322112", "322211", "212123", "212321",
	"232121", "111323", "131123", "131321", "112313", "132113", "132311", "211313",
	"231113", "231311", "112133", "112331", "132131", "113123", "113321", "133121",
	"313121", "211331", "231131", "213113", "213311", "213131", "311123", "311321",
	"331121", "312113", "312311", "332111", "314111", "221411", "431111", "111224",
	"111422", "121124", "121421", "141122", "141221", "112214", "112412", "122114",
	"122411", "142112", "142211", "241211", "221114", "413111", "241112", "134111",
	"111242", "121142", "121241", "114212", "124112", "124211", "411212", "421112",
	"421211", "212141", "214121", "412121", "111143", "111341", "131141", "114113",
	"114311", "411113", "411311", "113141", "114131", "311141", "411131", "211412",
	"211214", "211232", "2331112",
}

const (
	// inicioCode128A e inicioCode128C começam o código no conjunto A (letras e
	// dígitos, um por símbolo) ou no C (dois dígitos por símbolo).
	inicioCode128A = 103
	inicioCode128C = 105
	// trocaParaC e trocaParaA mudam o conjunto no meio do código.
	trocaParaC = 99
	trocaParaA = 101
	// paradaCode128 encerra o código de barras.
	paradaCode128 = 106
)

// ValoresCode128 codifica o texto (dígitos e letras maiúsculas) nos valores do
// Code 128, com o início, o dígito de controle e a parada.
//
// Segue o padrão híbrido da NT 2025.001 §6, que a chave de acesso com CNPJ
// alfanumérico exige: começa no conjunto C, que codifica pares de dígitos; uma
// letra, ou um dígito ímpar que sobra antes dela, passa ao conjunto A; volta ao
// C diante de quatro ou mais dígitos seguidos (com quantidade ímpar, o primeiro
// fica no A) ou de um par de dígitos no fim. Uma chave só de dígitos sai
// inteira no conjunto C, como antes.
func ValoresCode128(texto string) ([]int, error) {
	if texto == "" {
		return nil, fmt.Errorf("pdf: código de barras vazio")
	}
	for i := 0; i < len(texto); i++ {
		if !ehDigito(texto[i]) && (texto[i] < 'A' || texto[i] > 'Z') {
			return nil, fmt.Errorf("pdf: %q fora do Code 128 da chave (dígitos e letras maiúsculas)", texto[i:i+1])
		}
	}
	digitosDesde := func(i int) int {
		n := 0
		for i+n < len(texto) && ehDigito(texto[i+n]) {
			n++
		}
		return n
	}

	// A chave começa por dígitos e sai no conjunto C; um texto que começa por letra
	// ou por um dígito só começa no A, sem gastar um símbolo de troca.
	conjuntoC := digitosDesde(0) >= 2
	valores := []int{inicioCode128A}
	if conjuntoC {
		valores[0] = inicioCode128C
	}
	for i := 0; i < len(texto); {
		seguidos := digitosDesde(i)
		if conjuntoC {
			for ; seguidos >= 2; seguidos -= 2 {
				par, _ := strconv.Atoi(texto[i : i+2])
				valores = append(valores, par)
				i += 2
			}
			if i < len(texto) {
				valores = append(valores, trocaParaA)
				conjuntoC = false
			}
			continue
		}
		parNoFim := seguidos >= 2 && seguidos%2 == 0 && i+seguidos == len(texto)
		if seguidos >= 4 || parNoFim {
			if seguidos%2 == 1 {
				valores = append(valores, int(texto[i])-32)
				i++
			}
			valores = append(valores, trocaParaC)
			conjuntoC = true
			continue
		}
		// No conjunto A, o valor de um caractere é o código ASCII menos 32.
		valores = append(valores, int(texto[i])-32)
		i++
	}

	// O dígito de controle é a soma ponderada dos valores, com peso igual à
	// posição a partir do símbolo de início, módulo 103.
	soma := valores[0]
	for i, v := range valores[1:] {
		soma += v * (i + 1)
	}
	return append(valores, soma%103, paradaCode128), nil
}

func ehDigito(c byte) bool { return c >= '0' && c <= '9' }

// CodigoDeBarras desenha a chave de acesso em Code 128, no padrão híbrido C e A
// de [ValoresCode128]: só dígitos sai no modo C, como o DANFE sempre exigiu; com
// as letras do CNPJ alfanumérico, alterna para o modo A.
func (pg *Pagina) CodigoDeBarras(posX, posY, largura, altura float64, texto string) error {
	valores, err := ValoresCode128(texto)
	if err != nil {
		return err
	}

	// A largura de um módulo sai da largura total dividida pelo número de
	// módulos do código inteiro.
	modulos := 0
	for _, v := range valores {
		for _, d := range padroesCode128[v] {
			modulos += int(d - '0')
		}
	}
	larguraModulo := largura / float64(modulos)

	cursor := posX
	for _, v := range valores {
		barra := true
		for _, d := range padroesCode128[v] {
			espessura := float64(d-'0') * larguraModulo
			if barra {
				pg.RetanguloPreenchido(cursor, posY, espessura, altura, 0)
			}
			cursor += espessura
			barra = !barra
		}
	}
	return nil
}

// MatrizQR é um QR Code já codificado, em que cada posição verdadeira é um
// módulo escuro. A biblioteca não codifica QR Codes — ela produz o texto que
// vai dentro deles — e recebe a matriz pronta de quem imprime.
//
// Com github.com/skip2/go-qrcode, por exemplo:
//
//	q, _ := qrcode.New(nfe.InfNFeSupl.QrCode, qrcode.Medium)
//	matriz := pdf.MatrizQR(q.Bitmap())
type MatrizQR [][]bool

// Vazia informa se não há matriz a desenhar.
func (m MatrizQR) Vazia() bool { return len(m) == 0 || len(m[0]) == 0 }

// QRCode desenha a matriz como um quadrado de lado igual ao tamanho informado.
func (pg *Pagina) QRCode(posX, posY, tamanho float64, m MatrizQR) {
	if m.Vazia() {
		return
	}
	lado := len(m)
	modulo := tamanho / float64(lado)
	for linha := range m {
		for coluna := range m[linha] {
			if !m[linha][coluna] {
				continue
			}
			// Os módulos são desenhados com uma fração a mais de tamanho para
			// que não sobre um fio branco entre eles no arredondamento.
			pg.RetanguloPreenchido(
				posX+float64(coluna)*modulo,
				posY+float64(linha)*modulo,
				modulo*1.02, modulo*1.02, 0)
		}
	}
}
