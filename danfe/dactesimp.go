package danfe

import (
	"fmt"
	"strings"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/ctesimp"
	"github.com/sisgt-sistemas/gonfe/internal/pdf"
	"github.com/sisgt-sistemas/gonfe/tipos"
)

// GerarDACTESimp produz o documento auxiliar a partir do XML de distribuição do
// CT-e Simplificado, o cteSimpProc.
func GerarDACTESimp(procCTeSimp []byte, opc Opcoes) ([]byte, error) {
	c, prot, err := ctesimp.LerCTeSimpProc(procCTeSimp)
	if err != nil {
		return nil, fmt.Errorf("dacte simplificado: %w", err)
	}
	return DACTESimp(c, prot, opc)
}

// DACTESimp gera o DACTE do CT-e Simplificado, em A4.
//
// Nem o Manual do DACTE (2022) nem a NT 2024.002 definem o leiaute dele. Este
// segue as regras gerais do manual — o mesmo cabeçalho com QR Code, código de
// barras, consulta e protocolo, folhas numeradas e o aviso de homologação na
// área variável — e, no lugar de remetente e destinatário, que o Simplificado não
// tem, traz a tabela das entregas: item, origem, destino, valores e os documentos
// de cada uma. A tabela continua nas folhas seguintes.
func DACTESimp(c *ctesimp.CTeSimp, prot *cte.ProtCTe, opc Opcoes) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("dacte simplificado: conhecimento ausente")
	}
	opc = emRetrato(opc)
	paginas, linhas, err := paginarEntregas(c, prot, opc)
	if err != nil {
		return nil, fmt.Errorf("dacte simplificado: %w", err)
	}
	documento := pdf.Novo()
	for i, entregas := range paginas {
		pg := novaFolha(documento, opc)
		m := medidasDe(pg)
		if i == 0 {
			desenharPrimeiraFolhaSimp(pg, m, c, prot, opc, entregas, linhas[0], len(paginas))
			continue
		}
		y := desenharCabecalhoSimp(pg, m, c, prot, opc, i+1, len(paginas), m.margem)
		desenharEntregas(pg, m, entregas, linhas[1], y)
	}
	return documento.Bytes()
}

func desenharPrimeiraFolhaSimp(pg *pdf.Pagina, m medidas, c *ctesimp.CTeSimp, prot *cte.ProtCTe, opc Opcoes, entregas []linhaEntrega, linhas, folhas int) float64 {
	y := desenharCabecalhoSimp(pg, m, c, prot, opc, 1, folhas, m.margem)
	ide := &c.InfCte.Ide
	caixa(pg, m.margem, y, m.util*0.5, 7, "UF DE INÍCIO DA PRESTAÇÃO", ide.UFIni, valor)
	caixa(pg, m.margem+m.util*0.5, y, m.util*0.5, 7, "UF DE TÉRMINO DA PRESTAÇÃO", ide.UFFim, valor)
	y += 8.5
	t := &c.InfCte.Toma
	desenharParte(pg, m.margem, y, m.util, "TOMADOR DO SERVIÇO", &parte{
		nome: t.XNome, documento: formatarDocumento(t.CNPJ, t.CPF), ie: t.IE, fone: t.Fone, endereco: &t.EnderToma,
	})
	y += alturaParte + 1.5
	y = desenharCarga(pg, m, &c.InfCte.InfCarga, y)
	y = desenharICMS(pg, m, &c.InfCte.Imp.ICMS, y)
	y = desenharIBSCBSTotal(pg, m, c.InfCte.Total.IBSCBSTot, c.InfCte.Total.VTotDFe, y)
	y = desenharEntregas(pg, m, entregas, linhas, y)
	caixaDireita(pg, m.margem, y-1.5, m.util*0.5, 7, "VALOR TOTAL DA PRESTAÇÃO DO SERVIÇO", moeda(c.InfCte.Total.VTPrest), valorForte)
	caixaDireita(pg, m.margem+m.util*0.5, y-1.5, m.util*0.5, 7, "VALOR TOTAL A RECEBER", moeda(c.InfCte.Total.VTRec), valorForte)
	y += 7
	y = desenharFluxo(pg, m, fluxoDoSimp(c.InfCte.Compl), y)
	gerais, emissor, fisco := observacoesDoSimp(c.InfCte.Compl)
	y = quadroDeTexto(pg, m.margem, y, m.util, 10, "OBSERVAÇÕES", gerais, 4)
	if c.InfCte.InfModal.Rodo != nil {
		y = cabecalhoDeSecao(pg, m, y, "INFORMAÇÕES ESPECÍFICAS DO MODAL RODOVIÁRIO")
		caixa(pg, m.margem, y, m.util, 7, "RNTRC DA EMPRESA", c.InfCte.InfModal.Rodo.RNTRC, valor)
		y += 7
	}
	fimEmissor := quadroDeTexto(pg, m.margem, y, m.util*0.5, 10, "USO EXCLUSIVO DO EMISSOR DO CT-e", emissor, 3)
	fimFisco := quadroDeTexto(pg, m.margem+m.util*0.5, y, m.util*0.5, 10, "RESERVADO AO FISCO", fisco, 3)
	y = max(fimEmissor, fimFisco) + 1.5
	return desenharAdicionaisSimp(pg, m, c, y)
}

// desenharCabecalhoSimp imprime o cabeçalho comum e as linhas do tipo e da
// natureza da operação.
func desenharCabecalhoSimp(pg *pdf.Pagina, m medidas, c *ctesimp.CTeSimp, prot *cte.ProtCTe, opc Opcoes, folha, folhas int, y float64) float64 {
	ide := &c.InfCte.Ide
	emit := &c.InfCte.Emit
	qrCode := ""
	if c.InfCTeSupl != nil {
		qrCode = c.InfCTeSupl.QrCodCTe
	}
	y = desenharIdentificacao(pg, m, identificacao{
		sigla: "DACTE", nome1: "Documento Auxiliar do Conhecimento", nome2: "de Transporte Eletrônico - Simplificado",
		modal:    "MODAL " + strings.ToUpper(ide.Modal.Descricao()),
		emitNome: emit.XNome, emitLinhas: linhasDoEmitente(emit.XFant, emit.EnderEmit, emit.CNPJ, emit.CPF, emit.IE),
		modelo: string(ide.Mod), serie: fmt.Sprintf("%03d", ide.Serie), numero: fmt.Sprintf("%09d", ide.NCT),
		emissao: dataHora(ide.DhEmi), suframa: c.InfCte.Toma.ISUF, rotuloSuframa: "INSC. SUFRAMA TOMADOR", folha: folha, folhas: folhas,
		chave: c.Chave(), protocolo: protocoloCTeDescrito(prot), consulta: URLConsultaCTe, qrCode: qrCode,
		situacao: situacaoCTe(ide.TpAmb, ide.TpEmis, prot != nil && prot.Autorizado(), opc, "CT-e"),
	}, y)
	caixa(pg, m.margem, y, m.util*0.30, 7, "TIPO DO CT-e", tipoDoSimp(ide.TpCTe), valor)
	caixa(pg, m.margem+m.util*0.30, y, m.util*0.25, 7, "TIPO DO SERVIÇO", tipoDoServico(ide.TpServ), valor)
	caixa(pg, m.margem+m.util*0.55, y, m.util*0.45, 7, "CFOP - NATUREZA DA OPERAÇÃO", ide.CFOP+" - "+ide.NatOp, valor)
	return y + 7 + 1.5
}

func tipoDoSimp(t ctesimp.TipoCTe) string {
	switch t {
	case ctesimp.Simplificado:
		return "5 - SIMPLIFICADO"
	case ctesimp.SubstituicaoSimplificado:
		return "6 - SUBSTITUIÇÃO DO SIMPLIFICADO"
	default:
		return string(t)
	}
}

// linhaEntrega é uma linha da tabela de entregas: um documento de uma entrega.
// Os dados da entrega saem só na primeira linha dela.
type linhaEntrega struct {
	item, origem, destino, valor, receber, tipo, chave string
}

// linhasDeEntrega achata as entregas (det) em linhas, uma por documento
// transportado ou anterior.
func linhasDeEntrega(c *ctesimp.CTeSimp) []linhaEntrega {
	var linhas []linhaEntrega
	for _, det := range c.InfCte.Det {
		cabeca := linhaEntrega{
			item:    fmt.Sprintf("%d", det.NItem),
			origem:  det.XMunIni,
			destino: det.XMunFim,
			valor:   moeda(det.VPrest),
			receber: moeda(det.VRec),
		}
		var docs [][2]string
		for _, n := range det.InfNFe {
			docs = append(docs, [2]string{"NF-e", chaveFormatada(n.ChNFe)})
		}
		for _, a := range det.InfDocAnt {
			docs = append(docs, [2]string{"CT-e ANT.", chaveFormatada(a.ChCTe)})
		}
		if len(docs) == 0 {
			linhas = append(linhas, cabeca)
			continue
		}
		for i, d := range docs {
			linha := linhaEntrega{tipo: d[0], chave: d[1]}
			if i == 0 {
				linha.item, linha.origem, linha.destino, linha.valor, linha.receber = cabeca.item, cabeca.origem, cabeca.destino, cabeca.valor, cabeca.receber
			}
			linhas = append(linhas, linha)
		}
	}
	return linhas
}

// paginarEntregas mede, como o DACTE, quantas linhas de entregas cabem na
// primeira folha e nas demais.
func paginarEntregas(c *ctesimp.CTeSimp, prot *cte.ProtCTe, opc Opcoes) ([][]linhaEntrega, [2]int, error) {
	todas := linhasDeEntrega(c)
	rascunho := pdf.Novo()
	pg := novaFolha(rascunho, opc)
	m := medidasDe(pg)
	fim := desenharPrimeiraFolhaSimp(pg, m, c, prot, opc, nil, 1, 1)
	if err := cabeNaFolha(fim, m); err != nil {
		return nil, [2]int{}, err
	}
	naPrimeira := 1 + max(int((m.altura-m.margem-fim)/alturaDocOrig), 0)
	topo := desenharCabecalhoSimp(novaFolha(rascunho, opc), m, c, prot, opc, 2, 2, m.margem) + 3.4*2
	nasDemais := max(int((m.altura-m.margem-topo)/alturaDocOrig), 1)

	linhas := [2]int{naPrimeira, nasDemais}
	if len(todas) <= naPrimeira {
		return [][]linhaEntrega{todas}, linhas, nil
	}
	paginas := [][]linhaEntrega{todas[:naPrimeira]}
	for corte := naPrimeira; corte < len(todas); corte += nasDemais {
		paginas = append(paginas, todas[corte:min(corte+nasDemais, len(todas))])
	}
	return paginas, linhas, nil
}

// desenharEntregas imprime a tabela das entregas, com pelo menos minLinhas linhas.
func desenharEntregas(pg *pdf.Pagina, m medidas, linhas []linhaEntrega, minLinhas int, y float64) float64 {
	colunas := []struct {
		nome    string
		fracao  float64
		direita bool
	}{
		{"ITEM", 0.05, false}, {"ORIGEM", 0.15, false}, {"DESTINO", 0.15, false},
		{"VALOR DA PRESTAÇÃO", 0.11, true}, {"A RECEBER", 0.10, true}, {"DOC.", 0.07, false},
		{"CHAVE DE ACESSO", 0.37, false},
	}
	y = cabecalhoDeSecao(pg, m, y, "ENTREGAS (PRESTAÇÕES DO CT-e SIMPLIFICADO)")
	cursor := m.margem
	for _, col := range colunas {
		largura := m.util * col.fracao
		pg.Retangulo(cursor, y, largura, 3.4, 0.1)
		pg.Texto(cursor+0.8, y+0.4, col.nome, rotulo)
		cursor += largura
	}
	y += 3.4
	celula := pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5}
	for i := 0; i < max(len(linhas), minLinhas); i++ {
		var valores [7]string
		if i < len(linhas) {
			l := linhas[i]
			valores = [7]string{l.item, l.origem, l.destino, l.valor, l.receber, l.tipo, l.chave}
		}
		cursor = m.margem
		for j, col := range colunas {
			largura := m.util * col.fracao
			pg.Retangulo(cursor, y, largura, alturaDocOrig, 0.1)
			texto := pdf.Encurtar(valores[j], celula.Fonte, celula.Tamanho, largura-1.6)
			if col.direita {
				pg.TextoDireita(cursor, largura-0.8, y+0.6, texto, celula)
			} else {
				pg.Texto(cursor+0.8, y+0.6, texto, celula)
			}
			cursor += largura
		}
		y += alturaDocOrig
	}
	return y + 1.5
}

// desenharIBSCBSTotal imprime o total de IBS e CBS do Simplificado (IBSCBSTot),
// quando há.
func desenharIBSCBSTotal(pg *pdf.Pagina, m medidas, t *cte.IBSCBSTot, vTotDFe *tipos.Decimal, y float64) float64 {
	if t == nil {
		return y
	}
	total := ""
	if vTotDFe != nil {
		total = moeda(*vTotDFe)
	}
	y = cabecalhoDeSecao(pg, m, y, "TRIBUTOS DA REFORMA TRIBUTÁRIA (IBS E CBS) - TOTAL DAS ENTREGAS")
	campos := [][2]string{
		{"BASE DE CÁLCULO", moeda(t.VBCIBSCBS)},
		{"IBS UF", moeda(t.GIBS.GIBSUF.VIBSUF)},
		{"IBS MUN.", moeda(t.GIBS.GIBSMun.VIBSMun)},
		{"IBS", moeda(t.GIBS.VIBS)},
		{"CBS", moeda(t.GCBS.VCBS)},
		{"TOTAL DO DOCUMENTO", total},
	}
	largura := m.util / float64(len(campos))
	for i, c := range campos {
		caixaDireita(pg, m.margem+float64(i)*largura, y, largura, 7, c[0], c[1], valor)
	}
	return y + 7 + 1.5
}

func fluxoDoSimp(compl *ctesimp.Compl) *cte.Fluxo {
	if compl == nil {
		return nil
	}
	return compl.Fluxo
}

func observacoesDoSimp(compl *ctesimp.Compl) (gerais, emissor, fisco string) {
	if compl == nil {
		return "", "", ""
	}
	var partes []string
	for _, campo := range []string{compl.XCaracAd, compl.XCaracSer, compl.XObs} {
		if campo != "" {
			partes = append(partes, campo)
		}
	}
	return strings.Join(partes, " | "), juntarObs(compl.ObsCont), juntarObs(compl.ObsFisco)
}

// desenharAdicionaisSimp imprime, depois do leiaute, a partilha do ICMS e a
// informação adicional ao fisco.
func desenharAdicionaisSimp(pg *pdf.Pagina, m medidas, c *ctesimp.CTeSimp, y float64) float64 {
	var partes []string
	if p := partilhaICMS(c.InfCte.Imp.ICMSUFFim); p != "" {
		partes = append(partes, p)
	}
	if ad := c.InfCte.Imp.InfAdFisco; ad != "" {
		partes = append(partes, "INFORMAÇÕES ADICIONAIS DE INTERESSE DO FISCO: "+ad)
	}
	if sub := c.InfCte.InfCteSub; sub != nil {
		partes = append(partes, "SUBSTITUI O CT-e "+chaveFormatada(sub.ChCte))
	}
	if len(partes) == 0 {
		return y
	}
	return quadroDeTexto(pg, m.margem, y, m.util, 8, "INFORMAÇÕES ADICIONAIS", strings.Join(partes, " | "), 4) + 1.5
}
