package danfe

import (
	"fmt"
	"strings"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/internal/pdf"
	"github.com/sisgt-sistemas/gonfe/tipos"
	"github.com/sisgt-sistemas/gonfe/validacao"
)

// alturaDocOrig é o espaço vertical de cada linha da tabela de documentos
// originários.
const alturaDocOrig = 3.6

// GerarDACTE produz o documento auxiliar a partir do XML de distribuição do
// CT-e, o cteProc.
func GerarDACTE(procCTe []byte, opc Opcoes) ([]byte, error) {
	c, prot, err := cte.LerCTeProc(procCTe)
	if err != nil {
		return nil, fmt.Errorf("dacte: %w", err)
	}
	return DACTE(c, prot, opc)
}

// DACTE gera o documento auxiliar do CT-e modelo 57, em A4, pelo MOC CT-e 4.00,
// Anexo II (Manual do DACTE).
//
// Os quadros fixos ficam na primeira folha. Os documentos originários que não
// couberem nela continuam nas seguintes, que repetem o cabeçalho e só trazem a
// continuação (§2.12). O QR Code sai do qrCodCTe do documento. Os quadros que o
// manual não prevê (IBS e CBS, ordens de coleta, partilha do ICMS) vêm depois
// do leiaute padrão.
//
// A folha é sempre A4 retrato: em paisagem os quadros fixos não cabem, e
// [Opcoes.Orientacao] é ignorada. Se os quadros fixos não couberem nem em retrato
// (componentes ou quantidades demais), o erro embrulha [ErrNaoCabe].
func DACTE(c *cte.CTe, prot *cte.ProtCTe, opc Opcoes) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("dacte: conhecimento ausente")
	}

	opc = emRetrato(opc)
	paginas, linhas, err := paginarDocumentos(c, prot, opc)
	if err != nil {
		return nil, fmt.Errorf("dacte: %w", err)
	}
	documento := pdf.Novo()
	for i, docs := range paginas {
		pg := novaFolha(documento, opc)
		m := medidasDe(pg)
		if i == 0 {
			desenharPrimeiraFolhaCTe(pg, m, c, prot, opc, docs, linhas[0], len(paginas))
			continue
		}
		y := desenharCabecalhoCTe(pg, m, c, prot, opc, i+1, len(paginas), m.margem)
		desenharDocumentosCTe(pg, m, docs, linhas[1], y)
	}
	return documento.Bytes()
}

// desenharPrimeiraFolhaCTe desenha a primeira folha com os documentos dados e
// devolve onde ela terminou: é também a medida da paginação.
func desenharPrimeiraFolhaCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, prot *cte.ProtCTe, opc Opcoes, docs []docOrig, linhas, folhas int) float64 {
	y := m.margem
	if !opc.SemCanhoto {
		y = desenharCanhotoCTe(pg, m, c, y)
	}
	y = desenharCabecalhoCTe(pg, m, c, prot, opc, 1, folhas, y)
	y = desenharPrestacaoCTe(pg, m, c, y)
	y = desenharPartesCTe(pg, m, c, y)
	y = desenharCargaCTe(pg, m, c, y)
	y = desenharValoresCTe(pg, m, c.InfCte.VPrest, y)
	y = desenharImpostoCTe(pg, m, &c.InfCte.Imp, y)
	y = desenharIBSCBS(pg, m, c.InfCte.Imp.IBSCBS, c.InfCte.Imp.VTotDFe, y)
	y = desenharDocumentosCTe(pg, m, docs, linhas, y)
	y = desenharFluxo(pg, m, fluxoDe(c.InfCte.Compl), y)
	y = desenharObservacoesCTe(pg, m, c, y)
	return desenharAdicionaisCTe(pg, m, c, y)
}

// docOrig é uma linha da tabela de documentos originários, já achatada a
// partir das três formas que o leiaute aceita.
type docOrig struct {
	tipo, emitente, numero string
}

// paginarDocumentos divide os documentos originários entre as folhas, medindo o
// espaço que sobra: a primeira folha é desenhada à parte com a tabela vazia, e o
// que fica até o pé recebe linhas de dois documentos. As demais só têm o
// cabeçalho e a continuação. Devolve também quantas linhas a tabela tem na
// primeira folha e nas demais: as que sobram saem em branco, até o pé, como nos
// modelos do manual.
func paginarDocumentos(c *cte.CTe, prot *cte.ProtCTe, opc Opcoes) ([][]docOrig, [2]int, error) {
	docs := documentosOriginarios(c)

	rascunho := pdf.Novo()
	pg := novaFolha(rascunho, opc)
	m := medidasDe(pg)
	fim := desenharPrimeiraFolhaCTe(pg, m, c, prot, opc, nil, 1, 1)
	if err := cabeNaFolha(fim, m); err != nil {
		return nil, [2]int{}, err
	}
	// A tabela vazia já ocupa uma linha; o que sobra vira linhas a mais.
	naPrimeira := 2 * (1 + max(int((m.altura-m.margem-fim)/alturaDocOrig), 0))

	pg2 := novaFolha(rascunho, opc)
	topo := desenharCabecalhoCTe(pg2, m, c, prot, opc, 2, 2, m.margem) + 3.4*2
	nasDemais := 2 * max(int((m.altura-m.margem-topo)/alturaDocOrig), 1)

	linhas := [2]int{naPrimeira / 2, nasDemais / 2}
	if len(docs) <= naPrimeira {
		return [][]docOrig{docs}, linhas, nil
	}
	paginas := [][]docOrig{docs[:naPrimeira]}
	for corte := naPrimeira; corte < len(docs); corte += nasDemais {
		paginas = append(paginas, docs[corte:min(corte+nasDemais, len(docs))])
	}
	return paginas, linhas, nil
}

// documentosOriginarios achata as três listas de documentos transportados em
// uma só, nas colunas do manual: tipo, CNPJ/CPF do emitente, série e número.
func documentosOriginarios(c *cte.CTe) []docOrig {
	norm := c.InfCte.InfCTeNorm
	if norm == nil || norm.InfDoc == nil {
		return nil
	}

	var docs []docOrig
	for _, d := range norm.InfDoc.InfNFe {
		doc := docOrig{tipo: "NF-e"}
		// A chave carrega o emitente, a série e o número; aproveitá-los evita
		// exigir do chamador dados que ele já informou uma vez.
		if p, err := chave.Parse(d.Chave); err == nil {
			doc.emitente = validacao.FormatarCNPJ(p.CNPJ)
			doc.numero = fmt.Sprintf("%03d / %09d", p.Serie, p.Numero)
		} else {
			doc.numero = d.Chave
		}
		docs = append(docs, doc)
	}
	for _, d := range norm.InfDoc.InfNF {
		docs = append(docs, docOrig{tipo: "NF mod. " + d.Mod, numero: d.Serie + " / " + d.NDoc})
	}
	for _, d := range norm.InfDoc.InfOutros {
		tipo := "OUTROS"
		if d.DescOutros != "" {
			tipo = d.DescOutros
		}
		docs = append(docs, docOrig{tipo: tipo, numero: d.NDoc})
	}
	return docs
}

// desenharCanhotoCTe imprime o recibo de entrega no topo da primeira folha, com
// os campos do modelo do manual.
func desenharCanhotoCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	const altura = 15
	larguraRecibo := m.util * 0.84

	pg.Retangulo(m.margem, y, larguraRecibo, altura, 0.1)
	pg.Texto(m.margem+1, y+1,
		"DECLARO QUE RECEBI OS VOLUMES DESTE CONHECIMENTO EM PERFEITO ESTADO "+
			"PELO QUE DOU POR CUMPRIDO O PRESENTE CONTRATO DE TRANSPORTE",
		miudo)

	meio := y + 5
	colunas := []struct {
		nome    string
		largura float64
	}{
		{"NOME", 0.24}, {"RG", 0.12}, {"ASSINATURA / CARIMBO", 0.24},
		{"INÍCIO DA PRESTAÇÃO - DATA / HORA", 0.20}, {"TÉRMINO DA PRESTAÇÃO - DATA / HORA", 0.20},
	}
	cursor := m.margem
	for _, col := range colunas {
		largura := larguraRecibo * col.largura
		pg.Retangulo(cursor, meio, largura, altura-(meio-y), 0.1)
		pg.Texto(cursor+0.8, meio+0.5, col.nome, rotulo)
		cursor += largura
	}

	x := m.margem + larguraRecibo
	largura := m.util - larguraRecibo
	pg.Retangulo(x, y, largura, altura, 0.1)
	pg.TextoCentralizado(x, largura, y+1.5, "CT-e", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 8})
	pg.TextoCentralizado(x, largura, y+6, fmt.Sprintf("Nº %09d", c.InfCte.Ide.NCT), valorForte)
	pg.TextoCentralizado(x, largura, y+10.5, fmt.Sprintf("SÉRIE %03d", c.InfCte.Ide.Serie), valorForte)

	// Linha tracejada de corte.
	corte := y + altura + 1.5
	for cursor := m.margem; cursor < m.margem+m.util; cursor += 3 {
		pg.Linha(cursor, corte, cursor+1.5, corte, 0.1)
	}
	return corte + 2
}

// linhasDoEmitente são as linhas do emitente no cabeçalho: endereço completo,
// telefone e documentos (§2.6).
func linhasDoEmitente(nomeFantasia string, end cte.EnderecoEmitente, cnpj, cpf, ie string) []string {
	linhas := []string{enderecoEmUmaLinha(end.XLgr, end.Nro, end.XCpl)}
	linhas = append(linhas,
		end.XBairro+" - CEP: "+validacao.FormatarCEP(end.CEP),
		end.XMun+" - "+end.UF,
	)
	if end.Fone != "" {
		linhas = append(linhas, "FONE: "+end.Fone)
	}
	linhas = append(linhas, "CNPJ/CPF: "+formatarDocumento(cnpj, cpf)+"   IE: "+ie)
	if nomeFantasia != "" {
		linhas = append([]string{nomeFantasia}, linhas...)
	}
	return linhas
}

// situacaoCTe é o aviso da área variável (§2.20): homologação, cancelado, sem
// protocolo ou contingência.
func situacaoCTe(tpAmb cte.Ambiente, tpEmis cte.TipoEmissao, autorizado bool, opc Opcoes, nomeDoc string) string {
	homologacao := opc.Homologacao || tpAmb == cte.Homologacao
	switch {
	case opc.Cancelada && homologacao:
		return nomeDoc + " CANCELADO – EMITIDO EM AMBIENTE DE HOMOLOGAÇÃO"
	case opc.Cancelada:
		return nomeDoc + " CANCELADO"
	case homologacao:
		return "EMITIDO EM AMBIENTE DE HOMOLOGAÇÃO – SEM VALOR FISCAL"
	case tpEmis.Contingencia():
		// Em contingência o documento é impresso antes da autorização, que é para isso que ela existe.
		return "EMITIDO EM CONTINGÊNCIA"
	case !autorizado:
		return "DOCUMENTO SEM PROTOCOLO DE AUTORIZAÇÃO – SEM VALOR FISCAL"
	}
	return ""
}

// desenharCabecalhoCTe imprime o cabeçalho, que se repete em toda folha, e as
// linhas do tipo do conhecimento e da natureza da operação.
func desenharCabecalhoCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, prot *cte.ProtCTe, opc Opcoes, folha, folhas int, y float64) float64 {
	ide := &c.InfCte.Ide
	emit := &c.InfCte.Emit
	suframa := ""
	if d := c.InfCte.Dest; d != nil {
		suframa = d.ISUF
	}
	qrCode := ""
	if c.InfCTeSupl != nil {
		qrCode = c.InfCTeSupl.QrCodCTe
	}
	y = desenharIdentificacao(pg, m, identificacao{
		sigla: "DACTE", nome1: "Documento Auxiliar do Conhecimento", nome2: "de Transporte Eletrônico",
		modal:    "MODAL " + strings.ToUpper(ide.Modal.Descricao()),
		emitNome: emit.XNome, emitLinhas: linhasDoEmitente(emit.XFant, emit.EnderEmit, emit.CNPJ, emit.CPF, emit.IE),
		modelo: string(ide.Mod), serie: fmt.Sprintf("%03d", ide.Serie), numero: fmt.Sprintf("%09d", ide.NCT),
		emissao: dataHora(ide.DhEmi), suframa: suframa, folha: folha, folhas: folhas,
		chave: c.Chave(), protocolo: protocoloCTeDescrito(prot), consulta: URLConsultaCTe, qrCode: qrCode,
		situacao: situacaoCTe(ide.TpAmb, ide.TpEmis, prot != nil && prot.Autorizado(), opc, "CT-e"),
	}, y)

	globalizado := "NÃO"
	if ide.IndGlobalizado == "1" {
		globalizado = "SIM"
	}
	infGlobalizado := ""
	if n := c.InfCte.InfCTeNorm; n != nil && n.InfGlobalizado != nil {
		infGlobalizado = n.InfGlobalizado.XObs
	}
	caixa(pg, m.margem, y, m.util*0.22, 7, "TIPO DO CT-e", tipoDoCTe(ide.TpCTe), valor)
	caixa(pg, m.margem+m.util*0.22, y, m.util*0.26, 7, "TIPO DO SERVIÇO", tipoDoServico(ide.TpServ), valor)
	caixa(pg, m.margem+m.util*0.48, y, m.util*0.14, 7, "CT-e GLOBALIZADO", globalizado, valor)
	caixa(pg, m.margem+m.util*0.62, y, m.util*0.38, 7, "INF. DO CT-e GLOBALIZADO", infGlobalizado, valor)
	y += 7
	caixa(pg, m.margem, y, m.util, 7, "CFOP - NATUREZA DA OPERAÇÃO", ide.CFOP+" - "+ide.NatOp, valor)
	return y + 7
}

func protocoloCTeDescrito(prot *cte.ProtCTe) string {
	if prot == nil || prot.InfProt.NProt == "" {
		return "DOCUMENTO SEM PROTOCOLO DE AUTORIZAÇÃO"
	}
	return prot.InfProt.NProt + " - " + dataHora(prot.InfProt.DhRecbto)
}

func tipoDoCTe(t cte.TipoCTe) string {
	switch t {
	case cte.CTeNormal:
		return "0 - NORMAL"
	case cte.CTeComplemento:
		return "1 - COMPLEMENTO DE VALORES"
	case cte.CTeAnulacao:
		return "2 - ANULAÇÃO DE VALORES"
	case cte.CTeSubstituto:
		return "3 - SUBSTITUTO"
	default:
		return string(t)
	}
}

func tipoDoServico(t cte.TipoServico) string {
	switch t {
	case cte.ServicoNormal:
		return "0 - NORMAL"
	case cte.ServicoSubcontratacao:
		return "1 - SUBCONTRATAÇÃO"
	case cte.ServicoRedespacho:
		return "2 - REDESPACHO"
	case cte.ServicoRedespachoIntermediario:
		return "3 - REDESPACHO INTERMEDIÁRIO"
	case cte.ServicoVinculadoMultimodal:
		return "4 - VINCULADO A MULTIMODAL"
	default:
		return string(t)
	}
}

// desenharPrestacaoCTe imprime os extremos da prestação.
func desenharPrestacaoCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	ide := &c.InfCte.Ide
	caixa(pg, m.margem, y, m.util*0.5, 7, "INÍCIO DA PRESTAÇÃO", ide.XMunIni+" - "+ide.UFIni, valor)
	caixa(pg, m.margem+m.util*0.5, y, m.util*0.5, 7, "TÉRMINO DA PRESTAÇÃO", ide.XMunFim+" - "+ide.UFFim, valor)
	return y + 7 + 1.5
}

// parte reúne o que o DACTE imprime de cada participante do transporte. Os
// quatro papéis têm tipos distintos no modelo, porque o esquema nomeia o
// endereço de cada um de um jeito; aqui eles voltam a ser um só.
type parte struct {
	nome, documento, ie, fone string
	endereco                  *cte.Endereco
}

func parteDoRemetente(r *cte.Rem) *parte {
	if r == nil {
		return nil
	}
	return &parte{nome: r.XNome, documento: formatarDocumento(r.CNPJ, r.CPF), ie: r.IE, fone: r.Fone, endereco: r.EnderReme}
}

func parteDoExpedidor(e *cte.Exped) *parte {
	if e == nil {
		return nil
	}
	return &parte{nome: e.XNome, documento: formatarDocumento(e.CNPJ, e.CPF), ie: e.IE, fone: e.Fone, endereco: e.EnderExped}
}

func parteDoRecebedor(r *cte.Receb) *parte {
	if r == nil {
		return nil
	}
	return &parte{nome: r.XNome, documento: formatarDocumento(r.CNPJ, r.CPF), ie: r.IE, fone: r.Fone, endereco: r.EnderReceb}
}

func parteDoDestinatario(d *cte.Dest) *parte {
	if d == nil {
		return nil
	}
	return &parte{nome: d.XNome, documento: formatarDocumento(d.CNPJ, d.CPF), ie: d.IE, fone: d.Fone, endereco: d.EnderDest}
}

// tomador resolve quem paga o frete, seja pelo apontamento do toma3 para uma
// das partes, seja pelo grupo toma4.
func tomador(c *cte.CTe) *parte {
	ide := &c.InfCte.Ide
	if t := ide.Toma4; t != nil {
		return &parte{nome: t.XNome, documento: formatarDocumento(t.CNPJ, t.CPF), ie: t.IE, fone: t.Fone, endereco: t.EnderToma}
	}
	if ide.Toma3 == nil {
		return nil
	}
	switch ide.Toma3.Toma {
	case cte.TomadorRemetente:
		return parteDoRemetente(c.InfCte.Rem)
	case cte.TomadorExpedidor:
		return parteDoExpedidor(c.InfCte.Exped)
	case cte.TomadorRecebedor:
		return parteDoRecebedor(c.InfCte.Receb)
	case cte.TomadorDestinatario:
		return parteDoDestinatario(c.InfCte.Dest)
	default:
		return nil
	}
}

// desenharPartesCTe imprime remetente e destinatário, expedidor e recebedor
// (quando o documento os traz, §2.7) e o tomador.
func desenharPartesCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	metade := m.util / 2
	desenharParte(pg, m.margem, y, metade, "REMETENTE", parteDoRemetente(c.InfCte.Rem))
	desenharParte(pg, m.margem+metade, y, metade, "DESTINATÁRIO", parteDoDestinatario(c.InfCte.Dest))
	y += alturaParte

	exped, receb := parteDoExpedidor(c.InfCte.Exped), parteDoRecebedor(c.InfCte.Receb)
	if exped != nil || receb != nil {
		desenharParte(pg, m.margem, y, metade, "EXPEDIDOR", exped)
		desenharParte(pg, m.margem+metade, y, metade, "RECEBEDOR", receb)
		y += alturaParte
	}
	desenharParte(pg, m.margem, y, m.util, "TOMADOR DO SERVIÇO", tomador(c))
	return y + alturaParte + 1.5
}

// alturaParte é a altura do quadro de um participante: três linhas de campos.
const alturaParte = 21

// desenharParte desenha um participante em três linhas de campos: nome e
// documento; endereço, bairro e país; município, UF, CEP, IE e telefone.
func desenharParte(pg *pdf.Pagina, x, y, largura float64, nome string, p *parte) {
	pg.Retangulo(x, y, largura, alturaParte, 0.1)
	pg.RetanguloPreenchido(x, y, largura, 3.4, 0.88)
	pg.Texto(x+0.8, y+0.4, nome, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 5})
	if p == nil {
		pg.Texto(x+0.8, y+6, "NÃO INFORMADO", miudo)
		return
	}
	var end cte.Endereco
	if p.endereco != nil {
		end = *p.endereco
	}
	interna := y + 3.4
	caixa(pg, x, interna, largura*0.62, 6, "NOME / RAZÃO SOCIAL", p.nome, valor)
	caixa(pg, x+largura*0.62, interna, largura*0.38, 6, "CNPJ / CPF", p.documento, valor)
	interna += 6
	caixa(pg, x, interna, largura*0.50, 6, "ENDEREÇO", enderecoEmUmaLinha(end.XLgr, end.Nro, end.XCpl), valor)
	caixa(pg, x+largura*0.50, interna, largura*0.30, 6, "BAIRRO", end.XBairro, valor)
	caixa(pg, x+largura*0.80, interna, largura*0.20, 6, "PAÍS", end.XPais, valor)
	interna += 6
	caixa(pg, x, interna, largura*0.28, 6, "MUNICÍPIO", end.XMun, valor)
	caixa(pg, x+largura*0.28, interna, largura*0.07, 6, "UF", end.UF, valor)
	caixa(pg, x+largura*0.35, interna, largura*0.17, 6, "CEP", validacao.FormatarCEP(end.CEP), valor)
	caixa(pg, x+largura*0.52, interna, largura*0.26, 6, "INSCRIÇÃO ESTADUAL", p.ie, valor)
	caixa(pg, x+largura*0.78, interna, largura*0.22, 6, "FONE", p.fone, valor)
}

// desenharCargaCTe imprime o produto predominante e as quantidades.
func desenharCargaCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	norm := c.InfCte.InfCTeNorm
	if norm == nil {
		return y
	}
	return desenharCarga(pg, m, &norm.InfCarga, y)
}

// desenharCarga imprime o produto predominante, o valor e as quantidades da carga, em até cinco colunas por linha.
func desenharCarga(pg *pdf.Pagina, m medidas, carga *cte.InfCarga, y float64) float64 {
	valorCarga := ""
	if carga.VCarga != nil {
		valorCarga = moeda(*carga.VCarga)
	}
	caixa(pg, m.margem, y, m.util*0.40, 7, "PRODUTO PREDOMINANTE", carga.ProPred, valor)
	caixa(pg, m.margem+m.util*0.40, y, m.util*0.40, 7, "OUTRAS CARACTERÍSTICAS DA CARGA", carga.XOutCat, valor)
	caixaDireita(pg, m.margem+m.util*0.80, y, m.util*0.20, 7, "VALOR TOTAL DA CARGA", valorCarga, valor)
	y += 7

	const porLinha = 5
	largura := m.util / porLinha
	for i, q := range carga.InfQ {
		coluna := i % porLinha
		if coluna == 0 && i > 0 {
			y += 7
		}
		caixaDireita(pg, m.margem+float64(coluna)*largura, y, largura, 7,
			strings.ToUpper(q.TpMed)+" ("+unidadeCTe(q.CUnid)+")", quantidade(q.QCarga), valor)
	}
	if n := len(carga.InfQ); n > 0 {
		for coluna := n % porLinha; coluna > 0 && coluna < porLinha; coluna++ {
			pg.Retangulo(m.margem+float64(coluna)*largura, y, largura, 7, 0.1)
		}
		y += 7
	}
	return y + 1.5
}

func unidadeCTe(u cte.UnidadeDeMedida) string {
	switch u {
	case cte.UnidadeM3:
		return "M3"
	case cte.UnidadeKG:
		return "KG"
	case cte.UnidadeTON:
		return "TON"
	case cte.UnidadeUnidade:
		return "UNID"
	case cte.UnidadeLitros:
		return "LITROS"
	case cte.UnidadeMMBTU:
		return "MMBTU"
	default:
		return string(u)
	}
}

// desenharValoresCTe imprime os componentes do frete, em quatro colunas como
// no manual, e os totais.
func desenharValoresCTe(pg *pdf.Pagina, m medidas, v cte.VPrest, y float64) float64 {
	y = cabecalhoDeSecao(pg, m, y, "COMPONENTES DO VALOR DA PRESTAÇÃO DO SERVIÇO")
	const porLinha = 4
	largura := m.util / porLinha
	for i, comp := range v.Comp {
		coluna := i % porLinha
		if coluna == 0 && i > 0 {
			y += 7
		}
		caixaDireita(pg, m.margem+float64(coluna)*largura, y, largura, 7, strings.ToUpper(comp.XNome), moeda(comp.VComp), valor)
	}
	if n := len(v.Comp); n > 0 {
		for coluna := n % porLinha; coluna > 0 && coluna < porLinha; coluna++ {
			pg.Retangulo(m.margem+float64(coluna)*largura, y, largura, 7, 0.1)
		}
		y += 7
	}
	caixaDireita(pg, m.margem, y, m.util*0.5, 7, "VALOR TOTAL DA PRESTAÇÃO DO SERVIÇO", moeda(v.VTPrest), valorForte)
	caixaDireita(pg, m.margem+m.util*0.5, y, m.util*0.5, 7, "VALOR A RECEBER", moeda(v.VRec), valorForte)
	return y + 7 + 1.5
}

// impostoCTe reúne o que os sete grupos de ICMS têm de comum para a impressão.
type impostoCTe struct {
	cst, reducao, base, aliquota, valor string
}

// resumirICMS achata o grupo de ICMS preenchido em um conjunto único de
// campos. O ICMS60 (substituição) entra nas mesmas colunas, pelo valor retido.
func resumirICMS(i *cte.ICMS) impostoCTe {
	switch {
	case i.ICMS00 != nil:
		return impostoCTe{cst: i.ICMS00.CST, base: moeda(i.ICMS00.VBC),
			aliquota: percentual(i.ICMS00.PICMS), valor: moeda(i.ICMS00.VICMS)}
	case i.ICMS20 != nil:
		return impostoCTe{cst: i.ICMS20.CST, reducao: percentual(i.ICMS20.PRedBC),
			base: moeda(i.ICMS20.VBC), aliquota: percentual(i.ICMS20.PICMS), valor: moeda(i.ICMS20.VICMS)}
	case i.ICMS45 != nil:
		return impostoCTe{cst: i.ICMS45.CST}
	case i.ICMS60 != nil:
		return impostoCTe{cst: i.ICMS60.CST, base: moeda(i.ICMS60.VBCSTRet),
			aliquota: percentual(i.ICMS60.PICMSSTRet), valor: moeda(i.ICMS60.VICMSSTRet)}
	case i.ICMS90 != nil:
		r := impostoCTe{cst: i.ICMS90.CST, base: moeda(i.ICMS90.VBC),
			aliquota: percentual(i.ICMS90.PICMS), valor: moeda(i.ICMS90.VICMS)}
		if i.ICMS90.PRedBC != nil {
			r.reducao = percentual(*i.ICMS90.PRedBC)
		}
		return r
	case i.ICMSOutraUF != nil:
		r := impostoCTe{cst: i.ICMSOutraUF.CSTOutraUF, base: moeda(i.ICMSOutraUF.VBCOutraUF),
			aliquota: percentual(i.ICMSOutraUF.PICMSOutraUF), valor: moeda(i.ICMSOutraUF.VICMSOutraUF)}
		if i.ICMSOutraUF.PRedBCOutraUF != nil {
			r.reducao = percentual(*i.ICMSOutraUF.PRedBCOutraUF)
		}
		return r
	case i.ICMSSN != nil:
		return impostoCTe{cst: i.ICMSSN.CST}
	default:
		return impostoCTe{}
	}
}

// percentual apresenta uma alíquota com duas casas.
func percentual(d tipos.Decimal) string { return separarMilhar(d.ComCasas(2).String()) + "%" }

// desenharImpostoCTe imprime o quadro do ICMS com os campos do manual.
func desenharImpostoCTe(pg *pdf.Pagina, m medidas, imp *cte.Imp, y float64) float64 {
	return desenharICMS(pg, m, &imp.ICMS, y)
}

func desenharICMS(pg *pdf.Pagina, m medidas, icms *cte.ICMS, y float64) float64 {
	r := resumirICMS(icms)
	y = cabecalhoDeSecao(pg, m, y, "INFORMAÇÕES RELATIVAS AO IMPOSTO")
	campos := []struct {
		nome, conteudo string
		fracao         float64
		direita        bool
	}{
		{"CLASSIFICAÇÃO TRIBUTÁRIA DO SERVIÇO", situacaoTributaria(r.cst), 0.36, false},
		{"BASE DE CÁLCULO", r.base, 0.17, true},
		{"ALÍQ. ICMS", r.aliquota, 0.12, true},
		{"VALOR ICMS", r.valor, 0.17, true},
		{"% RED. BC. CALC.", r.reducao, 0.18, true},
	}
	cursor := m.margem
	for _, campo := range campos {
		largura := m.util * campo.fracao
		if campo.direita {
			caixaDireita(pg, cursor, y, largura, 7, campo.nome, campo.conteudo, valor)
		} else {
			caixa(pg, cursor, y, largura, 7, campo.nome, campo.conteudo, valor)
		}
		cursor += largura
	}
	return y + 7 + 1.5
}

// desenharIBSCBS imprime o quadro do IBS e da CBS, que o manual de 2022 ainda não
// prevê: vem depois do imposto, só quando o documento traz o grupo.
func desenharIBSCBS(pg *pdf.Pagina, m medidas, g *cte.IBSCBS, vTotDFe *tipos.Decimal, y float64) float64 {
	if g == nil || g.GIBSCBS == nil {
		return y
	}
	t := g.GIBSCBS
	total := ""
	if vTotDFe != nil {
		total = moeda(*vTotDFe)
	}
	y = cabecalhoDeSecao(pg, m, y, "TRIBUTOS DA REFORMA TRIBUTÁRIA (IBS E CBS)")
	campos := [][2]string{
		{"CST / CLASSIF.", g.CST + " / " + g.CClassTrib},
		{"BASE DE CÁLCULO", moeda(t.VBC)},
		{"ALÍQ. IBS UF", percentual(t.GIBSUF.PIBSUF)},
		{"IBS UF", moeda(t.GIBSUF.VIBSUF)},
		{"ALÍQ. IBS MUN.", percentual(t.GIBSMun.PIBSMun)},
		{"IBS MUN.", moeda(t.GIBSMun.VIBSMun)},
		{"ALÍQ. CBS", percentual(t.GCBS.PCBS)},
		{"CBS", moeda(t.GCBS.VCBS)},
		{"TOTAL DO DOCUMENTO", total},
	}
	largura := m.util / float64(len(campos))
	for i, c := range campos {
		caixaDireita(pg, m.margem+float64(i)*largura, y, largura, 7, c[0], c[1], valor)
	}
	return y + 7 + 1.5
}

func situacaoTributaria(cst string) string {
	nomes := map[string]string{
		"00": "00 - TRIBUTAÇÃO NORMAL",
		"20": "20 - BC REDUZIDA",
		"40": "40 - ISENTA",
		"41": "41 - NÃO TRIBUTADA",
		"45": "45 - ISENTA / NÃO TRIBUTADA",
		"51": "51 - DIFERIMENTO",
		"60": "60 - ICMS COBRADO ANTERIORMENTE POR ST",
		"90": "90 - OUTROS",
	}
	if nome, ok := nomes[cst]; ok {
		return nome
	}
	return cst
}

// desenharDocumentosCTe imprime a tabela de documentos originários em duas
// colunas lado a lado, como no modelo do manual (p.27 e 28), com pelo menos
// minLinhas linhas: as que sobram saem em branco.
func desenharDocumentosCTe(pg *pdf.Pagina, m medidas, docs []docOrig, minLinhas int, y float64) float64 {
	colunas := []struct {
		nome   string
		fracao float64
	}{
		{"TP DOC.", 0.22}, {"CNPJ/CPF EMITENTE", 0.40}, {"SÉRIE / Nº DOCUMENTO", 0.38},
	}
	metade := m.util / 2
	y = cabecalhoDeSecao(pg, m, y, "DOCUMENTOS ORIGINÁRIOS")
	for lado := 0; lado < 2; lado++ {
		cursor := m.margem + float64(lado)*metade
		for _, col := range colunas {
			largura := metade * col.fracao
			pg.Retangulo(cursor, y, largura, 3.4, 0.1)
			pg.Texto(cursor+0.8, y+0.4, col.nome, rotulo)
			cursor += largura
		}
	}
	y += 3.4

	if len(docs) == 0 {
		pg.Retangulo(m.margem, y, m.util, alturaDocOrig, 0.1)
		pg.Texto(m.margem+0.8, y+0.6, "SEM DOCUMENTOS ORIGINÁRIOS RELACIONADOS", miudo)
		return y + alturaDocOrig + 1.5
	}
	celula := pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5}
	for i := 0; i < max(len(docs), 2*minLinhas); i += 2 {
		for lado := 0; lado < 2; lado++ {
			cursor := m.margem + float64(lado)*metade
			var valores [3]string
			if i+lado < len(docs) {
				d := docs[i+lado]
				valores = [3]string{d.tipo, d.emitente, d.numero}
			}
			for j, col := range colunas {
				largura := metade * col.fracao
				pg.Retangulo(cursor, y, largura, alturaDocOrig, 0.1)
				pg.Texto(cursor+0.8, y+0.6, pdf.Encurtar(valores[j], celula.Fonte, celula.Tamanho, largura-1.6), celula)
				cursor += largura
			}
		}
		y += alturaDocOrig
	}
	return y + 1.5
}

func fluxoDe(compl *cte.Compl) *cte.Fluxo {
	if compl == nil {
		return nil
	}
	return compl.Fluxo
}

// desenharFluxo imprime a previsão do fluxo da carga (compl.fluxo), quando há.
func desenharFluxo(pg *pdf.Pagina, m medidas, f *cte.Fluxo, y float64) float64 {
	if f == nil {
		return y
	}
	passagens := make([]string, 0, len(f.Pass))
	for _, p := range f.Pass {
		passagens = append(passagens, p.XPass)
	}
	y = cabecalhoDeSecao(pg, m, y, "PREVISÃO DO FLUXO DA CARGA")
	caixa(pg, m.margem, y, m.util*0.25, 7, "SIGLA OU CÓDIGO INTERNO DA FILIAL/PORTO/ESTAÇÃO/AEROPORTO DE ORIGEM", f.XOrig, valor)
	caixa(pg, m.margem+m.util*0.25, y, m.util*0.40, 7, "PASSAGEM", strings.Join(passagens, ", "), valor)
	caixa(pg, m.margem+m.util*0.65, y, m.util*0.25, 7, "DESTINO", f.XDest, valor)
	caixa(pg, m.margem+m.util*0.90, y, m.util*0.10, 7, "ROTA", f.XRota, valor)
	return y + 7 + 1.5
}

// observacoesDoCTe junta as observações gerais do documento (§ quadro
// Observações): características adicionais e xObs. O ObsCont vai para "Uso
// exclusivo do emissor" e o ObsFisco para "Reservado ao fisco".
func observacoesDoCTe(compl *cte.Compl) (gerais, emissor, fisco string) {
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

func juntarObs(obs []cte.ObsCampo) string {
	partes := make([]string, 0, len(obs))
	for _, o := range obs {
		partes = append(partes, o.XCampo+": "+o.XTexto)
	}
	return strings.Join(partes, " | ")
}

// desenharObservacoesCTe imprime observações, o modal rodoviário, o uso
// exclusivo do emissor e o reservado ao fisco.
func desenharObservacoesCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	gerais, emissor, fisco := observacoesDoCTe(c.InfCte.Compl)
	y = quadroDeTexto(pg, m.margem, y, m.util, 10, "OBSERVAÇÕES", gerais, 4)
	if norm := c.InfCte.InfCTeNorm; norm != nil && norm.InfModal.Rodo != nil {
		y = cabecalhoDeSecao(pg, m, y, "INFORMAÇÕES ESPECÍFICAS DO MODAL RODOVIÁRIO")
		caixa(pg, m.margem, y, m.util, 7, "RNTRC DA EMPRESA", norm.InfModal.Rodo.RNTRC, valor)
		y += 7
	}
	fimEmissor := quadroDeTexto(pg, m.margem, y, m.util*0.5, 10, "USO EXCLUSIVO DO EMISSOR DO CT-e", emissor, 3)
	fimFisco := quadroDeTexto(pg, m.margem+m.util*0.5, y, m.util*0.5, 10, "RESERVADO AO FISCO", fisco, 3)
	return max(fimEmissor, fimFisco) + 1.5
}

// desenharAdicionaisCTe imprime, depois do leiaute padrão, o que o documento
// traz e o manual não prevê: ordens de coleta, partilha do ICMS e informação
// adicional ao fisco.
func desenharAdicionaisCTe(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	var partes []string
	if norm := c.InfCte.InfCTeNorm; norm != nil && norm.InfModal.Rodo != nil {
		if occ := ordensDeColeta(norm.InfModal.Rodo.Occ); occ != "" {
			partes = append(partes, "ORDENS DE COLETA: "+occ)
		}
	}
	if p := partilhaICMS(&c.InfCte.Imp); p != "" {
		partes = append(partes, p)
	}
	if ad := c.InfCte.Imp.InfAdFisco; ad != "" {
		partes = append(partes, "INFORMAÇÕES ADICIONAIS DE INTERESSE DO FISCO: "+ad)
	}
	if len(partes) == 0 {
		return y
	}
	return quadroDeTexto(pg, m.margem, y, m.util, 8, "INFORMAÇÕES ADICIONAIS", strings.Join(partes, " | "), 4)
}

// partilhaICMS descreve o ICMS de partilha com a UF de término (ICMSUFFim).
func partilhaICMS(imp *cte.Imp) string {
	p := imp.ICMSUFFim
	if p == nil {
		return ""
	}
	return fmt.Sprintf("PARTILHA DO ICMS: BC %s, ICMS UF FIM %s, ICMS UF INÍCIO %s, FCP %s",
		moeda(p.VBCUFFim), moeda(p.VICMSUFFim), moeda(p.VICMSUFIni), moeda(p.VFCPUFFim))
}

func ordensDeColeta(occ []cte.Occ) string {
	partes := make([]string, 0, len(occ))
	for _, o := range occ {
		if o.Serie != "" {
			partes = append(partes, fmt.Sprintf("%s/%d", o.Serie, o.NOcc))
			continue
		}
		partes = append(partes, fmt.Sprint(o.NOcc))
	}
	return strings.Join(partes, ", ")
}
