package cteos_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/internal/rtctest"
	"github.com/sisgt-sistemas/gonfe/tipos"
	"github.com/sisgt-sistemas/gonfe/xsd"
)

// O CT-e OS com os grupos da reforma que o esquema prevê para ele (sem NFF e sem PAA, que o TCTeOS não tem) passa no
// esquema oficial e no Validar.
func TestCTeOSComReformaPassaNoEsquema(t *testing.T) {
	c := fretamentoExemplo()
	ide := &c.InfCte.Ide
	ide.GCompraGov = rtctest.CompraGov()
	ide.TpPagAnt = "3"
	ide.GPagAntecipado = &cte.PagAntecipado{ChDFePagAnt: rtctest.ChaveDFe(701)}
	ide.GMultaJuros = &cte.MultaJuros{ChCTeRef: rtctest.ChaveDFe(702)}
	c.InfCte.VPrest.VTPrestLiq = tipos.Ptr(tipos.D("2475.00"))
	c.InfCte.Imp.IBSCBS = rtctest.IBSCBS("2500.00")
	c.InfCte.Imp.VTotDFe = tipos.Ptr(tipos.D("2525.00"))
	c.InfCte.PgtoVinc = rtctest.PgtoVinc()

	assinado, err := c.AssinarCom(certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora}))
	if err != nil {
		t.Fatalf("AssinarCom: %v", err)
	}
	if err := c.Validar(); err != nil {
		t.Errorf("Validar recusou o exemplo com a reforma: %v", err)
	}
	err = xsd.ValidarCTeOS(assinado)
	if errors.Is(err, xsd.ErrSemXmllint) {
		t.Skip(err)
	}
	if err != nil {
		t.Fatalf("o CT-e OS com a reforma não passa no esquema: %v", err)
	}
}

// O CT-e OS aceita menos formas de emissão que o CT-e (TCTeOS: 1, 2, 7 e 8): o 3 e o 4 são recusados.
func TestFormasDeEmissaoDoCTeOS(t *testing.T) {
	for _, tp := range []cte.TipoEmissao{cte.EmissaoRegimeEspecialNFF, cte.EmissaoEPEC} {
		c := fretamentoExemplo()
		c.InfCte.Ide.TpEmis = tp
		c.InfCte.Ide.DhCont = tipos.Ptr(tipos.DH("2026-03-04T07:00:00-03:00"))
		c.InfCte.Ide.XJust = "SEFAZ FORA DO AR NO MOMENTO"
		if err := c.Preparar(); err != nil {
			t.Fatal(err)
		}
		if err := c.Validar(); err == nil || !strings.Contains(err.Error(), "ide.tpEmis") {
			t.Errorf("tpEmis %s aceito no CT-e OS: %v", tp, err)
		}
	}
}
