package cteos_test

import (
	"errors"
	"strings"
	"testing"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/cteos"
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
	c.InfCte.Imp.ICMSUFFim = &cte.ICMSUFFim{
		VBCUFFim: tipos.D("2500.00"), PFCPUFFim: tipos.D("2.00"), PICMSUFFim: tipos.D("17.00"), PICMSInter: tipos.D("12.00"),
		VFCPUFFim: tipos.D("50.00"), VICMSUFFim: tipos.D("125.00"), VICMSUFIni: tipos.D("0.00"),
	}
	// Os tributos federais ficam entre a partilha e o IBS/CBS: o esquema confere a ordem.
	c.InfCte.Imp.InfTribFed = &cteos.InfTribFed{
		VPIS: tipos.Ptr(tipos.D("16.25")), VCOFINS: tipos.Ptr(tipos.D("75.00")), VIR: tipos.Ptr(tipos.D("37.50")),
		VINSS: tipos.Ptr(tipos.D("0.00")), VCSLL: tipos.Ptr(tipos.D("25.00")),
	}
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

// O percurso do veículo é conferido: UF válidas, no máximo 25.
func TestPercursoDoVeiculo(t *testing.T) {
	c := fretamentoExemplo()
	c.InfCte.Ide.InfPercurso = []cteos.Percurso{{UFPer: "SC"}, {UFPer: "XX"}}
	if err := c.Preparar(); err != nil {
		t.Fatal(err)
	}
	if err := c.Validar(); err == nil || !strings.Contains(err.Error(), "ide.infPercurso[1].UFPer") {
		t.Errorf("UF inválida no percurso aceita: %v", err)
	}
	c.InfCte.Ide.InfPercurso = make([]cteos.Percurso, 26)
	for i := range c.InfCte.Ide.InfPercurso {
		c.InfCte.Ide.InfPercurso[i] = cteos.Percurso{UFPer: "SC"}
	}
	if err := c.Validar(); err == nil || !strings.Contains(err.Error(), "ide.infPercurso") {
		t.Errorf("percurso com 26 UF aceito: %v", err)
	}
}

func TestTributosFederaisNaoNegativos(t *testing.T) {
	c := fretamentoExemplo()
	c.InfCte.Imp.InfTribFed = &cteos.InfTribFed{VIR: tipos.Ptr(tipos.D("-1.00"))}
	if err := c.Preparar(); err != nil {
		t.Fatal(err)
	}
	if err := c.Validar(); err == nil || !strings.Contains(err.Error(), "imp.infTribFed.vIR") {
		t.Errorf("IR negativo aceito: %v", err)
	}
}
