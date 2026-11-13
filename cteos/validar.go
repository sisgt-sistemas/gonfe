package cteos

import (
	"fmt"
	"strings"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/tipos"
	"github.com/sisgt-sistemas/gonfe/uf"
	"github.com/sisgt-sistemas/gonfe/validacao"
)

// Erro é uma inconsistência encontrada por [CTeOS.Validar].
type Erro struct {
	Campo    string
	Mensagem string
}

func (e Erro) Error() string { return e.Campo + ": " + e.Mensagem }

// Erros é o conjunto de inconsistências de um conhecimento.
type Erros []Erro

func (e Erros) Error() string {
	switch len(e) {
	case 0:
		return "cteos: nenhum erro"
	case 1:
		return "cteos: " + e[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "cteos: %d inconsistências:", len(e))
	for _, item := range e {
		b.WriteString("\n  - ")
		b.WriteString(item.Error())
	}
	return b.String()
}

// ToleranciaCentavos é a diferença máxima aceita nas conferências de
// somatórios.
var ToleranciaCentavos = tipos.D("0.01")

// Validar confere o conhecimento contra as regras estruturais do leiaute 4.00.
//
// A validação cobre o que dá para verificar sem consultar a SEFAZ: presença e
// formato dos campos obrigatórios, coerência entre grupos e dígitos
// verificadores. Ela não substitui as regras de negócio que a SEFAZ aplica na
// autorização.
func (c *CTeOS) Validar() error {
	var e Erros
	e = append(e, c.validarIde()...)
	e = append(e, c.validarEmitente()...)
	e = append(e, c.validarTomador()...)
	e = append(e, c.validarPrestacao()...)
	e = append(e, c.validarImposto()...)
	e = append(e, c.validarNorm()...)
	e = append(e, c.validarForaDoLeiaute()...)
	if len(e) == 0 {
		return nil
	}
	return e
}

func erro(campo, formato string, args ...any) Erro {
	return Erro{Campo: campo, Mensagem: fmt.Sprintf(formato, args...)}
}

func (c *CTeOS) validarIde() Erros {
	var e Erros
	ide := &c.InfCte.Ide

	if c.InfCte.Versao != Versao {
		e = append(e, erro("infCte@versao", "versão %q; este pacote implementa a %s", c.InfCte.Versao, Versao))
	}
	if ide.Mod != cte.ModeloCTeOS {
		e = append(e, erro("ide.mod", "modelo %q; este pacote implementa o 67", ide.Mod))
	}
	if ide.NatOp == "" {
		e = append(e, erro("ide.natOp", "natureza da operação é obrigatória"))
	}
	if len(ide.CFOP) != 4 {
		e = append(e, erro("ide.CFOP", "CFOP %q; informe 4 dígitos", ide.CFOP))
	}
	if ide.Serie < 0 || ide.Serie > 999 {
		e = append(e, erro("ide.serie", "série %d fora da faixa 0–999", ide.Serie))
	}
	if ide.NCT < 1 || ide.NCT > 999999999 {
		e = append(e, erro("ide.nCT", "número %d fora da faixa 1–999999999", ide.NCT))
	}
	if ide.DhEmi.Vazia() {
		e = append(e, erro("ide.dhEmi", "data e hora de emissão são obrigatórias"))
	}
	if ide.TpAmb != cte.Producao && ide.TpAmb != cte.Homologacao {
		e = append(e, erro("ide.tpAmb", "ambiente %q; use 1 (produção) ou 2 (homologação)", ide.TpAmb))
	}
	if ide.Modal != cte.ModalRodoviario {
		e = append(e, erro("ide.modal", "o CT-e OS só existe no modal rodoviário; veio %q", ide.Modal))
	}
	switch ide.TpServ {
	case ServicoTransportePessoas, ServicoTransporteValores, ServicoExcessoBagagem:
	default:
		e = append(e, erro("ide.tpServ",
			"tipo de serviço %q; o CT-e OS aceita 6 (pessoas), 7 (valores) ou 8 (excesso de bagagem)",
			ide.TpServ))
	}
	if ide.VerProc == "" {
		e = append(e, erro("ide.verProc", "versão do aplicativo emissor é obrigatória"))
	}

	for _, m := range []struct {
		campo   string
		codigo  int
		nome    string
		unidade string
	}{
		{"ide.cMunEnv", ide.CMunEnv, ide.XMunEnv, ide.UFEnv},
		{"ide.cMunIni", ide.CMunIni, ide.XMunIni, ide.UFIni},
		{"ide.cMunFim", ide.CMunFim, ide.XMunFim, ide.UFFim},
	} {
		if m.codigo == 0 {
			e = append(e, erro(m.campo, "código do município é obrigatório"))
		} else if m.codigo < 1000000 || m.codigo > 9999999 {
			e = append(e, erro(m.campo, "código de município %d não tem 7 dígitos", m.codigo))
		}
		if m.nome == "" {
			e = append(e, erro(m.campo, "nome do município é obrigatório"))
		}
		if _, err := uf.PorSigla(m.unidade); err != nil {
			e = append(e, erro(m.campo, "UF: %v", err))
		}
	}

	// O percurso do veículo: UF válidas, até 25 (infPercurso, maxOccurs 25).
	if len(ide.InfPercurso) > 25 {
		e = append(e, erro("ide.infPercurso", "%d UF no percurso; o leiaute aceita até 25", len(ide.InfPercurso)))
	}
	for i, p := range ide.InfPercurso {
		if _, err := uf.PorSigla(p.UFPer); err != nil {
			e = append(e, erro(fmt.Sprintf("ide.infPercurso[%d].UFPer", i), "%v", err))
		}
	}

	// O TCTeOS aceita menos formas que o CT-e: sem o Regime Especial NFF (3) e sem EPEC (4).
	switch ide.TpEmis {
	case cte.EmissaoNormal, cte.EmissaoContingenciaOffline, cte.EmissaoSVCRS, cte.EmissaoSVCSP:
	default:
		e = append(e, erro("ide.tpEmis", "forma de emissão %q fora do leiaute do CT-e OS (1, 2, 7 ou 8)", ide.TpEmis))
	}
	e = append(e, deCTe(cte.ValidarCompraGov("ide.gCompraGov", ide.GCompraGov))...)
	e = append(e, deCTe(cte.ValidarPagamentos("ide", ide.TpPagAnt, ide.GPagAntecipado, ide.GMultaJuros))...)
	e = append(e, deCTe(cte.ValidarPgtoVinc("pgtoVinc", c.InfCte.PgtoVinc))...)

	if ide.TpEmis.Contingencia() {
		if ide.DhCont == nil || ide.DhCont.Vazia() {
			e = append(e, erro("ide.dhCont", "emissão em contingência exige a data e hora de entrada"))
		}
		if tamanho := len([]rune(ide.XJust)); tamanho < 15 || tamanho > 256 {
			e = append(e, erro("ide.xJust",
				"justificativa tem %d caracteres; o leiaute aceita de 15 a 256", tamanho))
		}
	} else if ide.XJust != "" {
		e = append(e, erro("ide.xJust", "justificativa só é permitida na emissão em contingência"))
	}
	return e
}

func (c *CTeOS) validarEmitente() Erros {
	var e Erros
	emit := &c.InfCte.Emit

	switch {
	case emit.CNPJ != "" && emit.CPF != "":
		e = append(e, erro("emit", "informe CNPJ ou CPF, nunca os dois"))
	case emit.CNPJ != "":
		if err := validacao.ValidarCNPJ(emit.CNPJ); err != nil {
			e = append(e, erro("emit.CNPJ", "%v", err))
		}
	case emit.CPF != "":
		if err := validacao.ValidarCPF(emit.CPF); err != nil {
			e = append(e, erro("emit.CPF", "%v", err))
		}
	default:
		e = append(e, erro("emit", "o emitente precisa de CNPJ ou CPF"))
	}

	if tamanho := len([]rune(emit.XNome)); tamanho < 2 || tamanho > 60 {
		e = append(e, erro("emit.xNome", "razão social tem %d caracteres; o leiaute aceita de 2 a 60", tamanho))
	}
	if emit.IE == "" {
		e = append(e, erro("emit.IE", "inscrição estadual é obrigatória"))
	}
	endereco := emit.EnderEmit.Endereco()
	e = append(e, validarEndereco("emit.enderEmit", &endereco)...)
	if emit.CRT != "1" && emit.CRT != "2" && emit.CRT != "3" && emit.CRT != "4" {
		e = append(e, erro("emit.CRT", "regime tributário %q fora do leiaute (TCRT: 1, 2, 3 ou 4)", emit.CRT))
	}
	return e
}

func validarEndereco(campo string, end *cte.Endereco) Erros {
	var e Erros
	if end == nil {
		return append(e, erro(campo, "endereço é obrigatório"))
	}
	if end.XLgr == "" {
		e = append(e, erro(campo+".xLgr", "logradouro é obrigatório"))
	}
	if end.Nro == "" {
		e = append(e, erro(campo+".nro", `número é obrigatório; use "S/N" quando não houver`))
	}
	if end.XBairro == "" {
		e = append(e, erro(campo+".xBairro", "bairro é obrigatório"))
	}
	if end.CMun == 0 {
		e = append(e, erro(campo+".cMun", "código do município é obrigatório"))
	}
	if end.XMun == "" {
		e = append(e, erro(campo+".xMun", "município é obrigatório"))
	}
	if _, err := uf.PorSigla(end.UF); err != nil {
		e = append(e, erro(campo+".UF", "%v", err))
	}
	return e
}

// validarTomador confere o único participante do CT-e OS além do emitente.
func (c *CTeOS) validarTomador() Erros {
	var e Erros
	t := c.InfCte.Toma
	if t == nil {
		return append(e, erro("toma", "o tomador do serviço é obrigatório"))
	}

	switch {
	case t.CNPJ != "" && t.CPF != "":
		e = append(e, erro("toma", "informe CNPJ ou CPF, nunca os dois"))
	case t.CNPJ != "":
		if err := validacao.ValidarCNPJ(t.CNPJ); err != nil {
			e = append(e, erro("toma.CNPJ", "%v", err))
		}
	case t.CPF != "":
		if err := validacao.ValidarCPF(t.CPF); err != nil {
			e = append(e, erro("toma.CPF", "%v", err))
		}
	default:
		e = append(e, erro("toma", "informe CNPJ ou CPF do tomador"))
	}

	if t.XNome == "" {
		e = append(e, erro("toma.xNome", "nome do tomador é obrigatório"))
	}
	// Um tomador declarado como contribuinte do ICMS sem inscrição estadual é
	// a contradição que a SEFAZ mais rejeita neste grupo.
	if c.InfCte.Ide.IndIEToma == cte.ContribuinteICMS && t.IE == "" {
		e = append(e, erro("toma.IE",
			"o tomador foi declarado contribuinte do ICMS mas não tem inscrição estadual"))
	}
	if t.EnderToma != nil {
		e = append(e, validarEndereco("toma.enderToma", t.EnderToma)...)
	}
	return e
}

func (c *CTeOS) validarPrestacao() Erros {
	var e Erros
	v := &c.InfCte.VPrest

	if v.VTPrest.EhZero() || v.VTPrest.Negativo() {
		e = append(e, erro("vPrest.vTPrest", "o valor total da prestação precisa ser maior que zero"))
	}
	if v.VRec.Negativo() {
		e = append(e, erro("vPrest.vRec", "o valor a receber não pode ser negativo"))
	}

	if len(v.Comp) > 0 {
		soma := tipos.NovoDecimal(0, 2)
		for i, comp := range v.Comp {
			if comp.XNome == "" {
				e = append(e, erro(fmt.Sprintf("vPrest.Comp[%d].xNome", i), "nome do componente é obrigatório"))
			} else if tamanho := len([]rune(comp.XNome)); tamanho > 15 {
				e = append(e, erro(fmt.Sprintf("vPrest.Comp[%d].xNome", i), "nome do componente tem %d caracteres; o leiaute aceita até 15", tamanho))
			}
			soma = soma.Somar(comp.VComp)
		}
		if soma.Subtrair(v.VTPrest).Abs().Comparar(ToleranciaCentavos) > 0 {
			e = append(e, erro("vPrest.vTPrest",
				"o total informado é %s mas a soma dos componentes dá %s", v.VTPrest, soma))
		}
	}
	return e
}

func (c *CTeOS) validarImposto() Erros {
	var e Erros
	e = append(e, deCTe(cte.ValidarIBSCBS("imp.IBSCBS", c.InfCte.Imp.IBSCBS))...)
	if f := c.InfCte.Imp.InfTribFed; f != nil {
		for _, v := range []struct {
			campo string
			valor *tipos.Decimal
		}{
			{"vPIS", f.VPIS}, {"vCOFINS", f.VCOFINS}, {"vIR", f.VIR}, {"vINSS", f.VINSS}, {"vCSLL", f.VCSLL},
		} {
			if v.valor != nil && v.valor.Negativo() {
				e = append(e, erro("imp.infTribFed."+v.campo, "o valor não pode ser negativo"))
			}
		}
	}
	if !c.InfCte.Imp.ICMS.Preenchido() {
		return append(e, erro("imp.ICMS", "preencha exatamente uma variação do grupo ICMS"))
	}

	i := &c.InfCte.Imp.ICMS
	preenchidas := 0
	for _, ok := range []bool{
		i.ICMS00 != nil, i.ICMS20 != nil, i.ICMS45 != nil, i.ICMS60 != nil,
		i.ICMS90 != nil, i.ICMSOutraUF != nil, i.ICMSSN != nil,
	} {
		if ok {
			preenchidas++
		}
	}
	if preenchidas > 1 {
		e = append(e, erro("imp.ICMS", "preencha apenas uma variação do grupo ICMS; foram %d", preenchidas))
	}
	return e
}

func (c *CTeOS) validarNorm() Erros {
	var e Erros

	if c.InfCte.Ide.TpCTe == cte.CTeComplemento {
		if c.InfCte.InfCteComp == nil {
			e = append(e, erro("infCteComp", "o CT-e OS complementar precisa apontar o conhecimento complementado"))
		}
		return e
	}

	n := c.InfCte.InfCTeNorm
	if n == nil {
		return append(e, erro("infCTeNorm", "o CT-e OS normal precisa do grupo infCTeNorm"))
	}

	if n.InfServico.XDescServ == "" {
		e = append(e, erro("infCTeNorm.infServico.xDescServ", "a descrição do serviço é obrigatória"))
	} else if tamanho := len([]rune(n.InfServico.XDescServ)); tamanho > 30 {
		e = append(e, erro("infCTeNorm.infServico.xDescServ", "a descrição do serviço tem %d caracteres; o leiaute aceita até 30", tamanho))
	}
	if q := n.InfServico.InfQ; q != nil && (q.QCarga.EhZero() || q.QCarga.Negativo()) {
		e = append(e, erro("infCTeNorm.infServico.infQ.qCarga",
			"a quantidade precisa ser maior que zero quando informada"))
	}

	e = append(e, c.validarModal(&n.InfModal)...)

	for i, doc := range n.InfDocRef {
		campo := fmt.Sprintf("infCTeNorm.infDocRef[%d]", i)
		if doc.ChBPe != "" {
			if err := chave.Validar(doc.ChBPe); err != nil {
				e = append(e, erro(campo+".chBPe", "%v", err))
			}
			continue
		}
		if doc.NDoc == "" {
			e = append(e, erro(campo+".nDoc",
				"informe o número do documento ou a chave de um BP-e"))
		}
	}

	for i, seg := range n.Seg {
		switch seg.RespSeg {
		case SeguroTomador, SeguroEmitente:
		default:
			e = append(e, erro(fmt.Sprintf("infCTeNorm.seg[%d].respSeg", i),
				"responsável %q; use 4 (tomador) ou 5 (emitente)", seg.RespSeg))
		}
	}

	for i, gtv := range n.InfGTVe {
		if err := chave.Validar(gtv.ChCTe); err != nil {
			e = append(e, erro(fmt.Sprintf("infCTeNorm.infGTVe[%d].chCTe", i), "%v", err))
		}
	}
	return e
}

// validarModal confere o grupo rodoOS, o único modal do CT-e OS.
func (c *CTeOS) validarModal(m *InfModal) Erros {
	var e Erros
	if m.VersaoModal == "" {
		e = append(e, erro("infCTeNorm.infModal@versaoModal", "a versão do modal é obrigatória"))
	}
	if m.RodoOS == nil {
		return append(e, erro("infCTeNorm.infModal.rodoOS", "o grupo do modal rodoviário é obrigatório"))
	}

	rodo := m.RodoOS
	if rodo.TAF == "" && rodo.NroRegEstadual == "" {
		e = append(e, erro("infCTeNorm.infModal.rodoOS",
			"informe o TAF ou o número do registro estadual do transportador"))
	}

	if v := rodo.Veic; v != nil {
		if v.Placa == "" {
			e = append(e, erro("infCTeNorm.infModal.rodoOS.veic.placa", "a placa é obrigatória"))
		}
		if v.UF != "" {
			if _, err := uf.PorSigla(v.UF); err != nil {
				e = append(e, erro("infCTeNorm.infModal.rodoOS.veic.UF", "%v", err))
			}
		}
		if p := v.Prop; p != nil {
			if p.TpProp != ProprietarioTACAgregado && p.TpProp != ProprietarioTACIndependente && p.TpProp != ProprietarioOutros {
				e = append(e, erro("infCTeNorm.infModal.rodoOS.veic.prop.tpProp",
					"tipo do proprietário %q fora do leiaute (0 TAC agregado, 1 TAC independente, 2 outros)", p.TpProp))
			}
			switch {
			case p.CNPJ != "" && p.CPF != "":
				e = append(e, erro("infCTeNorm.infModal.rodoOS.veic.prop",
					"informe CNPJ ou CPF, nunca os dois"))
			case p.CNPJ != "":
				if err := validacao.ValidarCNPJ(p.CNPJ); err != nil {
					e = append(e, erro("infCTeNorm.infModal.rodoOS.veic.prop.CNPJ", "%v", err))
				}
			case p.CPF != "":
				if err := validacao.ValidarCPF(p.CPF); err != nil {
					e = append(e, erro("infCTeNorm.infModal.rodoOS.veic.prop.CPF", "%v", err))
				}
			default:
				e = append(e, erro("infCTeNorm.infModal.rodoOS.veic.prop",
					"o proprietário precisa de CNPJ ou CPF"))
			}
			if p.XNome == "" {
				e = append(e, erro("infCTeNorm.infModal.rodoOS.veic.prop.xNome",
					"o nome do proprietário é obrigatório"))
			}
		}
	}

	// O fretamento é do transporte de pessoas; no eventual, a SEFAZ exige a
	// data e hora da viagem.
	if f := rodo.InfFretamento; f != nil {
		if c.InfCte.Ide.TpServ != ServicoTransportePessoas {
			e = append(e, erro("infCTeNorm.infModal.rodoOS.infFretamento",
				"o grupo de fretamento só cabe ao transporte de pessoas"))
		}
		switch f.TpFretamento {
		case FretamentoEventual:
			if f.DhViagem == nil || f.DhViagem.Vazia() {
				e = append(e, erro("infCTeNorm.infModal.rodoOS.infFretamento.dhViagem",
					"o fretamento eventual exige a data e hora da viagem"))
			}
		case FretamentoContinuo:
		default:
			e = append(e, erro("infCTeNorm.infModal.rodoOS.infFretamento.tpFretamento",
				"tipo %q; use 1 (eventual) ou 2 (contínuo)", f.TpFretamento))
		}
	}
	return e
}

// deCTe traz para este pacote os erros das conferências compartilhadas do pacote cte.
func deCTe(erros cte.Erros) Erros {
	out := make(Erros, 0, len(erros))
	for _, e := range erros {
		out = append(out, Erro{Campo: e.Campo, Mensagem: e.Mensagem})
	}
	return out
}

// validarForaDoLeiaute recusa o que os tipos compartilhados com o CT-e admitem e o leiaute do CT-e OS não tem: a
// estrutura é a mesma, mas o esquema do OS não prevê esses campos, e o documento seria recusado.
func (c *CTeOS) validarForaDoLeiaute() Erros {
	var e Erros
	if compl := c.InfCte.Compl; compl != nil {
		for _, campo := range []struct {
			nome       string
			preenchido bool
		}{
			{"fluxo", compl.Fluxo != nil},
			{"Entrega", compl.Entrega != nil},
			{"origCalc", compl.OrigCalc != ""},
			{"destCalc", compl.DestCalc != ""},
		} {
			if campo.preenchido {
				e = append(e, erro("compl."+campo.nome, "o CT-e OS não tem o %s", campo.nome))
			}
		}
	}
	if c.InfCte.Emit.CPF != "" {
		e = append(e, erro("emit.CPF", "o emitente do CT-e OS é identificado pelo CNPJ"))
	}
	if c.InfCte.Imp.ICMS.ICMS60 != nil {
		e = append(e, erro("imp.ICMS.ICMS60", "o CT-e OS não tem o ICMS60"))
	}
	if norm := c.InfCte.InfCTeNorm; norm != nil && norm.InfCteSub != nil {
		sub := norm.InfCteSub
		if sub.IndAlteraToma != "" {
			e = append(e, erro("infCteSub.indAlteraToma", "o CT-e OS não tem o indAlteraToma"))
		}
		if sub.RefCteAnu != "" || sub.TomaICMS != nil { //nolint:staticcheck // Recusa os campos que só existem por compatibilidade.
			e = append(e, erro("infCteSub", "o leiaute 4.00 não tem o refCteAnu nem o tomaICMS"))
		}
	}
	return e
}
