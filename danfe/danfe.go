// Package danfe gera os documentos auxiliares em PDF: o DANFE da NF-e, em A4
// retrato, e o cupom da NFC-e, em bobina.
//
// O PDF é escrito em Go puro, com as fontes base-14 que todo leitor já possui,
// sem dependência gráfica e sem CGO.
//
// # Sobre a fidelidade ao manual
//
// O leiaute segue a estrutura de blocos do Manual de Especificação Técnica do
// DANFE — canhoto, identificação do emitente, destinatário, cálculo do imposto,
// transportador, itens e dados adicionais —, com o código de barras Code 128 da
// chave de acesso. Ele não é uma reprodução milimétrica do formulário oficial,
// e não passou por homologação visual em nenhuma SEFAZ. Para uso em produção,
// confira o resultado contra as exigências da sua unidade da federação.
//
// # QR Code da NFC-e
//
// A biblioteca produz o texto do QR Code, não a imagem — codificar QR é um
// domínio à parte, e uma implementação própria mal testada seria pior que
// nenhuma. Passe a matriz pronta em [Opcoes.QRCode], obtida com a biblioteca
// de QR de sua preferência.
package danfe

import (
	"fmt"
	"strings"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/internal/pdf"
	"github.com/sisgt-sistemas/gonfe/nfe"
	"github.com/sisgt-sistemas/gonfe/tipos"
	"github.com/sisgt-sistemas/gonfe/validacao"
)

// Opcoes ajusta a geração do documento auxiliar.
type Opcoes struct {
	// Orientacao define o sentido do DANFE da NF-e. O padrão é retrato.
	Orientacao Orientacao
	// SemCanhoto omite o recibo de entrega do topo. É o que se faz quando a
	// nota acompanha carga cujo canhoto vai em separado.
	SemCanhoto bool
	// Homologacao força a tarja de ambiente de teste. Quando falso, a tarja
	// aparece assim mesmo se o documento indicar homologação.
	Homologacao bool
	// Cancelada imprime a tarja de documento cancelado.
	Cancelada bool
	// QRCode é a matriz do QR Code da NFC-e, já codificada.
	QRCode pdf.MatrizQR
	// LarguraBobina é a largura do cupom da NFC-e em milímetros; o padrão é 80.
	LarguraBobina float64
	// Mensagem é um texto livre acrescentado ao rodapé do DANFE e do cupom da
	// NFC-e, para identificar o sistema emissor. Os documentos do transporte não
	// a imprimem: os manuais deles só admitem o que está no XML.
	Mensagem string
	// Papel é o tamanho da folha do DACTE: A4 retrato (o padrão) ou a meia
	// folha, A5 paisagem. Os demais documentos o ignoram e saem no formato deles.
	Papel Papel
	// CanhotoNaMeiaFolha imprime o canhoto no DACTE em meia folha, que por
	// padrão sai sem ele para caber numa folha só. [Opcoes.SemCanhoto] vale
	// para o A4.
	CanhotoNaMeiaFolha bool
}

// Papel é o tamanho da folha do DACTE.
type Papel int

const (
	// PapelA4 é o A4 em retrato, o padrão.
	PapelA4 Papel = iota
	// PapelA5 é a meia folha: A5 em paisagem, 210 × 148 mm, que o Manual do
	// DACTE admite (MOC CT-e 4.00, Anexo II, §2.15).
	PapelA5
)

// Orientacao é o sentido de impressão do DANFE.
type Orientacao int

const (
	// Retrato é a orientação padrão.
	Retrato Orientacao = iota
	// Paisagem gira a folha, útil em notas com muitos itens por linha.
	Paisagem
)

// MatrizQR reexporta o tipo da matriz de QR Code, para quem só importa este
// pacote.
type MatrizQR = pdf.MatrizQR

// Gerar produz o documento auxiliar apropriado ao modelo do documento: o DANFE
// para a NF-e e o cupom para a NFC-e.
//
// A entrada é o XML de distribuição, o nfeProc, que junta a nota assinada com o
// protocolo de autorização. Uma nota sem protocolo também é aceita, e o
// documento sai marcado como não autorizado.
func Gerar(procNFe []byte, opc Opcoes) ([]byte, error) {
	n, prot, err := nfe.LerNFeProc(procNFe)
	if err != nil {
		return nil, fmt.Errorf("danfe: %w", err)
	}
	switch n.Modelo() {
	case nfe.ModeloNFCe:
		return Cupom(n, prot, opc)
	default:
		return DANFE(n, prot, opc)
	}
}

// medidas reúne as dimensões do desenho, em milímetros, e as alturas dos
// quadros: as do A4 em [medidasDe] e as da meia folha em [medidasA5].
type medidas struct {
	largura, altura float64
	margem          float64
	util            float64 // largura útil, descontadas as margens

	// campo é a altura de uma linha de campos; secao, a da barra com o título
	// de um quadro; respiro, o espaço entre quadros.
	campo, secao, respiro float64
	// linhaParte é cada uma das três linhas de um participante, e parte, o
	// quadro inteiro, com o título.
	linhaParte, parte float64
	// linhaDoc é a altura de cada linha da tabela de documentos originários.
	linhaDoc float64
	// faixaIdentificacao, faixaChave e faixaSituacao são as faixas do
	// cabeçalho dos documentos do transporte.
	faixaIdentificacao, faixaChave, faixaSituacao float64
	// valor e valorForte são os corpos dos valores dos campos.
	valor, valorForte pdf.Estilo
	// compacta diz que é a meia folha: o cabeçalho e o canhoto se rearranjam
	// para caber na altura menor.
	compacta bool
}

// margem é a margem das páginas dos documentos auxiliares, em milímetros.
const margem = 5

func medidasDe(pg *pdf.Pagina) medidas {
	return medidas{
		largura: pg.Largura(),
		altura:  pg.Altura(),
		margem:  margem,
		util:    pg.Largura() - 2*margem,

		campo: 7, secao: 3.4, respiro: 1.5,
		linhaParte: 6, parte: alturaParte,
		linhaDoc:           alturaDocOrig,
		faixaIdentificacao: alturaFaixaIdentificacao, faixaChave: alturaFaixaChave, faixaSituacao: alturaSituacao,
		valor: valor, valorForte: valorForte,
	}
}

// medidasA5 são as medidas da meia folha. A largura é a do A4 em retrato, e a
// disposição dos quadros é a mesma; só as alturas diminuem (§2.16), menos a do
// código de barras e a do QR Code, que têm mínimo.
func medidasA5(pg *pdf.Pagina) medidas {
	m := medidasDe(pg)
	m.campo, m.secao, m.respiro = alturaCampoA5, 2.6, 0.6
	m.linhaParte = alturaCampoA5
	m.parte = m.secao + 3*m.linhaParte
	m.linhaDoc = 3
	m.faixaIdentificacao, m.faixaChave, m.faixaSituacao = 25, 17, 4.5
	m.valor = pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5}
	m.valorForte = pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5.5}
	m.compacta = true
	return m
}

// alturaCampoA5 é a linha de campos da meia folha. Abaixo de alturaCampoCompacto, a caixa usa o rótulo menor e
// encosta o valor no pé, para os dois caberem.
const (
	alturaCampoA5       = 4.4
	alturaCampoCompacto = 5
)

// posicoesNaCaixa devolve o estilo do rótulo e onde ficam o rótulo e o valor, a partir do topo da caixa.
func posicoesNaCaixa(altura float64, e pdf.Estilo) (estiloRotulo pdf.Estilo, yRotulo, yValor float64) {
	if altura < alturaCampoCompacto {
		return rotuloCompacto, 0.3, altura - e.Tamanho*mmPorPonto*0.8 - 0.5
	}
	return rotulo, 0.5, altura - e.Tamanho/2.4 - 1.2
}

// mmPorPonto converte o corpo da fonte, em pontos, para milímetros.
const mmPorPonto = 25.4 / 72

// Estilos usados no desenho. Os corpos seguem o que cabe em um A4 com a
// densidade de informação que o DANFE exige.
var (
	rotulo     = pdf.Estilo{Fonte: pdf.Normal, Tamanho: 4.5}
	valor      = pdf.Estilo{Fonte: pdf.Normal, Tamanho: 6.5}
	valorForte = pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6.5}
	titulo     = pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 9}
	miudo      = pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5}
	// rotuloCompacto é o rótulo dos campos da meia folha.
	rotuloCompacto = pdf.Estilo{Fonte: pdf.Normal, Tamanho: 3.8}
)

// caixa desenha um campo do formulário: uma moldura com o rótulo em cima e o
// valor embaixo.
func caixa(pg *pdf.Pagina, x, y, largura, altura float64, nome, conteudo string, e pdf.Estilo) {
	pg.Retangulo(x, y, largura, altura, 0.1)
	estiloRotulo, yRotulo, yValor := posicoesNaCaixa(altura, e)
	if nome != "" {
		pg.Texto(x+0.8, y+yRotulo, nome, estiloRotulo)
	}
	if conteudo != "" {
		texto := pdf.Encurtar(conteudo, e.Fonte, e.Tamanho, largura-1.6)
		pg.Texto(x+0.8, y+yValor, texto, e)
	}
}

// caixaCentro é como caixa, com o valor centralizado.
func caixaCentro(pg *pdf.Pagina, x, y, largura, altura float64, nome, conteudo string, e pdf.Estilo) {
	pg.Retangulo(x, y, largura, altura, 0.1)
	estiloRotulo, yRotulo, yValor := posicoesNaCaixa(altura, e)
	if nome != "" {
		pg.Texto(x+0.8, y+yRotulo, nome, estiloRotulo)
	}
	if conteudo != "" {
		texto := pdf.Encurtar(conteudo, e.Fonte, e.Tamanho, largura-1.6)
		pg.TextoCentralizado(x, largura, y+yValor, texto, e)
	}
}

// caixaDireita é como caixa, com o valor alinhado à direita — o formato dos
// campos monetários.
func caixaDireita(pg *pdf.Pagina, x, y, largura, altura float64, nome, conteudo string, e pdf.Estilo) {
	pg.Retangulo(x, y, largura, altura, 0.1)
	estiloRotulo, yRotulo, yValor := posicoesNaCaixa(altura, e)
	if nome != "" {
		pg.Texto(x+0.8, y+yRotulo, nome, estiloRotulo)
	}
	if conteudo != "" {
		texto := pdf.Encurtar(conteudo, e.Fonte, e.Tamanho, largura-1.6)
		pg.TextoDireita(x, largura-0.8, y+yValor, texto, e)
	}
}

// tarja imprime um aviso diagonal em cinza claro sobre a página.
func tarja(pg *pdf.Pagina, texto string) {
	e := pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 30, Cinza: 0.78}
	pg.TextoCentralizado(0, pg.Largura(), pg.Altura()/2-5, texto, e)
}

// formatarDocumento apresenta CNPJ ou CPF com máscara.
func formatarDocumento(cnpj, cpf string) string {
	switch {
	case cnpj != "":
		return validacao.FormatarCNPJ(cnpj)
	case cpf != "":
		return validacao.FormatarCPF(cpf)
	default:
		return ""
	}
}

// moeda apresenta um decimal no formato brasileiro, com separador de milhar.
func moeda(d tipos.Decimal) string {
	return separarMilhar(d.ComCasas(2).String())
}

// quantidade apresenta um decimal com até quatro casas, sem zeros à direita
// desnecessários.
func quantidade(d tipos.Decimal) string {
	s := d.ComCasas(4).String()
	if strings.Contains(s, ".") {
		s = strings.TrimRight(s, "0")
		s = strings.TrimSuffix(s, ".")
	}
	return separarMilhar(s)
}

// separarMilhar converte "1234.56" em "1.234,56".
func separarMilhar(s string) string {
	negativo := strings.HasPrefix(s, "-")
	s = strings.TrimPrefix(s, "-")

	inteiro, fracao, _ := strings.Cut(s, ".")
	var b strings.Builder
	for i, d := range inteiro {
		if i > 0 && (len(inteiro)-i)%3 == 0 {
			b.WriteByte('.')
		}
		b.WriteRune(d)
	}
	saida := b.String()
	if fracao != "" {
		saida += "," + fracao
	}
	if negativo {
		saida = "-" + saida
	}
	return saida
}

// dataHora apresenta uma data e hora no formato brasileiro.
func dataHora(d tipos.DataHora) string {
	if d.Vazia() {
		return ""
	}
	return d.Format("02/01/2006 15:04:05")
}

// data apresenta apenas a data.
func data(d tipos.DataHora) string {
	if d.Vazia() {
		return ""
	}
	return d.Format("02/01/2006")
}

// dataSimples apresenta uma data de calendário no formato brasileiro.
func dataSimples(d tipos.Data) string {
	if d.Vazia() {
		return ""
	}
	return d.Format("02/01/2006")
}

// hora apresenta apenas a hora.
func hora(d tipos.DataHora) string {
	if d.Vazia() {
		return ""
	}
	return d.Format("15:04:05")
}

// Os campos opcionais do leiaute chegam como ponteiro; estes três evitam a
// verificação de nulo espalhada pelo desenho.

func dataOpcional(d *tipos.DataHora) string {
	if d == nil {
		return ""
	}
	return data(*d)
}

func horaOpcional(d *tipos.DataHora) string {
	if d == nil {
		return ""
	}
	return hora(*d)
}

func dataHoraOpcional(d *tipos.DataHora) string {
	if d == nil {
		return ""
	}
	return dataHora(*d)
}

// enderecoEmUmaLinha junta logradouro, número e complemento.
func enderecoEmUmaLinha(lgr, nro, cpl string) string {
	partes := []string{lgr}
	if nro != "" {
		partes = append(partes, nro)
	}
	if cpl != "" {
		partes = append(partes, cpl)
	}
	return strings.Join(partes, ", ")
}

// chaveFormatada apresenta a chave em grupos de quatro dígitos.
func chaveFormatada(c string) string { return chave.Formatar(c) }

// naoAutorizada informa se falta protocolo de autorização.
func naoAutorizada(prot *nfe.ProtNFe) bool { return !prot.Autorizada() }
