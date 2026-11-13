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

	// Disposição da faixa de identificação. Na meia folha, o nome do documento cabe numa linha e o emitente aperta
	// as linhas, para a faixa ter só a altura do QR Code com a margem.
	topoEmitLinhas, passoEmitLinhas, corpoEmit := 7.0, 3.3, 5.5
	topoCampos, linhaCampos := 15.0, 7.5
	if m.compacta {
		topoEmitLinhas, passoEmitLinhas, corpoEmit = 5.2, 2.9, 5
		topoCampos, linhaCampos = 10, 7.5
	}

	// Emitente.
	pg.Retangulo(m.margem, y, larguraEmit, m.faixaIdentificacao, 0.1)
	negrito := pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7.5}
	topoNome := 2.0
	if m.compacta {
		topoNome = 1.2
	}
	pg.TextoCentralizado(m.margem, larguraEmit, y+topoNome, pdf.Encurtar(id.emitNome, negrito.Fonte, negrito.Tamanho, larguraEmit-2), negrito)
	linha := pdf.Estilo{Fonte: pdf.Normal, Tamanho: corpoEmit}
	for i, texto := range id.emitLinhas {
		pg.TextoCentralizado(m.margem, larguraEmit, y+topoEmitLinhas+float64(i)*passoEmitLinhas,
			pdf.Encurtar(texto, linha.Fonte, linha.Tamanho, larguraEmit-2), linha)
	}

	// Documento.
	x := m.margem + larguraEmit
	pg.Retangulo(x, y, larguraCentro, topoCampos, 0.1)
	if m.compacta {
		pg.TextoCentralizado(x, larguraCentro, y+0.6, id.sigla, titulo)
		pg.TextoCentralizado(x, larguraCentro, y+4.2, id.nome1+" "+id.nome2, pdf.Estilo{Fonte: pdf.Normal, Tamanho: 4.5})
		pg.TextoCentralizado(x, larguraCentro, y+6.6, id.modal, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5.5})
	} else {
		pg.TextoCentralizado(x, larguraCentro, y+1.2, id.sigla, titulo)
		pg.TextoCentralizado(x, larguraCentro, y+5.6, id.nome1, miudo)
		pg.TextoCentralizado(x, larguraCentro, y+7.8, id.nome2, miudo)
		pg.TextoCentralizado(x, larguraCentro, y+10.8, id.modal, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6})
	}
	quarto := larguraCentro / 4
	for i, campo := range [][2]string{
		{"MODELO", id.modelo}, {"SÉRIE", id.serie}, {"NÚMERO", id.numero},
		{"FL", fmtFolha(id.folha, id.folhas)},
	} {
		caixaCentro(pg, x+float64(i)*quarto, y+topoCampos, quarto, linhaCampos, campo[0], campo[1], m.valorForte)
	}
	topoEmissao := topoCampos + linhaCampos
	caixa(pg, x, y+topoEmissao, larguraCentro*0.58, linhaCampos, "DATA E HORA DE EMISSÃO", id.emissao, m.valor)
	rotuloSuframa := "INSC. SUFRAMA DESTINATÁRIO"
	if id.rotuloSuframa != "" {
		rotuloSuframa = id.rotuloSuframa
	}
	caixa(pg, x+larguraCentro*0.58, y+topoEmissao, larguraCentro*0.42, linhaCampos, rotuloSuframa, id.suframa, m.valor)

	// QR Code.
	x += larguraCentro
	pg.Retangulo(x, y, ladoCaixaQR, m.faixaIdentificacao, 0.1)
	if matriz, err := matrizQR(id.qrCode); err == nil {
		pg.QRCode(x+(ladoCaixaQR-ladoQR)/2, y+(m.faixaIdentificacao-ladoQR)/2, ladoQR, matriz)
	} else {
		meio := m.faixaIdentificacao / 2
		pg.TextoCentralizado(x, ladoCaixaQR, y+meio-3, "QR CODE", miudo)
		pg.TextoCentralizado(x, ladoCaixaQR, y+meio-0.5, "NÃO INFORMADO", miudo)
	}
	y += m.faixaIdentificacao

	// Código de barras da chave: a altura das barras não diminui na meia folha (§2.16).
	pg.Retangulo(m.margem, y, larguraAreaBarras, m.faixaChave, 0.1)
	if len(id.chave) == 44 {
		if err := pg.CodigoDeBarras(m.margem+(larguraAreaBarras-larguraBarras)/2, y+(m.faixaChave-alturaBarras)/2,
			larguraBarras, alturaBarras, id.chave); err != nil {
			pg.TextoCentralizado(m.margem, larguraAreaBarras, y+m.faixaChave/2-1, "código de barras indisponível", miudo)
		}
	}
	x = m.margem + larguraAreaBarras
	resto := m.util - larguraAreaBarras
	alturaChave, alturaConsulta, topoConsulta := 7.0, 5.5, 1.4
	if m.compacta {
		alturaChave, alturaConsulta, topoConsulta = 6, 4.5, 1.2
	}
	caixaCentro(pg, x, y, resto, alturaChave, "CHAVE DE ACESSO", chaveFormatada(id.chave), pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7})
	pg.Retangulo(x, y+alturaChave, resto, alturaConsulta, 0.1)
	pg.TextoCentralizado(x, resto, y+alturaChave+topoConsulta, "Consulta em "+id.consulta, miudo)
	topoProtocolo := alturaChave + alturaConsulta
	caixaCentro(pg, x, y+topoProtocolo, resto, m.faixaChave-topoProtocolo, "PROTOCOLO DE AUTORIZAÇÃO DE USO", id.protocolo, m.valorForte)
	y += m.faixaChave

	if id.situacao != "" {
		corpo, topo := 8.0, 1.6
		if m.compacta {
			corpo, topo = 7, 1.0
		}
		pg.RetanguloPreenchido(m.margem, y, m.util, m.faixaSituacao, 0.9)
		pg.Retangulo(m.margem, y, m.util, m.faixaSituacao, 0.1)
		pg.TextoCentralizado(m.margem, m.util, y+topo, id.situacao, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: corpo})
		y += m.faixaSituacao
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
	pg.RetanguloPreenchido(m.margem, y, m.util, m.secao, 0.88)
	pg.Retangulo(m.margem, y, m.util, m.secao, 0.1)
	pg.Texto(m.margem+0.8, y+m.topoTitulo(), nome, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5})
	return y + m.secao
}

// topoTitulo é onde começa o texto de uma barra de título, a partir do topo dela.
func (m medidas) topoTitulo() float64 {
	if m.compacta {
		return 0.3
	}
	return 0.4
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
