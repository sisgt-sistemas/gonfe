// Package xsd valida um documento contra os esquemas oficiais da SEFAZ (CT-e 4.00 e MDF-e 3.00b), embutidos aqui.
//
// É ferramenta de teste, não do caminho de emissão. Go não tem validador de XSD na biblioteca padrão, e o caminho
// usual (libxml2 por CGO) contraria a convenção do projeto de não usar CGO. Por isso o pacote chama o programa
// xmllint, da libxml2, como processo externo: sem ele instalado, [Validar] devolve [ErrSemXmllint] e o teste deve ser
// pulado. A emissão continua validando em Go, pelo Validar() de cada documento.
//
// Os esquemas e a origem de cada pacote estão no README desta pasta.
package xsd

import (
	"bytes"
	"embed"
	"errors"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

//go:embed esquemas/cte/*.xsd esquemas/mdfe/*.xsd
var esquemas embed.FS

// Esquema é o arquivo de esquema, relativo à pasta esquemas.
type Esquema string

// Os esquemas de documento e de modal. O infModal do CT-e e do MDF-e é xs:any com processContents="skip": o modal
// se valida à parte, pelo esquema dele ([ValidarCTe] e [ValidarMDFe] fazem os dois).
const (
	CTe                 Esquema = "cte/cte_v4.00.xsd"
	CTeSimp             Esquema = "cte/cteSimp_v4.00.xsd" // CT-e Simplificado (modelo 57, raiz CTeSimp)
	CTeModalRodoviario  Esquema = "cte/cteModalRodoviario_v4.00.xsd"
	MDFe                Esquema = "mdfe/mdfe_v3.00.xsd"
	MDFeModalRodoviario Esquema = "mdfe/mdfeModalRodoviario_v3.00.xsd"
)

// Os namespaces dos documentos.
const (
	namespaceCTe  = "http://www.portalfiscal.inf.br/cte"
	namespaceMDFe = "http://www.portalfiscal.inf.br/mdfe"
)

var (
	// ErrSemXmllint diz que o programa xmllint não está instalado: o teste que valida contra o esquema é pulado.
	ErrSemXmllint = errors.New("xsd: xmllint não encontrado (pacote libxml2-utils)")
	// ErrInvalido é a base do erro de documento que o esquema recusa; a mensagem traz o que o xmllint disse.
	ErrInvalido = errors.New("xsd: documento recusado pelo esquema")
)

// Validar confere o documento contra o esquema. Recusa devolve um erro que embrulha [ErrInvalido], com a saída do
// xmllint (linha, elemento e motivo).
func Validar(documento []byte, esquema Esquema) error {
	xmllint, err := exec.LookPath("xmllint")
	if err != nil {
		return ErrSemXmllint
	}
	if _, err := fs.Stat(esquemas, "esquemas/"+string(esquema)); err != nil {
		return fmt.Errorf("xsd: esquema %q desconhecido: %w", esquema, err)
	}
	pasta, err := os.MkdirTemp("", "gonfe-xsd-")
	if err != nil {
		return fmt.Errorf("xsd: pasta temporária: %w", err)
	}
	defer func() { _ = os.RemoveAll(pasta) }() // Pasta temporária: sobra, o sistema limpa.
	// O esquema inclui os vizinhos pelo nome: a pasta inteira vai para o disco.
	if err := os.CopyFS(pasta, esquemas); err != nil {
		return fmt.Errorf("xsd: extrair os esquemas: %w", err)
	}
	arquivo := filepath.Join(pasta, "documento.xml")
	if err := os.WriteFile(arquivo, documento, 0o600); err != nil {
		return fmt.Errorf("xsd: gravar o documento: %w", err)
	}
	comando := exec.Command(xmllint, //nolint:gosec // G204: o xmllint vem do exec.LookPath, e os argumentos são caminhos que este pacote cria.
		"--noout", "--nonet", "--schema",
		filepath.Join(pasta, "esquemas", filepath.FromSlash(string(esquema))), arquivo)
	var saida bytes.Buffer
	comando.Stdout, comando.Stderr = &saida, &saida
	if err := comando.Run(); err != nil {
		var exit *exec.ExitError
		if errors.As(err, &exit) {
			return fmt.Errorf("%w (%s): %s", ErrInvalido, esquema, strings.TrimSpace(strings.ReplaceAll(saida.String(), pasta+string(filepath.Separator), "")))
		}
		return fmt.Errorf("xsd: rodar o xmllint: %w", err)
	}
	return nil
}

// ValidarCTe confere um CT-e (assinado) contra o esquema do documento e o modal rodoviário contra o esquema dele.
func ValidarCTe(documento []byte) error {
	if err := Validar(documento, CTe); err != nil {
		return err
	}
	modal, err := Modal(documento, namespaceCTe)
	if err != nil {
		return err
	}
	return Validar(modal, CTeModalRodoviario)
}

// ValidarCTeSimp confere um CT-e Simplificado (assinado) contra o esquema dele e o modal rodoviário contra o esquema
// do modal, o mesmo do CT-e.
func ValidarCTeSimp(documento []byte) error {
	if err := Validar(documento, CTeSimp); err != nil {
		return err
	}
	modal, err := Modal(documento, namespaceCTe)
	if err != nil {
		return err
	}
	return Validar(modal, CTeModalRodoviario)
}

// ValidarMDFe confere um MDF-e (assinado) contra o esquema do documento e o modal rodoviário contra o esquema dele.
func ValidarMDFe(documento []byte) error {
	if err := Validar(documento, MDFe); err != nil {
		return err
	}
	modal, err := Modal(documento, namespaceMDFe)
	if err != nil {
		return err
	}
	return Validar(modal, MDFeModalRodoviario)
}

// Modal tira de dentro do infModal o elemento do modal, como documento próprio no namespace dado: é o que o esquema
// do modal valida.
func Modal(documento []byte, namespace string) ([]byte, error) {
	inicio := bytes.Index(documento, []byte("<infModal"))
	if inicio < 0 {
		return nil, fmt.Errorf("%w: sem infModal", ErrInvalido)
	}
	abre := bytes.IndexByte(documento[inicio:], '>')
	fim := bytes.Index(documento, []byte("</infModal>"))
	if abre < 0 || fim < 0 || inicio+abre+1 > fim {
		return nil, fmt.Errorf("%w: infModal malformado", ErrInvalido)
	}
	conteudo := bytes.TrimSpace(documento[inicio+abre+1 : fim])
	if len(conteudo) < 2 || conteudo[0] != '<' {
		return nil, fmt.Errorf("%w: infModal vazio", ErrInvalido)
	}
	// O elemento herdava o namespace do documento; sozinho, precisa declará-lo.
	nome := bytes.IndexAny(conteudo, " />")
	if nome < 0 {
		return nil, fmt.Errorf("%w: modal malformado", ErrInvalido)
	}
	if !bytes.Contains(conteudo[:bytes.IndexByte(conteudo, '>')], []byte("xmlns=")) {
		conteudo = append(append(append([]byte{}, conteudo[:nome]...), []byte(` xmlns="`+namespace+`"`)...), conteudo[nome:]...)
	}
	return conteudo, nil
}
