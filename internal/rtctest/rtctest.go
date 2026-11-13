// Package rtctest monta, para os testes, os grupos da reforma tributária do CT-e preenchidos e coerentes: cada
// subgrupo opcional presente, para que o exemplo prove contra o esquema a ordem e as casas de todos.
package rtctest

import (
	"encoding/base64"

	"github.com/sisgt-sistemas/gonfe/chave"
	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/tipos"
)

// ChaveDFe devolve uma chave de CT-e válida de outra transportadora, com o número dado.
func ChaveDFe(numero int) string {
	return chave.MustNova(chave.Chave{CUF: 43, Ano: 26, Mes: 10, CNPJ: "99999999000191", Modelo: 57, Serie: 1,
		Numero: numero, TpEmis: 1, CNF: "13572468"})
}

// IBSCBS devolve o grupo de IBS e CBS sobre a base dada, com IBS da UF de 0,1%, IBS municipal de 0% e CBS de 0,9%
// (as alíquotas de teste de 2026), e todos os subgrupos opcionais: diferimento, devolução, redução, CBS na Zona
// Franca, tributação regular, compra governamental e estorno de crédito.
func IBSCBS(base string) *cte.IBSCBS {
	bc := tipos.D(base)
	vIBSUF := bc.Percentual(tipos.D("0.10"), 2)
	vIBSMun := tipos.D("0.00")
	vCBS := bc.Percentual(tipos.D("0.90"), 2)
	return &cte.IBSCBS{
		CST: "000", CClassTrib: "000001", IndDoacao: "1",
		GIBSCBS: &cte.GIBSCBS{
			VBC: bc,
			GIBSUF: cte.IBSUF{
				PIBSUF:   tipos.D("0.1000"),
				GDif:     &cte.Diferimento{PDif: tipos.D("0.00"), VDif: tipos.D("0.00")},
				GDevTrib: &cte.DevTributo{PDevTrib: tipos.Ptr(tipos.D("0.00")), VDevTrib: tipos.D("0.00")},
				GRed:     &cte.Reducao{PRedAliq: tipos.D("0.00"), PAliqEfet: tipos.D("0.1000")},
				VIBSUF:   vIBSUF,
			},
			GIBSMun: cte.IBSMun{
				PIBSMun:  tipos.D("0.00"),
				GDif:     &cte.Diferimento{PDif: tipos.D("0.00"), VDif: tipos.D("0.00")},
				GDevTrib: &cte.DevTributo{VDevTrib: tipos.D("0.00")},
				GRed:     &cte.Reducao{PRedAliq: tipos.D("0.00"), PAliqEfet: tipos.D("0.00")},
				VIBSMun:  vIBSMun,
			},
			VIBS: vIBSUF.Somar(vIBSMun),
			GCBS: cte.CBS{
				PCBS:       tipos.D("0.9000"),
				GDif:       &cte.Diferimento{PDif: tipos.D("0.00"), VDif: tipos.D("0.00")},
				GDevTrib:   &cte.DevTributo{PDevTrib: tipos.Ptr(tipos.D("0.00")), VDevTrib: tipos.D("0.00")},
				GRed:       &cte.Reducao{PRedAliq: tipos.D("0.00"), PAliqEfet: tipos.D("0.9000")},
				GALCZFMCBS: &cte.ALCZFMCBS{PAliqEfetRegCBS: tipos.D("0.9000"), VTribRegCBS: vCBS},
				VCBS:       vCBS,
			},
			GTribRegular: &cte.TribRegular{
				CSTReg: "000", CClassTribReg: "000001",
				PAliqEfetRegIBSUF: tipos.D("0.1000"), VTribRegIBSUF: vIBSUF,
				PAliqEfetRegIBSMun: tipos.D("0.00"), VTribRegIBSMun: vIBSMun,
				PAliqEfetRegCBS: tipos.D("0.9000"), VTribRegCBS: vCBS,
			},
			GTribCompraGov: &cte.TribCompraGov{
				PAliqIBSUF: tipos.D("0.1000"), VTribIBSUF: vIBSUF,
				PAliqIBSMun: tipos.D("0.00"), VTribIBSMun: vIBSMun,
				PAliqCBS: tipos.D("0.9000"), VTribCBS: vCBS,
			},
		},
		GEstornoCred: &cte.EstornoCred{VIBSEstCred: tipos.D("0.00"), VCBSEstCred: tipos.D("0.00")},
	}
}

// CompraGov devolve uma compra governamental de ente federal, com um documento anterior.
func CompraGov() *cte.CompraGov {
	return &cte.CompraGov{TpEnteGov: "1", PRedutor: tipos.D("10.0000"), TpOperGov: "1", RefDFeAnt: []string{ChaveDFe(501)}}
}

// PgtoVinc devolve um pagamento vinculado.
func PgtoVinc() *cte.PgtoVinc {
	return &cte.PgtoVinc{Pgto: []cte.PagamentoRTC{{
		NPag: 1, IdTransacao: "TX-2026-0001", TpMeioPgto: "01", CNPJReceb: "11222333000181", CNPJBasePSP: "12345678",
	}}}
}

// PAA devolve um Provedor de Assinatura e Autorização com assinatura e chave em base64 (valores de teste).
func PAA() *cte.InfPAA {
	b64 := func(s string) string { return base64.StdEncoding.EncodeToString([]byte(s)) }
	return &cte.InfPAA{
		CNPJPAA: "11222333000181",
		PAASignature: cte.PAASignature{
			SignatureValue: b64("assinatura de teste"),
			RSAKeyValue:    cte.RSAKeyValue{Modulus: b64("modulo de teste"), Exponent: b64("AQAB")},
		},
	}
}

// NFF devolve uma solicitação da Nota Fiscal Fácil.
func NFF() *cte.InfSolicNFF { return &cte.InfSolicNFF{XSolic: "SOLICITACAO DE TESTE"} }
