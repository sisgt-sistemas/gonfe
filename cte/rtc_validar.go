package cte

import (
	"encoding/base64"
	"fmt"
	"regexp"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/tipos"
	"github.com/sisgt-sistemas/gonfe/validacao"
)

// Os limites do esquema para os grupos da reforma tributária.
const (
	// MaxRefDFeAnt é o maxOccurs de refDFeAnt na compra governamental.
	MaxRefDFeAnt = 99
	// MaxPgtoVinc é o maxOccurs de pgto no pgtoVinc.
	MaxPgtoVinc = 99
)

var (
	padraoCST       = regexp.MustCompile(`^[0-9]{3}$`)    // TCST
	padraoClassTrib = regexp.MustCompile(`^[0-9]{6}$`)    // TcClassTrib
	padraoMeioPgto  = regexp.MustCompile(`^[0-9]{2}$`)    // tpMeioPgto
	padraoCNPJBase  = regexp.MustCompile(`^[A-Z0-9]{8}$`) // TCnpjBaseRTC
	toleranciaRTC   = tipos.D("0.01")
	// maxAliquotaRTC é o maior valor do TDec_0302_04RTC (três dígitos inteiros).
	maxAliquotaRTC = tipos.D("999.9999")
	// maxValorRTC é o maior valor do TDec1302RTC (treze dígitos inteiros).
	maxValorRTC = tipos.D("9999999999999.99")
)

// ValidarIBSCBS confere o grupo IBSCBS contra o esquema e a coerência dos valores: o vIBS é a soma do IBS da UF e do
// município. Campo é o caminho do grupo no documento, para as mensagens.
func ValidarIBSCBS(campo string, g *IBSCBS) Erros {
	if g == nil {
		return nil
	}
	var e Erros
	if !padraoCST.MatchString(g.CST) {
		e = append(e, erro(campo+".CST", "CST %q; o leiaute pede três dígitos", g.CST))
	}
	if !padraoClassTrib.MatchString(g.CClassTrib) {
		e = append(e, erro(campo+".cClassTrib", "classificação %q; o leiaute pede seis dígitos", g.CClassTrib))
	}
	if g.IndDoacao != "" && g.IndDoacao != "1" {
		e = append(e, erro(campo+".indDoacao", `indicador %q; o leiaute só aceita "1"`, g.IndDoacao))
	}
	e = append(e, faixasIBSCBS(campo, g)...)
	if t := g.GIBSCBS; t != nil {
		soma := t.GIBSUF.VIBSUF.Somar(t.GIBSMun.VIBSMun)
		if soma.Subtrair(t.VIBS).Abs().Comparar(toleranciaRTC) > 0 {
			e = append(e, erro(campo+".gIBSCBS.vIBS", "o vIBS é %s, mas o IBS da UF e do município somam %s", t.VIBS, soma))
		}
		if r := t.GTribRegular; r != nil {
			if !padraoCST.MatchString(r.CSTReg) {
				e = append(e, erro(campo+".gIBSCBS.gTribRegular.CSTReg", "CST %q; o leiaute pede três dígitos", r.CSTReg))
			}
			if !padraoClassTrib.MatchString(r.CClassTribReg) {
				e = append(e, erro(campo+".gIBSCBS.gTribRegular.cClassTribReg", "classificação %q; o leiaute pede seis dígitos", r.CClassTribReg))
			}
		}
	}
	return e
}

// ValidarCompraGov confere a compra governamental (TCompraGovReduzido).
func ValidarCompraGov(campo string, g *CompraGov) Erros {
	if g == nil {
		return nil
	}
	var e Erros
	if len(g.TpEnteGov) != 1 || g.TpEnteGov < "1" || g.TpEnteGov > "6" {
		e = append(e, erro(campo+".tpEnteGov", "tipo do ente %q fora do leiaute (1 a 6)", g.TpEnteGov))
	}
	if len(g.TpOperGov) != 1 || g.TpOperGov < "1" || g.TpOperGov > "4" {
		e = append(e, erro(campo+".tpOperGov", "tipo da operação %q fora do leiaute (1 a 4)", g.TpOperGov))
	}
	e = append(e, faixa(campo+".pRedutor", g.PRedutor, maxAliquotaRTC)...)
	if len(g.RefDFeAnt) > MaxRefDFeAnt {
		e = append(e, erro(campo+".refDFeAnt", "%d documentos; o leiaute aceita até %d", len(g.RefDFeAnt), MaxRefDFeAnt))
	}
	for i, ch := range g.RefDFeAnt {
		if err := chave.Validar(ch); err != nil {
			e = append(e, erro(fmt.Sprintf("%s.refDFeAnt[%d]", campo, i), "%v", err))
		}
	}
	return e
}

// ValidarPagamentos confere o pagamento antecipado (tpPagAnt e gPagAntecipado), a multa e os juros (gMultaJuros), do
// grupo ide.
func ValidarPagamentos(campo, tpPagAnt string, antecipado *PagAntecipado, multa *MultaJuros) Erros {
	var e Erros
	if tpPagAnt != "" && tpPagAnt != "1" && tpPagAnt != "3" {
		e = append(e, erro(campo+".tpPagAnt", "tipo %q fora do leiaute (1 ou 3)", tpPagAnt))
	}
	if antecipado != nil {
		if err := chave.Validar(antecipado.ChDFePagAnt); err != nil {
			e = append(e, erro(campo+".gPagAntecipado.chDFePagAnt", "%v", err))
		}
	}
	if multa != nil {
		if err := chave.Validar(multa.ChCTeRef); err != nil {
			e = append(e, erro(campo+".gMultaJuros.chCTeRef", "%v", err))
		}
	}
	return e
}

// ValidarPgtoVinc confere os pagamentos vinculados (pgtoVinc).
func ValidarPgtoVinc(campo string, p *PgtoVinc) Erros {
	if p == nil {
		return nil
	}
	var e Erros
	if len(p.Pgto) == 0 || len(p.Pgto) > MaxPgtoVinc {
		return append(e, erro(campo+".pgto", "%d pagamentos; o leiaute aceita de 1 a %d", len(p.Pgto), MaxPgtoVinc))
	}
	for i, pg := range p.Pgto {
		sub := fmt.Sprintf("%s.pgto[%d]", campo, i)
		if pg.NPag < 1 || pg.NPag > 990 {
			e = append(e, erro(sub+"@nPag", "número %d fora do leiaute (1 a 990)", pg.NPag))
		}
		if t := len([]rune(pg.IdTransacao)); t < 2 || t > 35 {
			e = append(e, erro(sub+"@idTransacao", "identificação com %d caracteres; o leiaute aceita de 2 a 35", t))
		}
		if !padraoMeioPgto.MatchString(pg.TpMeioPgto) {
			e = append(e, erro(sub+".tpMeioPgto", "meio %q; o leiaute pede dois dígitos", pg.TpMeioPgto))
		}
		if err := validacao.ValidarCNPJ(pg.CNPJReceb); err != nil {
			e = append(e, erro(sub+".CNPJReceb", "%v", err))
		}
		if !padraoCNPJBase.MatchString(pg.CNPJBasePSP) {
			e = append(e, erro(sub+".CNPJBasePSP", "raiz %q; o leiaute pede oito posições", pg.CNPJBasePSP))
		}
	}
	return e
}

// ValidarPAA confere o Provedor de Assinatura e Autorização (infPAA): CNPJ e assinatura em base64.
func ValidarPAA(campo string, p *InfPAA) Erros {
	if p == nil {
		return nil
	}
	var e Erros
	if err := validacao.ValidarCNPJ(p.CNPJPAA); err != nil {
		e = append(e, erro(campo+".CNPJPAA", "%v", err))
	}
	for _, b := range []struct{ nome, valor string }{
		{"PAASignature.SignatureValue", p.PAASignature.SignatureValue},
		{"PAASignature.RSAKeyValue.Modulus", p.PAASignature.RSAKeyValue.Modulus},
		{"PAASignature.RSAKeyValue.Exponent", p.PAASignature.RSAKeyValue.Exponent},
	} {
		if _, err := base64.StdEncoding.DecodeString(b.valor); b.valor == "" || err != nil {
			e = append(e, erro(campo+"."+b.nome, "informe em base64"))
		}
	}
	return e
}

// ValidarNFF confere a solicitação da Nota Fiscal Fácil (xSolic: de 2 a 8000).
func ValidarNFF(campo string, n *InfSolicNFF) Erros {
	if n == nil {
		return nil
	}
	if t := len([]rune(n.XSolic)); t < 2 || t > 8000 {
		return Erros{erro(campo+".xSolic", "tem %d caracteres; o leiaute aceita de 2 a 8000", t)}
	}
	return nil
}

// faixa confere um decimal da reforma: sem sinal (os tipos RTC não aceitam negativo) e até o máximo do tipo.
func faixa(campo string, v tipos.Decimal, maximo tipos.Decimal) Erros {
	if v.Negativo() || v.Comparar(maximo) > 0 {
		return Erros{erro(campo, "%s fora do leiaute (de 0 a %s)", v, maximo)}
	}
	return nil
}

// faixasIBSCBS confere a faixa de cada alíquota (TDec_0302_04RTC) e de cada valor (TDec1302RTC) do grupo.
func faixasIBSCBS(campo string, g *IBSCBS) Erros {
	var e Erros
	al := func(nome string, v tipos.Decimal) { e = append(e, faixa(campo+"."+nome, v, maxAliquotaRTC)...) }
	vl := func(nome string, v tipos.Decimal) { e = append(e, faixa(campo+"."+nome, v, maxValorRTC)...) }
	subgrupos := func(prefixo string, dif *Diferimento, dev *DevTributo, red *Reducao) {
		if dif != nil {
			al(prefixo+".gDif.pDif", dif.PDif)
			vl(prefixo+".gDif.vDif", dif.VDif)
		}
		if dev != nil {
			if dev.PDevTrib != nil {
				al(prefixo+".gDevTrib.pDevTrib", *dev.PDevTrib)
			}
			vl(prefixo+".gDevTrib.vDevTrib", dev.VDevTrib)
		}
		if red != nil {
			al(prefixo+".gRed.pRedAliq", red.PRedAliq)
			al(prefixo+".gRed.pAliqEfet", red.PAliqEfet)
		}
	}
	if t := g.GIBSCBS; t != nil {
		vl("gIBSCBS.vBC", t.VBC)
		al("gIBSCBS.gIBSUF.pIBSUF", t.GIBSUF.PIBSUF)
		subgrupos("gIBSCBS.gIBSUF", t.GIBSUF.GDif, t.GIBSUF.GDevTrib, t.GIBSUF.GRed)
		vl("gIBSCBS.gIBSUF.vIBSUF", t.GIBSUF.VIBSUF)
		al("gIBSCBS.gIBSMun.pIBSMun", t.GIBSMun.PIBSMun)
		subgrupos("gIBSCBS.gIBSMun", t.GIBSMun.GDif, t.GIBSMun.GDevTrib, t.GIBSMun.GRed)
		vl("gIBSCBS.gIBSMun.vIBSMun", t.GIBSMun.VIBSMun)
		vl("gIBSCBS.vIBS", t.VIBS)
		al("gIBSCBS.gCBS.pCBS", t.GCBS.PCBS)
		subgrupos("gIBSCBS.gCBS", t.GCBS.GDif, t.GCBS.GDevTrib, t.GCBS.GRed)
		if z := t.GCBS.GALCZFMCBS; z != nil {
			al("gIBSCBS.gCBS.gALCZFMCBS.pAliqEfetRegCBS", z.PAliqEfetRegCBS)
			vl("gIBSCBS.gCBS.gALCZFMCBS.vTribRegCBS", z.VTribRegCBS)
		}
		vl("gIBSCBS.gCBS.vCBS", t.GCBS.VCBS)
		if r := t.GTribRegular; r != nil {
			al("gIBSCBS.gTribRegular.pAliqEfetRegIBSUF", r.PAliqEfetRegIBSUF)
			vl("gIBSCBS.gTribRegular.vTribRegIBSUF", r.VTribRegIBSUF)
			al("gIBSCBS.gTribRegular.pAliqEfetRegIBSMun", r.PAliqEfetRegIBSMun)
			vl("gIBSCBS.gTribRegular.vTribRegIBSMun", r.VTribRegIBSMun)
			al("gIBSCBS.gTribRegular.pAliqEfetRegCBS", r.PAliqEfetRegCBS)
			vl("gIBSCBS.gTribRegular.vTribRegCBS", r.VTribRegCBS)
		}
		if c := t.GTribCompraGov; c != nil {
			al("gIBSCBS.gTribCompraGov.pAliqIBSUF", c.PAliqIBSUF)
			vl("gIBSCBS.gTribCompraGov.vTribIBSUF", c.VTribIBSUF)
			al("gIBSCBS.gTribCompraGov.pAliqIBSMun", c.PAliqIBSMun)
			vl("gIBSCBS.gTribCompraGov.vTribIBSMun", c.VTribIBSMun)
			al("gIBSCBS.gTribCompraGov.pAliqCBS", c.PAliqCBS)
			vl("gIBSCBS.gTribCompraGov.vTribCBS", c.VTribCBS)
		}
	}
	if est := g.GEstornoCred; est != nil {
		vl("gEstornoCred.vIBSEstCred", est.VIBSEstCred)
		vl("gEstornoCred.vCBSEstCred", est.VCBSEstCred)
	}
	return e
}
