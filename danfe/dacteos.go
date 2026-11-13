package danfe

import (
	"fmt"
	"strings"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/cteos"
	"github.com/sisgt-sistemas/gonfe/internal/pdf"
	"github.com/sisgt-sistemas/gonfe/tipos"
)

// GerarDACTEOS produz o documento auxiliar a partir do XML de distribuição do
// CT-e OS, o cteOSProc.
func GerarDACTEOS(procCTeOS []byte, opc Opcoes) ([]byte, error) {
	c, prot, err := cteos.LerCTeOSProc(procCTeOS)
	if err != nil {
		return nil, fmt.Errorf("dacteos: %w", err)
	}
	return DACTEOS(c, prot, opc)
}

// DACTEOS gera o documento auxiliar do CT-e OS, modelo 67, em A4, pelo MOC
// CT-e 4.00, Anexo II (modelo da p.51): cabeçalho com QR Code, prestação com o
// percurso do veículo, tomador, serviço, valores, imposto, observações, seguro e
// modal rodoviário. Depois do leiaute padrão vêm o fretamento e os documentos
// referenciados (BP-e, GTV-e), que continuam nas folhas seguintes.
//
// O canhoto do modelo é opcional (§2.21.5) e não é impresso: não há volumes a
// receber. [Opcoes.SemCanhoto] é ignorada aqui.
func DACTEOS(c *cteos.CTeOS, prot *cte.ProtCTe, opc Opcoes) ([]byte, error) {
	if c == nil {
		return nil, fmt.Errorf("dacteos: conhecimento ausente")
	}
	opc = emRetrato(opc)
	paginas, linhas, err := paginarReferenciados(c, prot, opc)
	if err != nil {
		return nil, fmt.Errorf("dacteos: %w", err)
	}
	documento := pdf.Novo()
	for i, docs := range paginas {
		pg := novaFolha(documento, opc)
		m := medidasDe(pg)
		if i == 0 {
			desenharPrimeiraFolhaOS(pg, m, c, prot, opc, docs, linhas[0], len(paginas))
			continue
		}
		y := desenharCabecalhoOS(pg, m, c, prot, opc, i+1, len(paginas), m.margem)
		desenharReferenciadosOS(pg, m, docs, linhas[1], y)
	}
	return documento.Bytes()
}

func desenharPrimeiraFolhaOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, prot *cte.ProtCTe, opc Opcoes, docs []docReferenciado, linhas, folhas int) float64 {
	y := desenharCabecalhoOS(pg, m, c, prot, opc, 1, folhas, m.margem)
	y = desenharPrestacaoOS(pg, m, c, y)
	y = desenharTomadorOS(pg, m, c, y)
	y = desenharServicoOS(pg, m, c, y)
	y = desenharValoresCTe(pg, m, c.InfCte.VPrest, y)
	y = desenharICMS(pg, m, &c.InfCte.Imp.ICMS, y)
	y = desenharTributosFederais(pg, m, c.InfCte.Imp.InfTribFed, y)
	y = desenharIBSCBS(pg, m, c.InfCte.Imp.IBSCBS, c.InfCte.Imp.VTotDFe, y)
	gerais, emissor, fisco := observacoesDoCTe(c.InfCte.Compl)
	y = quadroDeTexto(pg, m.margem, y, m.util, 10, "OBSERVAÇÕES GERAIS", gerais, 4)
	y = desenharSeguroOS(pg, m, c, y)
	y = desenharModalOS(pg, m, c, y)
	fimEmissor := quadroDeTexto(pg, m.margem, y, m.util*0.5, 10, "USO EXCLUSIVO DO EMISSOR DO CT-e", emissor, 3)
	fimFisco := quadroDeTexto(pg, m.margem+m.util*0.5, y, m.util*0.5, 10, "RESERVADO AO FISCO", fisco, 3)
	y = max(fimEmissor, fimFisco) + 1.5
	y = desenharAdicionaisOS(pg, m, c, y)
	return desenharReferenciadosOS(pg, m, docs, linhas, y)
}

// docReferenciado é uma linha da tabela de documentos referenciados.
type docReferenciado struct {
	numero, serie, emissao, valor, chave string
}

// paginarReferenciados mede, como o DACTE, quantas linhas de documentos
// referenciados cabem na primeira folha e nas demais.
func paginarReferenciados(c *cteos.CTeOS, prot *cte.ProtCTe, opc Opcoes) ([][]docReferenciado, [2]int, error) {
	docs := documentosReferenciados(c)
	rascunho := pdf.Novo()
	pg := novaFolha(rascunho, opc)
	m := medidasDe(pg)
	fim := desenharPrimeiraFolhaOS(pg, m, c, prot, opc, nil, 1, 1)
	if err := cabeNaFolha(fim, m); err != nil {
		return nil, [2]int{}, err
	}
	naPrimeira := 1 + max(int((m.altura-m.margem-fim)/alturaDocOrig), 0)
	topo := desenharCabecalhoOS(novaFolha(rascunho, opc), m, c, prot, opc, 2, 2, m.margem) + 3.4*2
	nasDemais := max(int((m.altura-m.margem-topo)/alturaDocOrig), 1)

	linhas := [2]int{naPrimeira, nasDemais}
	if len(docs) <= naPrimeira {
		return [][]docReferenciado{docs}, linhas, nil
	}
	paginas := [][]docReferenciado{docs[:naPrimeira]}
	for corte := naPrimeira; corte < len(docs); corte += nasDemais {
		paginas = append(paginas, docs[corte:min(corte+nasDemais, len(docs))])
	}
	return paginas, linhas, nil
}

func documentosReferenciados(c *cteos.CTeOS) []docReferenciado {
	norm := c.InfCte.InfCTeNorm
	if norm == nil {
		return nil
	}
	docs := make([]docReferenciado, 0, len(norm.InfDocRef)+len(norm.InfGTVe))
	for _, d := range norm.InfDocRef {
		linha := docReferenciado{numero: d.NDoc, serie: d.Serie, chave: chaveFormatada(d.ChBPe)}
		if d.DEmi != nil {
			linha.emissao = dataSimples(*d.DEmi)
		}
		if d.VDoc != nil {
			linha.valor = moeda(*d.VDoc)
		}
		docs = append(docs, linha)
	}
	// As GTV-e entram na mesma tabela: são documentos que a prestação
	// referencia, e separá-las em um bloco próprio só gastaria espaço.
	for _, g := range norm.InfGTVe {
		docs = append(docs, docReferenciado{numero: "GTV-e", chave: chaveFormatada(g.ChCTe)})
	}
	return docs
}

// desenharCabecalhoOS imprime o cabeçalho comum e as linhas do tipo do
// conhecimento, do serviço e da natureza da operação.
func desenharCabecalhoOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, prot *cte.ProtCTe, opc Opcoes, folha, folhas int, y float64) float64 {
	ide := &c.InfCte.Ide
	emit := &c.InfCte.Emit
	qrCode := ""
	if c.InfCTeSupl != nil {
		qrCode = c.InfCTeSupl.QrCodCTe
	}
	y = desenharIdentificacao(pg, m, identificacao{
		sigla: "DACTE OS", nome1: "Documento Auxiliar do Conhecimento de", nome2: "Transporte Eletrônico para Outros Serviços",
		modal:    "MODAL " + strings.ToUpper(ide.Modal.Descricao()),
		emitNome: emit.XNome, emitLinhas: linhasDoEmitente(emit.XFant, emit.EnderEmit, emit.CNPJ, emit.CPF, emit.IE),
		modelo: string(ide.Mod), serie: fmt.Sprintf("%03d", ide.Serie), numero: fmt.Sprintf("%09d", ide.NCT),
		emissao: dataHora(ide.DhEmi), folha: folha, folhas: folhas,
		chave: c.Chave(), protocolo: protocoloCTeDescrito(prot), consulta: URLConsultaCTe, qrCode: qrCode,
		situacao: situacaoCTe(ide.TpAmb, ide.TpEmis, prot != nil && prot.Autorizado(), opc, "CT-e OS"),
	}, y)
	caixa(pg, m.margem, y, m.util*0.25, 7, "TIPO DO CT-e", tipoDoCTe(ide.TpCTe), valor)
	caixa(pg, m.margem+m.util*0.25, y, m.util*0.30, 7, "TIPO DO SERVIÇO", servicoOS(ide.TpServ), valor)
	caixa(pg, m.margem+m.util*0.55, y, m.util*0.45, 7, "CFOP - NATUREZA DA OPERAÇÃO", ide.CFOP+" - "+ide.NatOp, valor)
	return y + 7
}

func servicoOS(t cteos.TipoServico) string { return string(t) + " - " + strings.ToUpper(t.Descricao()) }

// desenharPrestacaoOS imprime o início, o percurso do veículo e o término.
func desenharPrestacaoOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	ide := &c.InfCte.Ide
	percurso := make([]string, 0, len(ide.InfPercurso))
	for _, p := range ide.InfPercurso {
		percurso = append(percurso, p.UFPer)
	}
	caixa(pg, m.margem, y, m.util*0.34, 7, "INÍCIO DA PRESTAÇÃO", ide.XMunIni+" - "+ide.UFIni, valor)
	caixa(pg, m.margem+m.util*0.34, y, m.util*0.32, 7, "PERCURSO DO VEÍCULO", strings.Join(percurso, " - "), valor)
	caixa(pg, m.margem+m.util*0.66, y, m.util*0.34, 7, "TÉRMINO DA PRESTAÇÃO", ide.XMunFim+" - "+ide.UFFim, valor)
	return y + 7 + 1.5
}

// desenharTomadorOS imprime o tomador ou usuário do serviço.
func desenharTomadorOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	t := c.InfCte.Toma
	if t == nil {
		desenharParte(pg, m, m.margem, y, m.util, "TOMADOR / USUÁRIO DO SERVIÇO", nil)
		return y + alturaParte + 1.5
	}
	p := &parte{nome: t.XNome, documento: formatarDocumento(t.CNPJ, t.CPF), ie: t.IE, fone: t.Fone, endereco: t.EnderToma}
	desenharParte(pg, m, m.margem, y, m.util, "TOMADOR / USUÁRIO DO SERVIÇO", p)
	return y + alturaParte + 1.5
}

// desenharServicoOS imprime a quantidade, à esquerda como no modelo, e a
// descrição do serviço prestado.
func desenharServicoOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	norm := c.InfCte.InfCTeNorm
	if norm == nil {
		return y
	}
	quantia := ""
	if q := norm.InfServico.InfQ; q != nil {
		quantia = quantidade(q.QCarga)
	}
	y = cabecalhoDeSecao(pg, m, y, "INFORMAÇÕES DA PRESTAÇÃO DO SERVIÇO")
	fim := quadroDeTexto(pg, m.margem+m.util*0.18, y, m.util*0.82, 10, "DESCRIÇÃO DO SERVIÇO PRESTADO", norm.InfServico.XDescServ, 3)
	caixaDireita(pg, m.margem, y, m.util*0.18, fim-y, "QUANTIDADE", quantia, valorForte)
	return fim + 1.5
}

// desenharSeguroOS imprime o seguro da viagem.
func desenharSeguroOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	norm := c.InfCte.InfCTeNorm
	if norm == nil || len(norm.Seg) == 0 {
		return y
	}
	y = cabecalhoDeSecao(pg, m, y, "SEGURO DA VIAGEM")
	for _, s := range norm.Seg {
		caixa(pg, m.margem, y, m.util*0.30, 6.5, "RESPONSÁVEL", responsavelSeguro(s.RespSeg), valor)
		caixa(pg, m.margem+m.util*0.30, y, m.util*0.44, 6.5, "NOME DA SEGURADORA", s.XSeg, valor)
		caixa(pg, m.margem+m.util*0.74, y, m.util*0.26, 6.5, "NÚMERO DA APÓLICE", s.NApol, valor)
		y += 6.5
	}
	return y + 1.5
}

func responsavelSeguro(r cteos.ResponsavelSeguro) string {
	switch r {
	case cteos.SeguroTomador:
		return "4 - TOMADOR DO SERVIÇO"
	case cteos.SeguroEmitente:
		return "5 - EMITENTE DO CT-e OS"
	default:
		return string(r)
	}
}

// desenharModalOS imprime os campos do modal rodoviário de outros serviços que
// o manual traz: TAF, registro estadual, placa, RENAVAM e o documento e o TAF ou
// registro do proprietário.
func desenharModalOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	norm := c.InfCte.InfCTeNorm
	if norm == nil || norm.InfModal.RodoOS == nil {
		return y
	}
	rodo := norm.InfModal.RodoOS
	placa, renavam, propDoc, propReg := "", "", "", ""
	if v := rodo.Veic; v != nil {
		placa, renavam = v.Placa, v.RENAVAM
		if p := v.Prop; p != nil {
			propDoc = formatarDocumento(p.CNPJ, p.CPF)
			propReg = primeiroNaoVazio(p.TAF, p.NroRegEstadual)
		}
	}
	y = cabecalhoDeSecao(pg, m, y, "INFORMAÇÕES ESPECÍFICAS DO MODAL RODOVIÁRIO")
	campos := []struct {
		nome, conteudo string
		fracao         float64
	}{
		{"TERMO DE AUTORIZAÇÃO DE FRETAMENTO", rodo.TAF, 0.20}, {"Nº DO REGISTRO ESTADUAL", rodo.NroRegEstadual, 0.16},
		{"PLACA DO VEÍCULO", placa, 0.12}, {"RENAVAM DO VEÍCULO", renavam, 0.15},
		{"CNPJ/CPF DO PROPRIETÁRIO", propDoc, 0.19}, {"TAF / REG. ESTADUAL DO PROPRIETÁRIO", propReg, 0.18},
	}
	cursor := m.margem
	for _, campo := range campos {
		largura := m.util * campo.fracao
		caixa(pg, cursor, y, largura, 7, campo.nome, campo.conteudo, valor)
		cursor += largura
	}
	return y + 7 + 1.5
}

func primeiroNaoVazio(valores ...string) string {
	for _, v := range valores {
		if v != "" {
			return v
		}
	}
	return ""
}

func fretamento(t cteos.TipoFretamento) string {
	switch t {
	case cteos.FretamentoEventual:
		return "1 - EVENTUAL OU TURÍSTICO"
	case cteos.FretamentoContinuo:
		return "2 - CONTÍNUO"
	default:
		return string(t)
	}
}

// desenharAdicionaisOS imprime, depois do leiaute padrão, o que o manual não
// prevê: o proprietário do veículo, o fretamento, a partilha do ICMS e a
// informação adicional ao fisco.
func desenharAdicionaisOS(pg *pdf.Pagina, m medidas, c *cteos.CTeOS, y float64) float64 {
	var partes []string
	if norm := c.InfCte.InfCTeNorm; norm != nil && norm.InfModal.RodoOS != nil {
		rodo := norm.InfModal.RodoOS
		if v := rodo.Veic; v != nil {
			if v.UF != "" {
				partes = append(partes, "UF DO VEÍCULO: "+v.UF)
			}
			if p := v.Prop; p != nil {
				partes = append(partes, "PROPRIETÁRIO: "+p.XNome+" - "+p.UF)
			}
		}
		if f := rodo.InfFretamento; f != nil {
			texto := "FRETAMENTO: " + fretamento(f.TpFretamento)
			if f.DhViagem != nil {
				texto += ", VIAGEM EM " + dataHora(*f.DhViagem)
			}
			partes = append(partes, texto)
		}
	}
	if p := partilhaICMS(c.InfCte.Imp.ICMSUFFim); p != "" {
		partes = append(partes, p)
	}
	if ad := c.InfCte.Imp.InfAdFisco; ad != "" {
		partes = append(partes, "INFORMAÇÕES ADICIONAIS DE INTERESSE DO FISCO: "+ad)
	}
	if len(partes) == 0 {
		return y
	}
	return quadroDeTexto(pg, m.margem, y, m.util, 8, "INFORMAÇÕES ADICIONAIS", strings.Join(partes, " | "), 4) + 1.5
}

// desenharReferenciadosOS imprime a tabela de documentos referenciados (BP-e e
// GTV-e), com pelo menos minLinhas linhas.
func desenharReferenciadosOS(pg *pdf.Pagina, m medidas, docs []docReferenciado, minLinhas int, y float64) float64 {
	colunas := []struct {
		nome   string
		fracao float64
	}{
		{"NÚMERO", 0.14}, {"SÉRIE", 0.08}, {"EMISSÃO", 0.12}, {"VALOR", 0.12}, {"CHAVE DO BP-e / GTV-e", 0.54},
	}
	y = cabecalhoDeSecao(pg, m, y, "DOCUMENTOS REFERENCIADOS")
	cursor := m.margem
	for _, col := range colunas {
		largura := m.util * col.fracao
		pg.Retangulo(cursor, y, largura, 3.4, 0.1)
		pg.Texto(cursor+0.8, y+0.4, col.nome, rotulo)
		cursor += largura
	}
	y += 3.4
	if len(docs) == 0 {
		pg.Retangulo(m.margem, y, m.util, alturaDocOrig, 0.1)
		pg.Texto(m.margem+0.8, y+0.6, "SEM DOCUMENTOS REFERENCIADOS", miudo)
		return y + alturaDocOrig
	}
	celula := pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5}
	for i := 0; i < max(len(docs), minLinhas); i++ {
		var valores [5]string
		if i < len(docs) {
			d := docs[i]
			valores = [5]string{d.numero, d.serie, d.emissao, d.valor, d.chave}
		}
		cursor = m.margem
		for j, col := range colunas {
			largura := m.util * col.fracao
			pg.Retangulo(cursor, y, largura, alturaDocOrig, 0.1)
			pg.Texto(cursor+0.8, y+0.6, pdf.Encurtar(valores[j], celula.Fonte, celula.Tamanho, largura-1.6), celula)
			cursor += largura
		}
		y += alturaDocOrig
	}
	return y
}

// desenharTributosFederais imprime PIS, COFINS, IR, INSS e CSLL da prestação
// (infTribFed), quando o documento os traz.
func desenharTributosFederais(pg *pdf.Pagina, m medidas, f *cteos.InfTribFed, y float64) float64 {
	if f == nil {
		return y
	}
	texto := func(v *tipos.Decimal) string {
		if v == nil {
			return ""
		}
		return moeda(*v)
	}
	y = cabecalhoDeSecao(pg, m, y, "TRIBUTOS FEDERAIS")
	campos := [][2]string{
		{"VALOR DO PIS", texto(f.VPIS)}, {"VALOR DA COFINS", texto(f.VCOFINS)}, {"VALOR DO IR", texto(f.VIR)},
		{"VALOR DO INSS", texto(f.VINSS)}, {"VALOR DA CSLL", texto(f.VCSLL)},
	}
	largura := m.util / float64(len(campos))
	for i, c := range campos {
		caixaDireita(pg, m.margem+float64(i)*largura, y, largura, 7, c[0], c[1], valor)
	}
	return y + 7 + 1.5
}
