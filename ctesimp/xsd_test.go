package ctesimp_test

import (
	"errors"
	"testing"

	"github.com/sisgt-sistemas/gonfe/ctesimp"
	"github.com/sisgt-sistemas/gonfe/internal/certtest"
	"github.com/sisgt-sistemas/gonfe/xsd"
)

// Os CT-e Simplificados de exemplo, assinados, passam no esquema oficial (cteSimp_v4.00) e o modal rodoviário no
// esquema dele: as entregas com NF-e, contêiner e componentes, o redespacho com o documento anterior e a substituição
// em contingência com todos os opcionais.
func TestCTeSimpPassaNoEsquema(t *testing.T) {
	cert := certtest.MustGerar(certtest.Opcoes{CNPJ: cnpjTransportadora})
	for nome, exemplo := range map[string]func() *ctesimp.CTeSimp{
		"entregas":   entregasExemplo,
		"redespacho": redespachoExemplo,
		"completo":   completoExemplo,
	} {
		t.Run(nome, func(t *testing.T) {
			assinado, err := exemplo().AssinarCom(cert)
			if err != nil {
				t.Fatalf("AssinarCom: %v", err)
			}
			err = xsd.ValidarCTeSimp(assinado)
			if errors.Is(err, xsd.ErrSemXmllint) {
				t.Skip(err)
			}
			if err != nil {
				t.Fatalf("o CT-e Simplificado de exemplo não passa no esquema: %v", err)
			}
		})
	}
}
