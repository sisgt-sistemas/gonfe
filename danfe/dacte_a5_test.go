package danfe_test

import (
	"bytes"
	"errors"
	"fmt"
	"testing"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/danfe"
	"github.com/sisgt-sistemas/gonfe/tipos"
)

var meiaFolha = danfe.Opcoes{Papel: danfe.PapelA5}

// comTodasAsPartes acrescenta expedidor e recebedor e troca o tomador por um terceiro (toma4): é o conhecimento com
// mais quadros fixos.
func comTodasAsPartes(c *cte.CTe) *cte.CTe {
	end := func(lgr string) *cte.Endereco {
		return &cte.Endereco{XLgr: lgr, Nro: "1", XBairro: "CENTRO", CMun: 4314902, XMun: "PORTO ALEGRE", CEP: "90000000", UF: "RS"}
	}
	c.InfCte.Exped = &cte.Exped{CNPJ: "11444777000161", XNome: "EXPEDIDOR LTDA", IE: "1111111111", EnderExped: end("RUA DO EXPEDIDOR")}
	c.InfCte.Receb = &cte.Receb{CNPJ: "11444777000161", XNome: "RECEBEDOR LTDA", IE: "2222222222", EnderReceb: end("RUA DO RECEBEDOR")}
	c.InfCte.Ide.Toma3 = nil
	c.InfCte.Ide.Toma4 = &cte.Toma4{
		Toma: cte.TomadorOutros, CNPJ: "11444777000161", XNome: "TOMADOR TERCEIRO LTDA", IE: "5555555555",
		EnderToma: end("RUA DO TOMADOR"),
	}
	return c
}

// A meia folha é o A5 em paisagem: 210 × 148 mm, em pontos.
func TestDACTEMeiaFolhaEhA5Paisagem(t *testing.T) {
	dados, err := danfe.GerarDACTE(procCTeDe(t, conhecimentoExemplo(4)), meiaFolha)
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	conferirPDF(t, dados)
	if !bytes.Contains(dados, []byte("/MediaBox [0 0 595.28 419.53]")) {
		t.Error("a folha não é A5 paisagem")
	}
	if paginas := paginasDe(dados); paginas != 1 {
		t.Errorf("um conhecimento com 4 documentos saiu em %d folhas, queria 1", paginas)
	}
	for _, esperado := range []string{"DACTE", "INDUSTRIA REMETENTE SA", "COMERCIO DESTINATARIO LTDA", "1.000,00", "RNTRC DA EMPRESA"} {
		if !bytes.Contains(dados, []byte(esperado)) {
			t.Errorf("falta %q", esperado)
		}
	}
}

// O papel A4, o valor zero, não muda o DACTE.
func TestDACTEPapelA4EhOPadrao(t *testing.T) {
	proc := procCTeDe(t, conhecimentoExemplo(3))
	padrao, err := danfe.GerarDACTE(proc, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	a4, err := danfe.GerarDACTE(proc, danfe.Opcoes{Papel: danfe.PapelA4})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if !bytes.Equal(padrao, a4) {
		t.Error("PapelA4 mudou o DACTE padrão")
	}
}

// Os outros documentos do transporte ignoram o papel.
func TestPapelA5SoValeParaODACTE(t *testing.T) {
	proc := procMDFeDe(t, manifestoExemplo(2))
	a4, err := danfe.GerarDAMDFE(proc, danfe.Opcoes{})
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}
	a5, err := danfe.GerarDAMDFE(proc, meiaFolha)
	if err != nil {
		t.Fatalf("GerarDAMDFE: %v", err)
	}
	if !bytes.Equal(a4, a5) {
		t.Error("o DAMDFE mudou com PapelA5")
	}
}

// Sem pedido, a meia folha sai sem canhoto; com CanhotoNaMeiaFolha, ele sai no topo da primeira folha só.
func TestDACTEMeiaFolhaCanhoto(t *testing.T) {
	proc := procCTeDe(t, conhecimentoExemplo(150))
	sem, err := danfe.GerarDACTE(proc, meiaFolha)
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	com, err := danfe.GerarDACTE(proc, danfe.Opcoes{Papel: danfe.PapelA5, CanhotoNaMeiaFolha: true})
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if bytes.Contains(sem, []byte("DECLARO QUE RECEBI")) {
		t.Error("a meia folha saiu com canhoto sem pedido")
	}
	if vezes := bytes.Count(com, []byte("DECLARO QUE RECEBI")); vezes != 1 {
		t.Errorf("o canhoto saiu %d vezes, queria só na primeira folha", vezes)
	}
}

// Os documentos originários que não cabem vão para as folhas seguintes, que repetem o cabeçalho: nenhum se perde.
func TestDACTEMeiaFolhaLevaTodosOsDocumentos(t *testing.T) {
	const documentos = 150
	dados, err := danfe.GerarDACTE(procCTeDe(t, conhecimentoExemplo(documentos)), meiaFolha)
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	folhas := paginasDe(dados)
	if folhas < 2 {
		t.Fatalf("%d documentos saíram em %d folha", documentos, folhas)
	}
	if vezes := bytes.Count(dados, []byte("(NF-e) Tj")); vezes != documentos {
		t.Errorf("a tabela tem %d documentos, queria %d", vezes, documentos)
	}
	if vezes := bytes.Count(dados, []byte("(DACTE) Tj")); vezes != folhas {
		t.Errorf("o cabeçalho saiu %d vezes em %d folhas", vezes, folhas)
	}
	conferirFolhas(t, "meia folha", dados)
}

// O tomador que é uma das partes (toma3) sai numa linha; o de fora (toma4) leva o endereço, que não aparece em outro
// lugar.
func TestDACTEMeiaFolhaTomador(t *testing.T) {
	toma3, err := danfe.GerarDACTE(procCTeDe(t, conhecimentoExemplo(1)), meiaFolha)
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	if !bytes.Contains(toma3, []byte("TOMADOR DO SERVI")) {
		t.Error("falta o tomador")
	}
	if vezes := bytes.Count(toma3, []byte("(INDUSTRIA REMETENTE SA) Tj")); vezes != 2 {
		t.Errorf("o remetente aparece %d vezes; deveria sair no quadro dele e como tomador", vezes)
	}

	toma4, err := danfe.GerarDACTE(procCTeDe(t, comTodasAsPartes(conhecimentoExemplo(1))), meiaFolha)
	if err != nil {
		t.Fatalf("GerarDACTE: %v", err)
	}
	for _, esperado := range []string{"TOMADOR TERCEIRO LTDA", "RUA DO TOMADOR", "EXPEDIDOR LTDA", "RECEBEDOR LTDA"} {
		if !bytes.Contains(toma4, []byte(esperado)) {
			t.Errorf("falta %q", esperado)
		}
	}
}

// Um quadro mais alto que a meia folha inteira é recusado, em vez de sair cortado.
func TestDACTEMeiaFolhaQueNaoCabeERecusada(t *testing.T) {
	c := conhecimentoExemplo(1)
	for i := range 120 {
		c.InfCte.VPrest.Comp = append(c.InfCte.VPrest.Comp, cte.Componente{XNome: fmt.Sprintf("TAXA %d", i), VComp: tipos.D("1.00")})
	}
	if _, err := danfe.GerarDACTE(procCTeDe(t, c), meiaFolha); !errors.Is(err, danfe.ErrNaoCabe) {
		t.Errorf("erro = %v, queria ErrNaoCabe", err)
	}
}
