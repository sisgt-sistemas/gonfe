package ctesimp

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/tipos"
	"github.com/sisgt-sistemas/gonfe/uf"
	"github.com/sisgt-sistemas/gonfe/validacao"
)

// Erro é uma inconsistência encontrada por [CTeSimp.Validar].
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
		return "ctesimp: nenhum erro"
	case 1:
		return "ctesimp: " + e[0].Error()
	}
	var b strings.Builder
	fmt.Fprintf(&b, "ctesimp: %d inconsistências:", len(e))
	for _, item := range e {
		b.WriteString("\n  - ")
		b.WriteString(item.Error())
	}
	return b.String()
}

// ToleranciaCentavos é a diferença máxima aceita nas conferências de somatórios.
var ToleranciaCentavos = tipos.D("0.01")

// MaxItens é o número máximo de prestações (det) de um documento, pelo esquema.
const MaxItens = 999

// MaxAutXML e MaxObservacoes são os maxOccurs de autXML, ObsCont e ObsFisco no esquema.
const (
	MaxAutXML      = 10
	MaxObservacoes = 10
)

// Os padrões do esquema que o Validar confere.
var (
	padraoCFOP   = regexp.MustCompile(`^[123567][0-9]([0-9][1-9]|[1-9][0-9])$`) // TCfop
	padraoPIN    = regexp.MustCompile(`^[1-9][0-9]{1,8}$`)
	padraoCEP    = regexp.MustCompile(`^[0-9]{8}$`)
	padraoFone   = regexp.MustCompile(`^[0-9]{6,14}$`) // TFone
	padraoISUF   = regexp.MustCompile(`^[0-9]{8,9}$`)
	padraoEmail  = regexp.MustCompile(`^[^@]+@[^.]+\..+$`)      // TEmail
	padraoIE     = regexp.MustCompile(`^[0-9]{2,14}$`)          // TIe, do emitente
	padraoIEToma = regexp.MustCompile(`^([0-9]{0,14}|ISENTO)$`) // TIeDest, do tomador
)

// Validar confere o conhecimento antes do envio. A maior parte das regras é do
// esquema (TCTeSimp): presença, faixa, padrão e tamanho dos campos, ordem dos
// itens e dígitos verificadores das chaves. Algumas são da SEFAZ, que o esquema
// não expressa e a autorização recusa: valor da prestação maior que zero,
// data e justificativa obrigatórias na contingência, inscrição estadual do
// tomador contribuinte, totais iguais à soma dos itens. Não substitui as demais
// regras de negócio que a SEFAZ aplica na autorização.
func (c *CTeSimp) Validar() error {
	var e Erros
	e = append(e, c.validarIde()...)
	e = append(e, c.validarCompl()...)
	e = append(e, c.validarEmitente()...)
	e = append(e, c.validarTomador()...)
	e = append(e, c.validarCarga()...)
	e = append(e, c.validarItens()...)
	e = append(e, c.validarModal()...)
	e = append(e, c.validarImposto()...)
	e = append(e, c.validarTotal()...)
	e = append(e, c.validarSubstituicao()...)
	if len(c.InfCte.AutXML) > MaxAutXML {
		e = append(e, erro("autXML", "%d autorizados; o leiaute aceita até %d", len(c.InfCte.AutXML), MaxAutXML))
	}
	for i, aut := range c.InfCte.AutXML {
		e = append(e, validarDocumento(fmt.Sprintf("autXML[%d]", i), aut.CNPJ, aut.CPF)...)
	}
	if len(e) == 0 {
		return nil
	}
	return e
}

func erro(campo, formato string, args ...any) Erro {
	return Erro{Campo: campo, Mensagem: fmt.Sprintf(formato, args...)}
}

// tamanho confere um texto contra o minLength e o maxLength do esquema; vazio só passa se o campo for opcional.
func tamanho(campo, valor string, minimo, maximo int, opcional bool) Erros {
	n := len([]rune(valor))
	if n == 0 && opcional {
		return nil
	}
	if n < minimo || n > maximo {
		return Erros{erro(campo, "tem %d caracteres; o leiaute aceita de %d a %d", n, minimo, maximo)}
	}
	return nil
}

func (c *CTeSimp) validarIde() Erros {
	var e Erros
	ide := &c.InfCte.Ide

	if c.InfCte.Versao != Versao {
		e = append(e, erro("infCte@versao", "versão %q; este pacote implementa a %s", c.InfCte.Versao, Versao))
	}
	if ide.Mod != cte.ModeloCTe {
		e = append(e, erro("ide.mod", "modelo %q; o CT-e Simplificado é o modelo 57", ide.Mod))
	}
	if ide.TpCTe != Simplificado && ide.TpCTe != SubstituicaoSimplificado {
		e = append(e, erro("ide.tpCTe", "finalidade %q; o CT-e Simplificado aceita 5 (simplificado) ou 6 (substituição)", ide.TpCTe))
	}
	e = append(e, tamanho("ide.natOp", ide.NatOp, 1, 60, false)...)
	if !padraoCFOP.MatchString(ide.CFOP) {
		e = append(e, erro("ide.CFOP", "CFOP %q fora do padrão do leiaute (TCfop: quatro dígitos, começando por 1, 2, 3, 5, 6 ou 7)", ide.CFOP))
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
	if ide.TpImp != cte.Retrato && ide.TpImp != cte.Paisagem {
		e = append(e, erro("ide.tpImp", "formato %q; use 1 (retrato) ou 2 (paisagem)", ide.TpImp))
	}
	switch ide.TpEmis {
	case cte.EmissaoNormal, cte.EmissaoContingenciaOffline, "3", cte.EmissaoEPEC, cte.EmissaoSVCRS, cte.EmissaoSVCSP:
	default:
		e = append(e, erro("ide.tpEmis", "forma de emissão %q fora do leiaute (1, 2, 3, 4, 7 ou 8)", ide.TpEmis))
	}
	if ide.TpAmb != cte.Producao && ide.TpAmb != cte.Homologacao {
		e = append(e, erro("ide.tpAmb", "ambiente %q; use 1 (produção) ou 2 (homologação)", ide.TpAmb))
	}
	switch ide.ProcEmi {
	case cte.EmissaoAplicativoContribuinte, cte.EmissaoAplicativoFisco, "4":
	default:
		e = append(e, erro("ide.procEmi", "processo de emissão %q fora do leiaute (0, 3 ou 4)", ide.ProcEmi))
	}
	e = append(e, tamanho("ide.verProc", ide.VerProc, 1, 20, false)...)
	if ide.Modal != cte.ModalRodoviario {
		e = append(e, erro("ide.modal", "modal %q; este pacote implementa o rodoviário (01)", ide.Modal))
	}
	switch ide.TpServ {
	case cte.ServicoNormal, cte.ServicoSubcontratacao, cte.ServicoRedespacho:
	default:
		e = append(e, erro("ide.tpServ", "tipo de serviço %q; o CT-e Simplificado aceita 0 (normal), 1 (subcontratação) ou 2 (redespacho)", ide.TpServ))
	}
	e = append(e, validarMunicipio("ide.cMunEnv", ide.CMunEnv, ide.XMunEnv)...)
	for _, u := range []struct{ campo, sigla string }{
		{"ide.UFEnv", ide.UFEnv}, {"ide.UFIni", ide.UFIni}, {"ide.UFFim", ide.UFFim},
	} {
		e = append(e, validarUF(u.campo, u.sigla, true)...)
	}
	if ide.Retira != cte.RetiraSim && ide.Retira != cte.RetiraNao {
		e = append(e, erro("ide.retira", "indicador %q; use 0 (sim) ou 1 (não)", ide.Retira))
	}
	e = append(e, tamanho("ide.xDetRetira", ide.XDetRetira, 1, 160, true)...)

	if ide.TpEmis.Contingencia() {
		if ide.DhCont == nil || ide.DhCont.Vazia() {
			e = append(e, erro("ide.dhCont", "emissão em contingência exige a data e hora de entrada"))
		}
		e = append(e, tamanho("ide.xJust", ide.XJust, 15, 256, false)...)
	} else if ide.XJust != "" || ide.DhCont != nil {
		e = append(e, erro("ide.xJust", "data e justificativa de contingência só cabem na emissão em contingência"))
	}
	return e
}

// validarUF confere a sigla. O tipo TUf aceita também EX (exterior); o TUF_sem_EX, do emitente, não.
func validarUF(campo, sigla string, aceitaExterior bool) Erros {
	if aceitaExterior && sigla == "EX" {
		return nil
	}
	if _, err := uf.PorSigla(sigla); err != nil {
		return Erros{erro(campo, "%v", err)}
	}
	return nil
}

// validarCompl confere tamanhos e repetições das informações complementares.
func (c *CTeSimp) validarCompl() Erros {
	compl := c.InfCte.Compl
	if compl == nil {
		return nil
	}
	var e Erros
	e = append(e, tamanho("compl.xCaracAd", compl.XCaracAd, 1, 15, true)...)
	e = append(e, tamanho("compl.xCaracSer", compl.XCaracSer, 1, 30, true)...)
	e = append(e, tamanho("compl.xObs", compl.XObs, 1, 2000, true)...)
	if f := compl.Fluxo; f != nil {
		e = append(e, tamanho("compl.fluxo.xOrig", f.XOrig, 1, 60, true)...)
		for i, p := range f.Pass {
			e = append(e, tamanho(fmt.Sprintf("compl.fluxo.pass[%d].xPass", i), p.XPass, 1, 15, true)...)
		}
		e = append(e, tamanho("compl.fluxo.xDest", f.XDest, 1, 60, true)...)
		e = append(e, tamanho("compl.fluxo.xRota", f.XRota, 1, 10, true)...)
	}
	for _, grupo := range []struct {
		campo  string
		obs    []cte.ObsCampo
		maximo int
	}{
		{"compl.ObsCont", compl.ObsCont, 160},
		{"compl.ObsFisco", compl.ObsFisco, 60},
	} {
		if len(grupo.obs) > MaxObservacoes {
			e = append(e, erro(grupo.campo, "%d observações; o leiaute aceita até %d", len(grupo.obs), MaxObservacoes))
		}
		for i, o := range grupo.obs {
			e = append(e, tamanho(fmt.Sprintf("%s[%d]@xCampo", grupo.campo, i), o.XCampo, 1, 20, false)...)
			e = append(e, tamanho(fmt.Sprintf("%s[%d].xTexto", grupo.campo, i), o.XTexto, 1, grupo.maximo, false)...)
		}
	}
	return e
}

// padrao confere um campo opcional contra o padrão do esquema.
func padrao(campo, valor string, re *regexp.Regexp, descricao string) Erros {
	if valor == "" || re.MatchString(valor) {
		return nil
	}
	return Erros{erro(campo, "%q fora do padrão do leiaute (%s)", valor, descricao)}
}

func validarMunicipio(campo string, codigo int, nome string) Erros {
	var e Erros
	if codigo < 1000000 || codigo > 9999999 {
		e = append(e, erro(campo, "código de município %d não tem 7 dígitos", codigo))
	}
	e = append(e, tamanho(strings.Replace(campo, ".c", ".x", 1), nome, 2, 60, false)...)
	return e
}

func validarDocumento(campo, cnpj, cpf string) Erros {
	switch {
	case cnpj != "" && cpf != "":
		return Erros{erro(campo, "informe CNPJ ou CPF, nunca os dois")}
	case cnpj != "":
		if err := validacao.ValidarCNPJ(cnpj); err != nil {
			return Erros{erro(campo+".CNPJ", "%v", err)}
		}
	case cpf != "":
		if err := validacao.ValidarCPF(cpf); err != nil {
			return Erros{erro(campo+".CPF", "%v", err)}
		}
	default:
		return Erros{erro(campo, "informe CNPJ ou CPF")}
	}
	return nil
}

// validarEndereco confere o endereço. maxLgr é o tamanho do logradouro (60 no TEndeEmi, 255 no TEndereco), e
// aceitaExterior diz se a UF pode ser EX.
func validarEndereco(campo string, end cte.Endereco, maxLgr int, aceitaExterior bool) Erros {
	var e Erros
	e = append(e, tamanho(campo+".xLgr", end.XLgr, 2, maxLgr, false)...)
	if end.Nro == "" {
		e = append(e, erro(campo+".nro", `número é obrigatório; use "S/N" quando não houver`))
	} else {
		e = append(e, tamanho(campo+".nro", end.Nro, 1, 60, false)...)
	}
	e = append(e, tamanho(campo+".xCpl", end.XCpl, 1, 60, true)...)
	e = append(e, tamanho(campo+".xBairro", end.XBairro, 2, 60, false)...)
	e = append(e, validarMunicipio(campo+".cMun", end.CMun, end.XMun)...)
	e = append(e, padrao(campo+".CEP", end.CEP, padraoCEP, "oito dígitos")...)
	e = append(e, validarUF(campo+".UF", end.UF, aceitaExterior)...)
	return e
}

func (c *CTeSimp) validarEmitente() Erros {
	emit := &c.InfCte.Emit
	e := validarDocumento("emit", emit.CNPJ, emit.CPF)
	e = append(e, tamanho("emit.xNome", emit.XNome, 2, 60, false)...)
	e = append(e, tamanho("emit.xFant", emit.XFant, 2, 60, true)...)
	e = append(e, padrao("emit.IE", emit.IE, padraoIE, "TIe: de 2 a 14 dígitos")...)
	e = append(e, validarEndereco("emit.enderEmit", emit.EnderEmit.Endereco(), 60, false)...)
	e = append(e, padrao("emit.enderEmit.fone", emit.EnderEmit.Fone, padraoFone, "TFone: de 6 a 14 dígitos")...)
	if emit.CRT != "1" && emit.CRT != "2" && emit.CRT != "3" && emit.CRT != "4" {
		e = append(e, erro("emit.CRT", "regime tributário %q fora do leiaute (TCRT: 1, 2, 3 ou 4)", emit.CRT))
	}
	e = append(e, padrao("emit.ISUFEmit", emit.ISUFEmit, padraoISUF, "8 ou 9 dígitos")...)
	return e
}

func (c *CTeSimp) validarTomador() Erros {
	var e Erros
	t := &c.InfCte.Toma
	switch t.Toma {
	case cte.TomadorRemetente, cte.TomadorExpedidor, cte.TomadorRecebedor, cte.TomadorDestinatario, cte.TomadorOutros:
	default:
		e = append(e, erro("toma.toma", "papel do tomador %q; use de 0 (remetente) a 4 (outros)", t.Toma))
	}
	switch t.IndIEToma {
	case cte.ContribuinteICMS, cte.IsentoIE, cte.NaoContribuinte:
	default:
		e = append(e, erro("toma.indIEToma", "indicador %q; use 1, 2 ou 9", t.IndIEToma))
	}
	e = append(e, validarDocumento("toma", t.CNPJ, t.CPF)...)
	// Tomador contribuinte do ICMS sem inscrição estadual é a contradição que a SEFAZ mais rejeita neste grupo.
	if t.IndIEToma == cte.ContribuinteICMS && t.IE == "" {
		e = append(e, erro("toma.IE", "o tomador foi declarado contribuinte do ICMS mas não tem inscrição estadual"))
	}
	e = append(e, padrao("toma.IE", t.IE, padraoIEToma, "TIeDest: até 14 dígitos ou ISENTO")...)
	e = append(e, tamanho("toma.xNome", t.XNome, 2, 60, false)...)
	e = append(e, padrao("toma.ISUF", t.ISUF, padraoISUF, "8 ou 9 dígitos")...)
	e = append(e, padrao("toma.fone", t.Fone, padraoFone, "TFone: de 6 a 14 dígitos")...)
	e = append(e, validarEndereco("toma.enderToma", t.EnderToma, 255, true)...)
	if t.Email != "" && (len([]rune(t.Email)) > 60 || !padraoEmail.MatchString(t.Email)) {
		e = append(e, erro("toma.email", "e-mail %q fora do padrão do leiaute (TEmail, até 60)", t.Email))
	}
	return e
}

func (c *CTeSimp) validarCarga() Erros {
	var e Erros
	carga := &c.InfCte.InfCarga
	if carga.VCarga == nil {
		e = append(e, erro("infCarga.vCarga", "o valor da carga é obrigatório no CT-e Simplificado"))
	} else if carga.VCarga.Negativo() {
		e = append(e, erro("infCarga.vCarga", "o valor da carga não pode ser negativo"))
	}
	e = append(e, tamanho("infCarga.proPred", carga.ProPred, 1, 60, false)...)
	e = append(e, tamanho("infCarga.xOutCat", carga.XOutCat, 1, 30, true)...)
	if len(carga.InfQ) == 0 {
		e = append(e, erro("infCarga.infQ", "informe ao menos uma quantidade da carga"))
	}
	for i, q := range carga.InfQ {
		campo := fmt.Sprintf("infCarga.infQ[%d]", i)
		switch q.CUnid {
		case cte.UnidadeM3, cte.UnidadeKG, cte.UnidadeTON, cte.UnidadeUnidade, cte.UnidadeLitros, cte.UnidadeMMBTU:
		default:
			e = append(e, erro(campo+".cUnid", "unidade %q fora do leiaute (00 a 05)", q.CUnid))
		}
		if !tipoDeMedidaValido(q.TpMed) {
			e = append(e, erro(campo+".tpMed", "tipo de medida %q fora do leiaute (00 a 15 ou 99)", q.TpMed))
		}
		if q.QCarga.Negativo() {
			e = append(e, erro(campo+".qCarga", "a quantidade não pode ser negativa"))
		}
	}
	return e
}

// tipoDeMedidaValido confere o tpMed do CT-e Simplificado: 00 a 15 e 99.
func tipoDeMedidaValido(tpMed string) bool {
	if tpMed == "99" {
		return true
	}
	if len(tpMed) != 2 || tpMed[0] < '0' || tpMed[0] > '1' || tpMed[1] < '0' || tpMed[1] > '9' {
		return false
	}
	return tpMed <= "15"
}

func (c *CTeSimp) validarItens() Erros {
	var e Erros
	itens := c.InfCte.Det
	if len(itens) == 0 || len(itens) > MaxItens {
		return append(e, erro("det", "o documento tem %d prestações; o leiaute aceita de 1 a %d", len(itens), MaxItens))
	}
	for i := range itens {
		det := &itens[i]
		campo := fmt.Sprintf("det[%d]", i)
		if det.NItem != i+1 {
			e = append(e, erro(campo+"@nItem", "item %d; os itens são numerados de 1 em diante, na ordem", det.NItem))
		}
		e = append(e, validarMunicipio(campo+".cMunIni", det.CMunIni, det.XMunIni)...)
		e = append(e, validarMunicipio(campo+".cMunFim", det.CMunFim, det.XMunFim)...)
		if det.VPrest.EhZero() || det.VPrest.Negativo() {
			e = append(e, erro(campo+".vPrest", "o valor da prestação precisa ser maior que zero"))
		}
		if det.VRec.Negativo() {
			e = append(e, erro(campo+".vRec", "o valor a receber não pode ser negativo"))
		}
		if len(det.Comp) > 0 {
			soma := tipos.NovoDecimal(0, 2)
			for j, comp := range det.Comp {
				e = append(e, tamanho(fmt.Sprintf("%s.Comp[%d].xNome", campo, j), comp.XNome, 1, 15, false)...)
				soma = soma.Somar(comp.VComp)
			}
			if soma.Subtrair(det.VPrest).Abs().Comparar(ToleranciaCentavos) > 0 {
				e = append(e, erro(campo+".vPrest", "o valor informado é %s mas a soma dos componentes dá %s", det.VPrest, soma))
			}
		}
		e = append(e, validarDocumentos(campo, det)...)
	}
	return e
}

// validarDocumentos confere o que a prestação transporta: NF-e ou CT-e anteriores, um dos dois.
func validarDocumentos(campo string, det *Det) Erros {
	var e Erros
	switch {
	case len(det.InfNFe) > 0 && len(det.InfDocAnt) > 0:
		return append(e, erro(campo, "informe as NF-e (infNFe) ou os documentos anteriores (infDocAnt), nunca os dois"))
	case len(det.InfNFe) == 0 && len(det.InfDocAnt) == 0:
		return append(e, erro(campo, "a prestação precisa das NF-e transportadas (infNFe) ou dos documentos anteriores (infDocAnt)"))
	}
	for i, nfe := range det.InfNFe {
		sub := fmt.Sprintf("%s.infNFe[%d]", campo, i)
		if err := chave.Validar(nfe.ChNFe); err != nil {
			e = append(e, erro(sub+".chNFe", "%v", err))
		}
		e = append(e, padrao(sub+".PIN", nfe.PIN, padraoPIN, "de 2 a 9 dígitos, sem zero à esquerda")...)
		if len(nfe.InfUnidCarga) > 0 && len(nfe.InfUnidTransp) > 0 {
			e = append(e, erro(sub, "informe as unidades de carga ou as de transporte, nunca as duas"))
		}
		for j, u := range nfe.InfUnidCarga {
			e = append(e, validarUnidadeCarga(fmt.Sprintf("%s.infUnidCarga[%d]", sub, j), u)...)
		}
		for j, u := range nfe.InfUnidTransp {
			e = append(e, validarUnidadeTransporte(fmt.Sprintf("%s.infUnidTransp[%d]", sub, j), u)...)
		}
	}
	for i, ant := range det.InfDocAnt {
		sub := fmt.Sprintf("%s.infDocAnt[%d]", campo, i)
		if err := chave.Validar(ant.ChCTe); err != nil {
			e = append(e, erro(sub+".chCTe", "%v", err))
		}
		if ant.TpPrest != PrestacaoTotal && ant.TpPrest != PrestacaoParcial {
			e = append(e, erro(sub+".tpPrest", "tipo %q; use 1 (total) ou 2 (parcial)", ant.TpPrest))
		}
		for j, parcial := range ant.InfNFeTranspParcial {
			if err := chave.Validar(parcial.ChNFe); err != nil {
				e = append(e, erro(fmt.Sprintf("%s.infNFeTranspParcial[%d].chNFe", sub, j), "%v", err))
			}
		}
	}
	return e
}

// identificacaoValida confere o TContainer: letras maiúsculas e dígitos, de 1 a 20.
func identificacaoValida(id string) bool {
	if id == "" || len(id) > 20 {
		return false
	}
	for _, r := range id {
		if (r < 'A' || r > 'Z') && (r < '0' || r > '9') {
			return false
		}
	}
	return true
}

func validarLacres(campo string, lacres []Lacre) Erros {
	var e Erros
	for i, l := range lacres {
		e = append(e, tamanho(fmt.Sprintf("%s[%d].nLacre", campo, i), l.NLacre, 1, 20, false)...)
	}
	return e
}

func validarRateio(campo string, qtd *tipos.Decimal) Erros {
	// TDec_0302_0303: até três dígitos inteiros.
	if qtd != nil && (qtd.Negativo() || qtd.Comparar(tipos.D("999.99")) > 0) {
		return Erros{erro(campo, "quantidade rateada %s fora da faixa 0–999,99", qtd)}
	}
	return nil
}

func validarUnidadeCarga(campo string, u UnidadeCarga) Erros {
	var e Erros
	if u.TpUnidCarga < "1" || u.TpUnidCarga > "4" || len(u.TpUnidCarga) != 1 {
		e = append(e, erro(campo+".tpUnidCarga", "tipo %q fora do leiaute (1 a 4)", u.TpUnidCarga))
	}
	if !identificacaoValida(u.IdUnidCarga) {
		e = append(e, erro(campo+".idUnidCarga", "identificação %q; use de 1 a 20 letras maiúsculas e dígitos", u.IdUnidCarga))
	}
	e = append(e, validarLacres(campo+".lacUnidCarga", u.LacUnidCarga)...)
	e = append(e, validarRateio(campo+".qtdRat", u.QtdRat)...)
	return e
}

func validarUnidadeTransporte(campo string, u UnidadeTransporte) Erros {
	var e Erros
	if u.TpUnidTransp < "1" || u.TpUnidTransp > "7" || len(u.TpUnidTransp) != 1 {
		e = append(e, erro(campo+".tpUnidTransp", "tipo %q fora do leiaute (1 a 7)", u.TpUnidTransp))
	}
	if !identificacaoValida(u.IdUnidTransp) {
		e = append(e, erro(campo+".idUnidTransp", "identificação %q; use de 1 a 20 letras maiúsculas e dígitos", u.IdUnidTransp))
	}
	e = append(e, validarLacres(campo+".lacUnidTransp", u.LacUnidTransp)...)
	for i, carga := range u.InfUnidCarga {
		e = append(e, validarUnidadeCarga(fmt.Sprintf("%s.infUnidCarga[%d]", campo, i), carga)...)
	}
	e = append(e, validarRateio(campo+".qtdRat", u.QtdRat)...)
	return e
}

// validarModal confere o grupo rodo, o único modal implementado aqui.
func (c *CTeSimp) validarModal() Erros {
	var e Erros
	m := &c.InfCte.InfModal
	if m.VersaoModal == "" {
		e = append(e, erro("infModal@versaoModal", "a versão do modal é obrigatória"))
	}
	if m.Rodo == nil {
		return append(e, erro("infModal.rodo", "o grupo do modal rodoviário é obrigatório"))
	}
	if m.Aereo != nil || m.Aquav != nil || m.Ferrov != nil || m.Duto != nil || m.Multimodal != nil {
		e = append(e, erro("infModal", "informe só o grupo do modal rodoviário"))
	}
	if m.Rodo.RNTRC == "" {
		e = append(e, erro("infModal.rodo.RNTRC", `o RNTRC é obrigatório; use "ISENTO" quando couber`))
	}
	return e
}

func (c *CTeSimp) validarImposto() Erros {
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
	if preenchidas != 1 {
		return Erros{erro("imp.ICMS", "preencha exatamente uma variação do grupo ICMS; foram %d", preenchidas)}
	}
	return nil
}

// validarTotal confere o grupo total contra a soma das prestações.
func (c *CTeSimp) validarTotal() Erros {
	var e Erros
	prest := tipos.NovoDecimal(0, 2)
	rec := tipos.NovoDecimal(0, 2)
	for _, det := range c.InfCte.Det {
		prest = prest.Somar(det.VPrest)
		rec = rec.Somar(det.VRec)
	}
	t := &c.InfCte.Total
	if t.VTPrest.Subtrair(prest).Abs().Comparar(ToleranciaCentavos) > 0 {
		e = append(e, erro("total.vTPrest", "o total informado é %s mas a soma das prestações dá %s", t.VTPrest, prest))
	}
	if t.VTRec.Subtrair(rec).Abs().Comparar(ToleranciaCentavos) > 0 {
		e = append(e, erro("total.vTRec", "o total a receber informado é %s mas a soma das prestações dá %s", t.VTRec, rec))
	}
	return e
}

// validarSubstituicao confere o grupo infCteSub, que existe só na finalidade 6.
func (c *CTeSimp) validarSubstituicao() Erros {
	sub := c.InfCte.InfCteSub
	switch {
	case c.InfCte.Ide.TpCTe == SubstituicaoSimplificado && sub == nil:
		return Erros{erro("infCteSub", "o CT-e de substituição precisa apontar o conhecimento substituído")}
	case c.InfCte.Ide.TpCTe != SubstituicaoSimplificado && sub != nil:
		return Erros{erro("infCteSub", "o grupo de substituição só cabe na finalidade 6")}
	case sub == nil:
		return nil
	}
	var e Erros
	if err := chave.Validar(sub.ChCte); err != nil {
		e = append(e, erro("infCteSub.chCte", "%v", err))
	}
	if sub.IndAlteraToma != "" && sub.IndAlteraToma != "1" {
		e = append(e, erro("infCteSub.indAlteraToma", `indicador %q; o leiaute só aceita "1"`, sub.IndAlteraToma))
	}
	return e
}
