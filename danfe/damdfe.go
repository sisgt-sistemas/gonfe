package danfe

import (
	"fmt"
	"strings"
	"time"

	"github.com/sisgt-sistemas/gonfe/internal/pdf"
	"github.com/sisgt-sistemas/gonfe/mdfe"
	"github.com/sisgt-sistemas/gonfe/validacao"
)

// alturaDocMDFe é o espaço vertical de cada linha da tabela de documentos.
const alturaDocMDFe = 3.6

// URLConsultaMDFe é o endereço de consulta impresso no quadro "Controle do
// fisco" (MOC MDF-e 3.00b, Anexo II, §2.7).
const URLConsultaMDFe = "https://dfe-portal.sefazvirtual.rs.gov.br/MDFe/consulta"

// GerarDAMDFE produz o documento auxiliar a partir do XML de distribuição do
// MDF-e, o mdfeProc.
func GerarDAMDFE(procMDFe []byte, opc Opcoes) ([]byte, error) {
	m, prot, err := mdfe.LerMDFeProc(procMDFe)
	if err != nil {
		return nil, fmt.Errorf("damdfe: %w", err)
	}
	return DAMDFE(m, prot, opc)
}

// DAMDFE gera o documento auxiliar do MDF-e, modelo 58, em A4, pelo MOC MDF-e
// 3.00b, Anexo II (modelo rodoviário, §2.7.1 e 2.7.2): emitente com RNTRC, QR
// Code, identificação, totais, controle do fisco com a chave, protocolo (ou o
// aviso de contingência), veículos, condutores, vale-pedágio e observações.
//
// O que o manual não prevê vem abaixo do leiaute padrão, como ele permite
// (§2.1): viagem, CIOT, dados dos veículos, carga, seguro e a relação dos
// documentos por município de descarregamento, que continua nas folhas
// seguintes. O DAMDFE não tem canhoto.
func DAMDFE(m *mdfe.MDFe, prot *mdfe.ProtMDFe, opc Opcoes) ([]byte, error) {
	if m == nil {
		return nil, fmt.Errorf("damdfe: manifesto ausente")
	}
	opc = emRetrato(opc)
	paginas, err := paginarDocumentosMDFe(m, prot, opc)
	if err != nil {
		return nil, fmt.Errorf("damdfe: %w", err)
	}
	documento := pdf.Novo()
	for i, docs := range paginas {
		pg := novaFolha(documento, opc)
		md := medidasDe(pg)
		if i == 0 {
			// No estilo leve do manual, a relação não é preenchida até o pé com linhas em branco.
			desenharPrimeiraFolhaMDFe(pg, md, m, prot, opc, docs, 0, len(paginas))
			continue
		}
		y := desenharCabecalhoMDFe(pg, md, m, i+1, len(paginas), md.margem)
		desenharDocumentosMDFe(pg, md, docs, 0, y+2)
	}
	return documento.Bytes()
}

func desenharPrimeiraFolhaMDFe(pg *pdf.Pagina, md medidas, m *mdfe.MDFe, prot *mdfe.ProtMDFe, opc Opcoes, docs []docMDFe, linhas, folhas int) float64 {
	y := desenharCabecalhoMDFe(pg, md, m, 1, folhas, md.margem)
	y = desenharControleMDFe(pg, md, m, prot, opc, y+3)
	y = desenharVeiculosECondutores(pg, md, m, y+3)
	y = desenharObservacoesMDFe(pg, md, m, y+3)
	y = desenharAdicionaisMDFe(pg, md, m, y+4)
	return desenharDocumentosMDFe(pg, md, docs, linhas, y+2)
}

// Estilos do DAMDFE, mais leves que os do DACTE, como no modelo do manual.
var (
	tituloSecao  = pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 9}
	rotuloCampo  = pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5}
	valorCampo   = pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 8}
	textoComum   = pdf.Estilo{Fonte: pdf.Normal, Tamanho: 6.5}
	textoNegrito = pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 6.5}
)

// campoClaro desenha um campo do modelo: fundo claro, rótulo pequeno e valor em
// destaque.
func campoClaro(pg *pdf.Pagina, x, y, largura float64, nome, conteudo string) {
	const altura = 8
	pg.RetanguloPreenchido(x, y, largura-0.8, altura, 0.92)
	pg.Texto(x+1, y+0.8, nome, rotuloCampo)
	pg.Texto(x+1, y+4.2, pdf.Encurtar(conteudo, valorCampo.Fonte, valorCampo.Tamanho, largura-2.6), valorCampo)
}

// tracejado desenha um separador horizontal tracejado.
func tracejado(pg *pdf.Pagina, x, y, largura float64) {
	for cursor := x; cursor < x+largura; cursor += 2 {
		pg.Linha(cursor, y, min(cursor+1, x+largura), y, 0.1)
	}
}

// desenharCabecalhoMDFe imprime o emitente, o título, a linha de identificação
// e o QR Code. Repete-se no alto de cada folha.
func desenharCabecalhoMDFe(pg *pdf.Pagina, md medidas, m *mdfe.MDFe, folha, folhas int, y float64) float64 {
	emit := &m.InfMDFe.Emit
	ide := &m.InfMDFe.Ide
	rntrc := ""
	if r := m.InfMDFe.InfModal.Rodo; r != nil && r.InfANTT != nil {
		rntrc = r.InfANTT.RNTRC
	}
	larguraTexto := md.util - ladoCaixaQR

	pg.Texto(md.margem, y+1, pdf.Encurtar(emit.XNome, pdf.Negrito, 9, larguraTexto), pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 9})
	end := emit.EnderEmit
	pg.Texto(md.margem, y+5.5, enderecoEmUmaLinha(end.XLgr, end.Nro, end.XCpl)+" - "+end.XBairro, textoComum)
	pg.Texto(md.margem, y+8.8, end.XMun+" - "+end.UF+"    CEP "+validacao.FormatarCEP(end.CEP), textoComum)
	linhaDocs := "CNPJ: " + formatarDocumento(emit.CNPJ, emit.CPF) + "    IE: " + emit.IE
	if rntrc != "" {
		linhaDocs += "    RNTRC: " + rntrc
	}
	pg.Texto(md.margem, y+12.1, linhaDocs, textoNegrito)
	pg.Texto(md.margem, y+17, "DAMDFE - Documento Auxiliar de Manifesto Eletrônico de Documentos Fiscais", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7.5})

	qrCode := ""
	if m.InfMDFeSupl != nil {
		qrCode = m.InfMDFeSupl.QrCodMDFe
	}
	xQR := md.margem + larguraTexto
	if matriz, err := matrizQR(qrCode); err == nil {
		pg.QRCode(xQR+(ladoCaixaQR-ladoQR)/2, y+1, ladoQR, matriz)
	} else {
		pg.TextoCentralizado(xQR, ladoCaixaQR, y+10, "QR CODE NÃO INFORMADO", miudo)
	}

	y += 22
	cursor := md.margem
	for _, c := range []struct {
		nome, conteudo string
		largura        float64
	}{
		{"Modelo", string(ide.Mod), 13}, {"Série", fmt.Sprintf("%d", ide.Serie), 12},
		{"Número", fmt.Sprintf("%09d", ide.NMDF), 20}, {"FL", fmtFolha(folha, folhas), 14},
		{"Data e hora de Emissão", dataHora(ide.DhEmi), 42}, {"UF Carreg.", ide.UFIni, 20},
		{"UF Descarreg.", ide.UFFim, 22},
	} {
		campoClaro(pg, cursor, y, c.largura, c.nome, c.conteudo)
		cursor += c.largura
	}
	return y + 8
}

// desenharControleMDFe imprime os totais do modal, o controle do fisco (código
// de barras, chave e consulta) e o protocolo, ou o aviso de contingência no
// lugar dele (§2.4).
func desenharControleMDFe(pg *pdf.Pagina, md medidas, m *mdfe.MDFe, prot *mdfe.ProtMDFe, opc Opcoes, y float64) float64 {
	metade := md.util / 2
	tot := &m.InfMDFe.Tot
	pg.Texto(md.margem, y, "Modelo Rodoviário de Carga", tituloSecao)
	terco := (metade - 4) / 3
	campoClaro(pg, md.margem, y+5, terco, "Qtd. CTe", fmt.Sprintf("%d", tot.QCTe))
	campoClaro(pg, md.margem+terco, y+5, terco, "Qtd. NFe", fmt.Sprintf("%d", tot.QNFe))
	campoClaro(pg, md.margem+2*terco, y+5, terco, "Peso total ("+unidadeMDFe(tot.CUnid)+")", quantidade(tot.QCarga))

	x := md.margem + metade
	pg.Texto(x, y, "CONTROLE DO FISCO", rotuloCampo)
	if ch := m.Chave(); len(ch) == 44 {
		if err := pg.CodigoDeBarras(x+2, y+3, larguraBarras, alturaBarras, ch); err != nil {
			pg.Texto(x, y+8, "código de barras indisponível", miudo)
		}
	}
	yChave := y + 3 + alturaBarras + 2
	pg.Texto(x, yChave, "Chave de Acesso", textoNegrito)
	pg.Texto(x, yChave+3.4, chaveFormatada(m.Chave()), pdf.Estilo{Fonte: pdf.Normal, Tamanho: 7})
	pg.Texto(x, yChave+7.2, "Consulte em "+URLConsultaMDFe, textoComum)

	// Protocolo, ou o aviso que o substitui.
	ide := &m.InfMDFe.Ide
	yProt := y + 17
	switch {
	case opc.Cancelada:
		avisoMDFe(pg, md.margem, yProt, metade-4, "MDF-e CANCELADO", "")
	case ide.TpEmis.Contingencia():
		limite := ide.DhEmi.Add(168 * time.Hour)
		avisoMDFe(pg, md.margem, yProt, metade-4, "EMISSÃO EM CONTINGÊNCIA",
			"Obrigatória a autorização em 168 horas após esta emissão ("+limite.Format("02/01/2006 15:04")+")")
	default:
		pg.Texto(md.margem, yProt, "Protocolo de autorização", textoNegrito)
		pg.Texto(md.margem, yProt+3.6, protocoloMDFeDescrito(prot), pdf.Estilo{Fonte: pdf.Normal, Tamanho: 7.5})
	}
	if opc.Homologacao || ide.TpAmb == mdfe.Homologacao {
		pg.Texto(md.margem, yProt+8, "EMITIDO EM AMBIENTE DE HOMOLOGAÇÃO – SEM VALOR FISCAL", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7})
	} else if !opc.Cancelada && !ide.TpEmis.Contingencia() && (prot == nil || !prot.Autorizado()) {
		pg.Texto(md.margem, yProt+8, "SEM VALOR FISCAL", pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7})
	}
	return max(yChave+9, yProt+11)
}

// avisoMDFe desenha uma caixa escura com o aviso em branco, como o de
// contingência do modelo do manual.
func avisoMDFe(pg *pdf.Pagina, x, y, largura float64, titulo, detalhe string) {
	altura := 6.0
	if detalhe != "" {
		altura = 9
	}
	pg.RetanguloPreenchido(x, y-1, largura, altura, 0.15)
	pg.Texto(x+1.5, y, titulo, pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 8, Cinza: 1})
	if detalhe != "" {
		pg.Texto(x+1.5, y+4, pdf.Encurtar(detalhe, pdf.Normal, 5.5, largura-3), pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5, Cinza: 1})
	}
}

func protocoloMDFeDescrito(prot *mdfe.ProtMDFe) string {
	if prot == nil || prot.InfProt.NProt == "" {
		return "DOCUMENTO SEM PROTOCOLO DE AUTORIZAÇÃO"
	}
	return prot.InfProt.NProt + " - " + dataHora(prot.InfProt.DhRecbto)
}

// desenharVeiculosECondutores imprime, lado a lado, os veículos (placa e
// RNTRC) com o vale-pedágio embaixo, e os condutores (CPF e nome).
func desenharVeiculosECondutores(pg *pdf.Pagina, md medidas, m *mdfe.MDFe, y float64) float64 {
	rodo := m.InfMDFe.InfModal.Rodo
	if rodo == nil {
		return y
	}
	metade := md.util / 2
	rntrcEmitente := ""
	if rodo.InfANTT != nil {
		rntrcEmitente = rodo.InfANTT.RNTRC
	}
	rntrcDe := func(p *mdfe.Proprietario) string {
		if p != nil && p.RNTRC != "" {
			return p.RNTRC
		}
		return rntrcEmitente
	}
	veiculos := [][]string{{rodo.VeicTracao.Placa, rntrcDe(rodo.VeicTracao.Prop)}}
	for _, r := range rodo.VeicReboque {
		veiculos = append(veiculos, []string{r.Placa, rntrcDe(r.Prop)})
	}
	condutores := make([][]string, 0, len(rodo.VeicTracao.Condutor))
	for _, c := range rodo.VeicTracao.Condutor {
		condutores = append(condutores, []string{validacao.FormatarCPF(c.CPF), c.XNome})
	}

	fimEsquerda := tabelaLeve(pg, md.margem, y, metade-4, "Veículo", []string{"Placa", "RNTRC"}, []float64{0.4, 0.6}, veiculos)
	fimDireita := tabelaLeve(pg, md.margem+metade, y, metade, "Condutor", []string{"CPF", "Nome"}, []float64{0.32, 0.68}, condutores)

	if rodo.InfANTT != nil && rodo.InfANTT.ValePed != nil && len(rodo.InfANTT.ValePed.Disp) > 0 {
		vales := make([][]string, 0, len(rodo.InfANTT.ValePed.Disp))
		for _, d := range rodo.InfANTT.ValePed.Disp {
			vales = append(vales, []string{formatarDocumento(d.CNPJPg, d.CPFPg), formatarDocumento(d.CNPJForn, ""), d.NCompra})
		}
		fimEsquerda = tabelaLeve(pg, md.margem, fimEsquerda+3, metade-4, "Vale Pedágio",
			[]string{"Responsável CNPJ", "Fornecedor CNPJ", "Nº Comprovante"}, []float64{0.32, 0.32, 0.36}, vales)
	}
	return max(fimEsquerda, fimDireita)
}

// tabelaLeve desenha uma seção do modelo do manual: título, cabeçalho com linha
// embaixo e linhas separadas por tracejado.
func tabelaLeve(pg *pdf.Pagina, x, y, largura float64, titulo string, colunas []string, fracoes []float64, linhas [][]string) float64 {
	pg.Texto(x, y, titulo, tituloSecao)
	y += 5
	cursor := x
	for i, c := range colunas {
		pg.Texto(cursor, y, c, rotuloCampo)
		cursor += largura * fracoes[i]
	}
	y += 3
	pg.Linha(x, y, x+largura, y, 0.2)
	celula := pdf.Estilo{Fonte: pdf.Negrito, Tamanho: 7}
	for i, linha := range linhas {
		cursor = x
		for j, conteudo := range linha {
			pg.Texto(cursor, y+1.2, pdf.Encurtar(conteudo, celula.Fonte, celula.Tamanho, largura*fracoes[j]-1), celula)
			cursor += largura * fracoes[j]
		}
		y += 5
		if i < len(linhas)-1 {
			tracejado(pg, x, y, largura)
		}
	}
	return y
}

// desenharObservacoesMDFe imprime o último quadro do leiaute padrão: as
// informações complementares do manifesto.
func desenharObservacoesMDFe(pg *pdf.Pagina, md medidas, m *mdfe.MDFe, y float64) float64 {
	var partes []string
	if ad := m.InfMDFe.InfAdic; ad != nil {
		for _, campo := range []string{ad.InfCpl, ad.InfAdFisco} {
			if campo != "" {
				partes = append(partes, campo)
			}
		}
	}
	pg.Texto(md.margem, y, "Observações", textoNegrito)
	y += 3.2
	pg.Linha(md.margem, y, md.margem+md.util, y, 0.2)
	if len(partes) == 0 {
		return y + 3
	}
	return pg.TextoQuebrado(md.margem, y+1.5, md.util, strings.Join(partes, " | "), textoComum, 6) + 1.5
}

// desenharAdicionaisMDFe imprime, abaixo do leiaute padrão, o que o manual não
// prevê: viagem, CIOT, veículos, carga e seguro.
func desenharAdicionaisMDFe(pg *pdf.Pagina, md medidas, m *mdfe.MDFe, y float64) float64 {
	ide := &m.InfMDFe.Ide
	tot := &m.InfMDFe.Tot
	// juntar monta uma linha só com os pares que têm valor.
	juntar := func(pares ...string) string {
		var partes []string
		for i := 0; i+1 < len(pares); i += 2 {
			if strings.TrimSpace(pares[i+1]) != "" {
				partes = append(partes, pares[i]+": "+pares[i+1])
			}
		}
		return strings.Join(partes, "    ")
	}
	qtdMDFe := ""
	if tot.QMDFe > 0 {
		qtdMDFe = fmt.Sprintf("%d", tot.QMDFe)
	}
	linhas := []string{
		juntar("Emitente", tipoDoEmitente(ide.TpEmit), "Transportador", tipoDoTransportador(ide.TpTransp)),
		juntar("Carregamento", municipiosDeCarregamento(ide.InfMunCarrega), "Percurso", percurso(ide.InfPercurso),
			"Início da viagem", dataHoraOpcional(ide.DhIniViagem)),
		juntar("Valor total da carga", "R$ "+moeda(tot.VCarga), "Qtd. MDF-e", qtdMDFe),
	}
	if p := m.InfMDFe.ProdPred; p != nil {
		linhas = append(linhas, "Produto predominante: "+p.XProd)
	}
	if rodo := m.InfMDFe.InfModal.Rodo; rodo != nil {
		if antt := rodo.InfANTT; antt != nil && len(antt.InfCIOT) > 0 {
			ciots := make([]string, 0, len(antt.InfCIOT))
			for _, c := range antt.InfCIOT {
				ciots = append(ciots, c.CIOT)
			}
			linhas = append(linhas, "CIOT: "+strings.Join(ciots, ", "))
		}
		if rodo.CodAgPorto != "" {
			linhas = append(linhas, "Código de agendamento no porto: "+rodo.CodAgPorto)
		}
		t := rodo.VeicTracao
		linhas = append(linhas, "Tração "+t.Placa+" - "+juntar("RENAVAM", t.RENAVAM, "Tara (kg)", fmt.Sprint(t.Tara),
			"Capacidade (kg)", fmt.Sprint(t.CapKG), "Rodado", rodado(t.TpRod), "Carroceria", carroceria(t.TpCar)))
		for _, r := range rodo.VeicReboque {
			linhas = append(linhas, "Reboque "+r.Placa+" - "+juntar("RENAVAM", r.RENAVAM, "Tara (kg)", fmt.Sprint(r.Tara),
				"Capacidade (kg)", fmt.Sprint(r.CapKG), "Carroceria", carroceria(r.TpCar)))
		}
	}
	for _, s := range m.InfMDFe.Seg {
		texto := "Seguro: apólice " + s.NApol
		if s.InfSeg != nil {
			texto += ", " + s.InfSeg.XSeg + " (" + validacao.FormatarCNPJ(s.InfSeg.CNPJ) + ")"
		}
		if len(s.NAver) > 0 {
			texto += ", averbação " + strings.Join(s.NAver, ", ")
		}
		linhas = append(linhas, texto)
	}

	pg.Texto(md.margem, y, "Informações adicionais", textoNegrito)
	y += 3.2
	pg.Linha(md.margem, y, md.margem+md.util, y, 0.2)
	y += 1.5
	for _, l := range linhas {
		pg.Texto(md.margem, y, pdf.Encurtar(l, textoComum.Fonte, textoComum.Tamanho, md.util), textoComum)
		y += 3.4
	}
	return y
}

// docMDFe é uma linha da relação de documentos.
type docMDFe struct {
	municipio, tipo, chave string
}

// paginarDocumentosMDFe mede quantas linhas da relação de documentos cabem na
// primeira folha e nas demais, e divide a relação entre as folhas.
func paginarDocumentosMDFe(m *mdfe.MDFe, prot *mdfe.ProtMDFe, opc Opcoes) ([][]docMDFe, error) {
	docs := documentosDoManifesto(m)
	rascunho := pdf.Novo()
	pg := novaFolha(rascunho, opc)
	md := medidasDe(pg)
	fim := desenharPrimeiraFolhaMDFe(pg, md, m, prot, opc, nil, 1, 1)
	if err := cabeNaFolha(fim, md); err != nil {
		return nil, err
	}
	naPrimeira := 1 + max(int((md.altura-md.margem-fim)/alturaDocMDFe), 0)
	topo := desenharCabecalhoMDFe(novaFolha(rascunho, opc), md, m, 2, 2, md.margem) + 2 + 3.4*2
	nasDemais := max(int((md.altura-md.margem-topo)/alturaDocMDFe), 1)

	if len(docs) <= naPrimeira {
		return [][]docMDFe{docs}, nil
	}
	paginas := [][]docMDFe{docs[:naPrimeira]}
	for corte := naPrimeira; corte < len(docs); corte += nasDemais {
		paginas = append(paginas, docs[corte:min(corte+nasDemais, len(docs))])
	}
	return paginas, nil
}

// documentosDoManifesto achata os grupos por município de descarregamento em
// uma lista única, preservando a ordem do documento.
func documentosDoManifesto(m *mdfe.MDFe) []docMDFe {
	var docs []docMDFe
	for _, mun := range m.InfMDFe.InfDoc.InfMunDescarga {
		local := mun.XMunDescarga
		for _, d := range mun.InfCTe {
			docs = append(docs, docMDFe{local, "CT-e", chaveFormatada(d.ChCTe)})
		}
		for _, d := range mun.InfNFe {
			docs = append(docs, docMDFe{local, "NF-e", chaveFormatada(d.ChNFe)})
		}
		for _, d := range mun.InfMDFeTransp {
			docs = append(docs, docMDFe{local, "MDF-e", chaveFormatada(d.ChMDFe)})
		}
	}
	return docs
}

// desenharDocumentosMDFe imprime a relação de documentos por município de
// descarregamento, com pelo menos minLinhas linhas.
func desenharDocumentosMDFe(pg *pdf.Pagina, md medidas, docs []docMDFe, minLinhas int, y float64) float64 {
	colunas := []struct {
		nome   string
		fracao float64
	}{{"MUNICÍPIO DE DESCARREGAMENTO", 0.32}, {"TIPO", 0.10}, {"CHAVE DE ACESSO", 0.58}}
	y = cabecalhoDeSecao(pg, md, y, "DOCUMENTOS VINCULADOS")
	cursor := md.margem
	for _, c := range colunas {
		largura := md.util * c.fracao
		pg.Retangulo(cursor, y, largura, 3.4, 0.1)
		pg.Texto(cursor+0.8, y+0.4, c.nome, rotulo)
		cursor += largura
	}
	y += 3.4
	if len(docs) == 0 {
		pg.Retangulo(md.margem, y, md.util, alturaDocMDFe, 0.1)
		pg.Texto(md.margem+0.8, y+0.6, "SEM DOCUMENTOS VINCULADOS", miudo)
		return y + alturaDocMDFe
	}
	celula := pdf.Estilo{Fonte: pdf.Normal, Tamanho: 5.5}
	anterior := ""
	for i := 0; i < max(len(docs), minLinhas); i++ {
		var valores [3]string
		if i < len(docs) {
			d := docs[i]
			valores = [3]string{d.municipio, d.tipo, d.chave}
			// O município sai só na primeira linha do grupo.
			if d.municipio == anterior {
				valores[0] = ""
			}
			anterior = d.municipio
		}
		cursor = md.margem
		for j, c := range colunas {
			largura := md.util * c.fracao
			pg.Retangulo(cursor, y, largura, alturaDocMDFe, 0.1)
			pg.Texto(cursor+0.8, y+0.6, pdf.Encurtar(valores[j], celula.Fonte, celula.Tamanho, largura-1.6), celula)
			cursor += largura
		}
		y += alturaDocMDFe
	}
	return y
}

func tipoDoEmitente(t mdfe.TipoEmitente) string {
	switch t {
	case mdfe.EmitentePrestadorServico:
		return "1 - PRESTADOR DE SERVIÇO DE TRANSPORTE"
	case mdfe.EmitenteCargaPropria:
		return "2 - CARGA PRÓPRIA"
	case mdfe.EmitenteCTeGlobalizado:
		return "3 - EMITENTE DE CT-E GLOBALIZADO"
	default:
		return string(t)
	}
}

func tipoDoTransportador(t mdfe.TipoTransportador) string {
	switch t {
	case mdfe.TransportadorETC:
		return "1 - ETC"
	case mdfe.TransportadorTAC:
		return "2 - TAC"
	case mdfe.TransportadorCTC:
		return "3 - CTC"
	default:
		return string(t)
	}
}

func municipiosDeCarregamento(muns []mdfe.InfMunCarrega) string {
	nomes := make([]string, 0, len(muns))
	for _, mun := range muns {
		nomes = append(nomes, mun.XMunCarrega)
	}
	return strings.Join(nomes, ", ")
}

func percurso(ufs []mdfe.InfPercurso) string {
	siglas := make([]string, 0, len(ufs))
	for _, u := range ufs {
		siglas = append(siglas, u.UFPer)
	}
	return strings.Join(siglas, " - ")
}

func rodado(t mdfe.TipoRodado) string {
	nomes := map[mdfe.TipoRodado]string{
		mdfe.RodadoTruck:      "TRUCK",
		mdfe.RodadoToco:       "TOCO",
		mdfe.RodadoCavaloMec:  "CAVALO",
		mdfe.RodadoVAN:        "VAN",
		mdfe.RodadoUtilitario: "UTILITÁRIO",
		mdfe.RodadoOutros:     "OUTROS",
	}
	if nome, ok := nomes[t]; ok {
		return nome
	}
	return string(t)
}

func carroceria(t mdfe.TipoCarroceria) string {
	nomes := map[mdfe.TipoCarroceria]string{
		mdfe.CarroceriaNaoAplicavel:   "N/A",
		mdfe.CarroceriaAberta:         "ABERTA",
		mdfe.CarroceriaFechadaBau:     "BAÚ",
		mdfe.CarroceriaGraneleira:     "GRANELEIRA",
		mdfe.CarroceriaPortaContainer: "PORTA-CONT.",
		mdfe.CarroceriaSider:          "SIDER",
	}
	if nome, ok := nomes[t]; ok {
		return nome
	}
	return string(t)
}

func unidadeMDFe(u mdfe.UnidadeDeMedida) string {
	switch u {
	case mdfe.UnidadeKG:
		return "KG"
	case mdfe.UnidadeTON:
		return "TON"
	default:
		return string(u)
	}
}
