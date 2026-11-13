package danfe

import (
	"fmt"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/internal/pdf"
)

// A meia folha do DACTE é o A5 em paisagem, que o Manual do DACTE admite (MOC CT-e 4.00, Anexo II, §2.15). A
// disposição dos quadros é a do A4 em retrato, que tem a mesma largura; só as alturas diminuem (§2.16, medidasA5),
// menos a do código de barras e a do QR Code. Os quadros entram um depois do outro, e o que não cabe numa folha vai
// para a seguinte, que repete o cabeçalho (§2.12); a tabela de documentos originários se divide entre as folhas.
const (
	larguraA5 = 210.0
	alturaA5  = 148.0
)

// dacteA5 gera o DACTE em meia folha.
func dacteA5(c *cte.CTe, prot *cte.ProtCTe, opc Opcoes) ([]byte, error) {
	documento, err := meiasFolhasDACTE(c, prot, opc)
	if err != nil {
		return nil, fmt.Errorf("dacte: %w", err)
	}
	return documento.Bytes()
}

// meiasFolhasDACTE desenha o DACTE em meia folha e devolve o documento, com uma página A5 por folha.
func meiasFolhasDACTE(c *cte.CTe, prot *cte.ProtCTe, opc Opcoes) (*pdf.PDF, error) {
	// O campo FL traz o total de folhas: a primeira passada só conta, a segunda desenha.
	folhas, err := montarDACTEA5(pdf.Novo(), c, prot, opc, 0)
	if err != nil {
		return nil, err
	}
	documento := pdf.Novo()
	if _, err := montarDACTEA5(documento, c, prot, opc, folhas); err != nil {
		return nil, err
	}
	return documento, nil
}

// blocoA5 desenha um quadro a partir de y e devolve onde ele terminou.
type blocoA5 func(pg *pdf.Pagina, m medidas, y float64) float64

// folhasA5 distribui os quadros pelas folhas da meia folha.
type folhasA5 struct {
	documento *pdf.PDF
	rascunho  *pdf.Pagina // onde os quadros são medidos antes de entrar na folha
	c         *cte.CTe
	prot      *cte.ProtCTe
	opc       Opcoes
	folhas    int // total, para o campo FL; zero na passada que só conta

	pg      *pdf.Pagina
	m       medidas
	folha   int
	y, topo float64 // onde a folha está e onde os quadros começam, abaixo do cabeçalho
}

// nova abre a folha seguinte com o cabeçalho; o canhoto, quando pedido, só sai na primeira.
func (f *folhasA5) nova() {
	f.folha++
	f.pg = f.documento.NovaPagina(larguraA5, alturaA5)
	f.m = medidasA5(f.pg)
	f.y = f.m.margem
	if f.folha == 1 && f.opc.CanhotoNaMeiaFolha {
		f.y = desenharCanhotoCTe(f.pg, f.m, f.c, f.y)
	}
	f.y = desenharCabecalhoCTe(f.pg, f.m, f.c, f.prot, f.opc, f.folha, f.folhas, f.y)
	f.topo = f.y
}

// cabe diz se um quadro dessa altura cabe na folha atual, sem invadir a margem.
func (f *folhasA5) cabe(altura float64) bool {
	return f.y+altura <= f.m.altura-f.m.margem
}

// colocar desenha o quadro na folha atual ou, se ele não couber, na seguinte.
func (f *folhasA5) colocar(b blocoA5) error {
	altura := b(f.rascunho, f.m, 0)
	if !f.cabe(altura) && f.y > f.topo {
		f.nova()
	}
	if !f.cabe(altura) {
		return fmt.Errorf("%w: um quadro tem %.0f mm, e a meia folha tem %.0f mm abaixo do cabeçalho",
			ErrNaoCabe, altura, f.m.altura-f.m.margem-f.topo)
	}
	f.y = b(f.pg, f.m, f.y)
	return nil
}

// colocarDocumentos divide os documentos originários entre as folhas, duas por linha, e repete o título da tabela em
// cada folha.
func (f *folhasA5) colocarDocumentos(docs []docOrig) error {
	if len(docs) == 0 {
		return f.colocar(func(pg *pdf.Pagina, m medidas, y float64) float64 {
			return desenharDocumentosCTe(pg, m, nil, 0, y)
		})
	}
	for len(docs) > 0 {
		// O título e a linha das colunas, e então quantas linhas sobram.
		linhas := int((f.m.altura - f.m.margem - f.y - 2*f.m.secao) / f.m.linhaDoc)
		if linhas < 1 {
			if f.y == f.topo {
				return fmt.Errorf("%w: a tabela de documentos originários não cabe abaixo do cabeçalho", ErrNaoCabe)
			}
			f.nova()
			continue
		}
		n := min(2*linhas, len(docs))
		f.y = desenharDocumentosCTe(f.pg, f.m, docs[:n], 0, f.y)
		docs = docs[n:]
	}
	return nil
}

// montarDACTEA5 desenha o DACTE em meia folha no documento e devolve quantas folhas usou. Com folhas zero, o campo FL
// sai sem o total.
func montarDACTEA5(documento *pdf.PDF, c *cte.CTe, prot *cte.ProtCTe, opc Opcoes, folhas int) (int, error) {
	f := &folhasA5{
		documento: documento, rascunho: pdf.Novo().NovaPagina(larguraA5, alturaA5),
		c: c, prot: prot, opc: opc, folhas: folhas,
	}
	f.nova()

	exped, receb := parteDoExpedidor(c.InfCte.Exped), parteDoRecebedor(c.InfCte.Receb)
	antes := []blocoA5{
		func(pg *pdf.Pagina, m medidas, y float64) float64 { return desenharPrestacaoCTe(pg, m, c, y) },
		func(pg *pdf.Pagina, m medidas, y float64) float64 {
			metade := m.util / 2
			desenharParte(pg, m, m.margem, y, metade, "REMETENTE", parteDoRemetente(c.InfCte.Rem))
			desenharParte(pg, m, m.margem+metade, y, metade, "DESTINATÁRIO", parteDoDestinatario(c.InfCte.Dest))
			return y + m.parte
		},
	}
	if exped != nil || receb != nil {
		antes = append(antes, func(pg *pdf.Pagina, m medidas, y float64) float64 {
			metade := m.util / 2
			desenharParte(pg, m, m.margem, y, metade, "EXPEDIDOR", exped)
			desenharParte(pg, m, m.margem+metade, y, metade, "RECEBEDOR", receb)
			return y + m.parte
		})
	}
	antes = append(antes,
		func(pg *pdf.Pagina, m medidas, y float64) float64 { return desenharTomadorA5(pg, m, c, y) },
		func(pg *pdf.Pagina, m medidas, y float64) float64 { return desenharCargaCTe(pg, m, c, y) },
		func(pg *pdf.Pagina, m medidas, y float64) float64 {
			return desenharValoresCTe(pg, m, c.InfCte.VPrest, y)
		},
		func(pg *pdf.Pagina, m medidas, y float64) float64 { return desenharImpostoCTe(pg, m, &c.InfCte.Imp, y) },
		func(pg *pdf.Pagina, m medidas, y float64) float64 {
			return desenharIBSCBS(pg, m, c.InfCte.Imp.IBSCBS, c.InfCte.Imp.VTotDFe, y)
		},
	)
	depois := []blocoA5{
		func(pg *pdf.Pagina, m medidas, y float64) float64 {
			return desenharFluxo(pg, m, fluxoDe(c.InfCte.Compl), y)
		},
		func(pg *pdf.Pagina, m medidas, y float64) float64 { return desenharObservacoesCTe(pg, m, c, y) },
		func(pg *pdf.Pagina, m medidas, y float64) float64 { return desenharAdicionaisCTe(pg, m, c, y) },
	}

	for _, b := range antes {
		if err := f.colocar(b); err != nil {
			return 0, err
		}
	}
	if err := f.colocarDocumentos(documentosOriginarios(c)); err != nil {
		return 0, err
	}
	for _, b := range depois {
		if err := f.colocar(b); err != nil {
			return 0, err
		}
	}
	return f.folha, nil
}

// desenharTomadorA5 imprime o tomador. Quando ele é uma das partes, que já saíram completas acima (toma3), basta uma
// linha que o identifique; o tomador de fora (toma4) leva o quadro inteiro, com o endereço.
func desenharTomadorA5(pg *pdf.Pagina, m medidas, c *cte.CTe, y float64) float64 {
	t := tomador(c)
	if c.InfCte.Ide.Toma4 != nil || t == nil {
		desenharParte(pg, m, m.margem, y, m.util, "TOMADOR DO SERVIÇO", t)
		return y + m.parte + m.respiro
	}
	municipio := ""
	if t.endereco != nil {
		municipio = t.endereco.XMun + " - " + t.endereco.UF
	}
	x := m.margem
	for _, campo := range []struct {
		nome, conteudo string
		fracao         float64
	}{
		{"TOMADOR DO SERVIÇO", t.nome, 0.46},
		{"CNPJ / CPF", t.documento, 0.18},
		{"INSCRIÇÃO ESTADUAL", t.ie, 0.16},
		{"MUNICÍPIO", municipio, 0.20},
	} {
		caixa(pg, x, y, m.util*campo.fracao, m.campo, campo.nome, campo.conteudo, m.valorForte)
		x += m.util * campo.fracao
	}
	return y + m.campo + m.respiro
}

// desenharRodapeA5 imprime numa linha as observações, o uso exclusivo do emissor e o reservado ao fisco, com a mesma
// altura, e depois o modal rodoviário.
func desenharRodapeA5(pg *pdf.Pagina, m medidas, c *cte.CTe, gerais, emissor, fisco string, y float64) float64 {
	quadros := []struct {
		nome, texto string
		fracao      float64
	}{
		{"OBSERVAÇÕES", gerais, 0.5},
		{"USO EXCLUSIVO DO EMISSOR DO CT-e", emissor, 0.25},
		{"RESERVADO AO FISCO", fisco, 0.25},
	}
	altura := 6.0
	for _, q := range quadros {
		altura = max(altura, alturaDeTextoA5(q.texto, m.util*q.fracao, 3))
	}
	x := m.margem
	for _, q := range quadros {
		quadroDeTextoEm(pg, m, x, y, m.util*q.fracao, altura, q.nome, q.texto, 3)
		x += m.util * q.fracao
	}
	y += altura
	if norm := c.InfCte.InfCTeNorm; norm != nil && norm.InfModal.Rodo != nil {
		y = cabecalhoDeSecao(pg, m, y, "INFORMAÇÕES ESPECÍFICAS DO MODAL RODOVIÁRIO")
		caixa(pg, m.margem, y, m.util, m.campo, "RNTRC DA EMPRESA", norm.InfModal.Rodo.RNTRC, m.valor)
		y += m.campo
	}
	return y + m.respiro
}
