package danfe

import (
	"errors"
	"fmt"

	"github.com/sisgt-sistemas/gonfe/internal/pdf"
)

// O cabeçalho dos documentos auxiliares do CT-e segue o MOC CT-e 4.00, Anexo II (Manual do DACTE): emitente,
// identificação do documento e QR Code numa faixa; código de barras da chave, a chave, a consulta e o protocolo na
// outra. Vale para o DACTE, o DACTE OS e o DACTE do CT-e Simplificado, e se repete no alto de cada folha.
const (
	// alturaFaixaIdentificacao é a faixa do emitente, do documento e do QR Code.
	alturaFaixaIdentificacao = 30
	// alturaFaixaChave é a faixa do código de barras, da chave e do protocolo.
	alturaFaixaChave = 20
	// ladoCaixaQR é o quadro do QR Code; o desenho tem ladoQR e o resto é a zona de silêncio. O manual pede no
	// mínimo 25 mm: 22 de conteúdo e 3 de margem (§2.18.1).
	ladoCaixaQR = 30
	ladoQR      = 22
	// larguraAreaBarras é a área do código de barras (o manual reserva 9 cm, §2.17); larguraBarras é o desenho, com
	// a margem clara dos lados, e alturaBarras fica entre 15 e 25 mm.
	larguraAreaBarras = 92
	larguraBarras     = 80
	alturaBarras      = 15
	// alturaSituacao é a faixa do aviso de homologação, cancelamento ou contingência.
	alturaSituacao = 6
	// URLConsultaCTe é o portal de consulta impresso no campo 1 da área variável (§2.19).
	URLConsultaCTe = "https://www.cte.fazenda.gov.br/portal"
)

// identificacao reúne o que o cabeçalho imprime, já formatado.
type identificacao struct {
	// sigla e nome do documento: "DACTE", "Documento Auxiliar do Conhecimento de Transporte Eletrônico".
	sigla, nome1, nome2 string
	modal               string
	// emitente: nome em destaque e as linhas de endereço, telefone e documentos.
	emitNome   string
	emitLinhas []string
	modelo     string
	serie      string
	numero     string
	emissao    string
	suframa    string
	// rotuloSuframa nomeia de quem é a inscrição na SUFRAMA; vazio, a do destinatário.
	rotuloSuframa string
	folha         int
	folhas        int
	chave         string
	protocolo     string
	consulta      string
	qrCode        string
	// situacao é o aviso da área variável: homologação, cancelado, sem protocolo, contingência; vazio, nada.
	situacao string
}

// desenharIdentificacao imprime as duas faixas do cabeçalho e, se houver, a do aviso de situação. Devolve a altura
// seguinte.
func desenharIdentificacao(pg *pdf.Pagina, m medidas, id identificacao, y float64) float64 {
	larguraCentro := 62.0
	larguraEmit := m.util - larguraCentro - ladoCaixaQR

	// Emitente.
	pg.Retangulo(m.margem, y, larguraEmit, alturaFaixaIdentificacao, 0.1)
	negrito := pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7.5}
	pg.TextoCentralizado(m.margem, larguraEmit, y+2, pdf.Encurtar(id.emitNome, negrito.Fonte, negrito.Tamanho, larguraEmit-2), negrito)
	linha := pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5}
	for i, texto := range id.emitLinhas {
		pg.TextoCentralizado(m.margem, larguraEmit, y+7+float64(i)*3.3,
			pdf.Encurtar(texto, linha.Fonte, linha.Tamanho, larguraEmit-2), linha)
	}

	// Documento.
	x := m.margem + larguraEmit
	pg.Retangulo(x, y, larguraCentro, 15, 0.1)
	pg.TextoCentralizado(x, larguraCentro, y+1.2, id.sigla, titulo)
	pg.TextoCentralizado(x, larguraCentro, y+5.6, id.nome1, miudo)
	pg.TextoCentralizado(x, larguraCentro, y+7.8, id.nome2, miudo)
	pg.TextoCentralizado(x, larguraCentro, y+10.8, id.modal, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6})
	quarto := larguraCentro / 4
	for i, campo := range [][2]string{
		{"MODELO", id.modelo}, {"SÉRIE", id.serie}, {"NÚMERO", id.numero},
		{"FL", fmtFolha(id.folha, id.folhas)},
	} {
		caixaCentro(pg, x+float64(i)*quarto, y+15, quarto, 7.5, campo[0], campo[1], valorForte)
	}
	caixa(pg, x, y+22.5, larguraCentro*0.58, 7.5, "DATA E HORA DE EMISSÃO", id.emissao, valor)
	rotuloSuframa := "INSC. SUFRAMA DESTINATÁRIO"
	if id.rotuloSuframa != "" {
		rotuloSuframa = id.rotuloSuframa
	}
	caixa(pg, x+larguraCentro*0.58, y+22.5, larguraCentro*0.42, 7.5, rotuloSuframa, id.suframa, valor)

	// QR Code.
	x += larguraCentro
	pg.Retangulo(x, y, ladoCaixaQR, alturaFaixaIdentificacao, 0.1)
	if matriz, err := matrizQR(id.qrCode); err == nil {
		pg.QRCode(x+(ladoCaixaQR-ladoQR)/2, y+(alturaFaixaIdentificacao-ladoQR)/2, ladoQR, matriz)
	} else {
		pg.TextoCentralizado(x, ladoCaixaQR, y+12, "QR CODE", miudo)
		pg.TextoCentralizado(x, ladoCaixaQR, y+14.5, "NÃO INFORMADO", miudo)
	}
	y += alturaFaixaIdentificacao

	// Código de barras da chave.
	pg.Retangulo(m.margem, y, larguraAreaBarras, alturaFaixaChave, 0.1)
	if len(id.chave) == 44 {
		if err := pg.CodigoDeBarras(m.margem+(larguraAreaBarras-larguraBarras)/2, y+(alturaFaixaChave-alturaBarras)/2,
			larguraBarras, alturaBarras, id.chave); err != nil {
			pg.TextoCentralizado(m.margem, larguraAreaBarras, y+9, "código de barras indisponível", miudo)
		}
	}
	x = m.margem + larguraAreaBarras
	resto := m.util - larguraAreaBarras
	caixaCentro(pg, x, y, resto, 7, "CHAVE DE ACESSO", chaveFormatada(id.chave), pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7})
	pg.Retangulo(x, y+7, resto, 5.5, 0.1)
	pg.TextoCentralizado(x, resto, y+8.4, "Consulta em "+id.consulta, miudo)
	caixaCentro(pg, x, y+12.5, resto, alturaFaixaChave-12.5, "PROTOCOLO DE AUTORIZAÇÃO DE USO", id.protocolo, valorForte)
	y += alturaFaixaChave

	if id.situacao != "" {
		pg.RetanguloPreenchido(m.margem, y, m.util, alturaSituacao, 0.9)
		pg.Retangulo(m.margem, y, m.util, alturaSituacao, 0.1)
		pg.TextoCentralizado(m.margem, m.util, y+1.6, id.situacao, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 8})
		y += alturaSituacao
	}
	return y
}

func fmtFolha(folha, folhas int) string {
	if folhas == 0 {
		return ""
	}
	return fmt.Sprintf("%d/%d", folha, folhas)
}

// cabecalhoDeSecao desenha a barra cinza com o título de um quadro e devolve a altura seguinte.
func cabecalhoDeSecao(pg *pdf.Pagina, m medidas, y float64, nome string) float64 {
	pg.RetanguloPreenchido(m.margem, y, m.util, 3.4, 0.88)
	pg.Retangulo(m.margem, y, m.util, 3.4, 0.1)
	pg.Texto(m.margem+0.8, y+0.4, nome, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5})
	return y + 3.4
}

// quadroDeTexto desenha um quadro com rótulo e texto quebrado em até maxLinhas, com altura mínima, e devolve a
// altura seguinte.
func quadroDeTexto(pg *pdf.Pagina, x, y, largura, minimo float64, nome, texto string, maxLinhas int) float64 {
	altura := minimo
	if texto != "" {
		if usada := pg.TextoQuebrado(x+0.8, y+4, largura-1.6, texto, miudo, maxLinhas) + 5; usada > altura {
			altura = usada
		}
	}
	pg.Retangulo(x, y, largura, altura, 0.1)
	pg.Texto(x+0.8, y+0.5, nome, rotulo)
	return y + altura
}

// ErrNaoCabe diz que os quadros fixos da primeira folha não cabem numa folha A4: componentes ou quantidades
// demais para o leiaute do manual, que não os leva a outra folha.
var ErrNaoCabe = errors.New("os quadros fixos não cabem na primeira folha")

// cabeNaFolha confere a medição da primeira folha: o que passa da margem inferior seria cortado sem aviso.
func cabeNaFolha(fim float64, m medidas) error {
	if fim > m.altura-m.margem {
		return fmt.Errorf("%w: o conteúdo vai a %.0f mm, e a folha acaba em %.0f mm", ErrNaoCabe, fim, m.altura-m.margem)
	}
	return nil
}

// emRetrato devolve as opções com a folha em retrato. Os documentos do transporte saem sempre assim: o leiaute do
// manual em A4 paisagem não comporta os quadros fixos da primeira folha.
func emRetrato(opc Opcoes) Opcoes {
	opc.Orientacao = Retrato
	return opc
}
