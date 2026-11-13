// Package archtest guarda as regras do projeto que valem para todo pacote: as que o compilador não confere.
package archtest

import (
	"go/ast"
	"go/parser"
	"go/token"
	"io/fs"
	"os"
	"path/filepath"
	"reflect"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"testing"
)

const root = "../.."

// goFiles percorre os arquivos Go do repositório, fora de .git e docs.
func goFiles(t *testing.T, visit func(relative string, test bool)) {
	t.Helper()
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if entry.IsDir() && (entry.Name() == ".git" || entry.Name() == "docs") {
			return filepath.SkipDir
		}
		if !entry.IsDir() && strings.HasSuffix(path, ".go") {
			relative, _ := filepath.Rel(root, path)
			visit(filepath.ToSlash(relative), strings.HasSuffix(path, "_test.go"))
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}
}

func parse(t *testing.T, relative string, mode parser.Mode) *ast.File {
	t.Helper()
	file, err := parser.ParseFile(token.NewFileSet(), filepath.Join(root, relative), nil, mode)
	if err != nil {
		t.Fatal(err)
	}
	return file
}

func imports(t *testing.T, relative string) []string {
	t.Helper()
	var out []string
	for _, spec := range parse(t, relative, parser.ImportsOnly).Imports {
		value, _ := strconv.Unquote(spec.Path.Value)
		out = append(out, value)
	}
	return out
}

// TestNoCGO: a biblioteca não usa CGO, para compilar em qualquer sistema sem toolchain de C (convenção do projeto).
func TestNoCGO(t *testing.T) {
	goFiles(t, func(relative string, _ bool) {
		if slices.Contains(imports(t, relative), "C") {
			t.Errorf("%s: import \"C\"; a biblioteca não usa CGO", relative)
		}
	})
}

// TestOnlyXSDRunsProcesses: só o pacote xsd chama programa externo (o xmllint, nos testes). Nenhum caminho de emissão
// depende de algo instalado na máquina.
func TestOnlyXSDRunsProcesses(t *testing.T) {
	goFiles(t, func(relative string, test bool) {
		if test || strings.HasPrefix(relative, "xsd/") {
			return
		}
		if slices.Contains(imports(t, relative), "os/exec") {
			t.Errorf("%s: os/exec fora do pacote xsd", relative)
		}
	})
}

// schemaChecked são os pacotes de documento validados contra o esquema oficial: cada um tem teste que chama o xsd.
var schemaChecked = []string{"cte", "cteos", "ctesimp", "mdfe"}

// TestEveryDocumentPassesTheSchema: todo documento validado contra o esquema tem teste que o confere com o xsd.
// Tirar o teste não passa despercebido.
func TestEveryDocumentPassesTheSchema(t *testing.T) {
	calls := regexp.MustCompile(`xsd\.Validar(CTe|CTeOS|CTeSimp|MDFe)?\(`)
	for _, pkg := range schemaChecked {
		found := false
		entries, err := os.ReadDir(filepath.Join(root, pkg))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			content, err := os.ReadFile(filepath.Join(root, pkg, entry.Name()))
			if err != nil {
				t.Fatal(err)
			}
			if calls.Match(content) {
				found = true
			}
		}
		if !found {
			t.Errorf("%s: nenhum teste valida o documento contra o esquema oficial (xsd)", pkg)
		}
	}
}

// casas é o intervalo de casas decimais que o esquema aceita para um elemento.
type casas struct{ min, max int }

// tdec lê o nome do tipo decimal do leiaute: TDec_1302 (2 casas), TDec_0302a04 e TDec_0302_0303 (2 a 4 e 2 a 3),
// com ou sem Opc.
var tdec = regexp.MustCompile(`^TDec_\d{2}(\d{2})(?:(?:a|_\d{2})(\d{2}))?(?:Opc)?$`)

// schemaDecimals lê, dos esquemas de uma família, as casas decimais de cada elemento de tipo TDec. Um nome que aparece
// com tipos diferentes em grupos diferentes junta os intervalos.
func schemaDecimals(t *testing.T, folder string) map[string][]casas {
	t.Helper()
	element := regexp.MustCompile(`<xs:element name="([A-Za-z0-9]+)"[^>]*type="([A-Za-z0-9_]+)"`)
	out := map[string][]casas{}
	files, err := filepath.Glob(filepath.Join(root, "xsd", "esquemas", folder, "*.xsd"))
	if err != nil || len(files) == 0 {
		t.Fatalf("esquemas de %s: %v", folder, err)
	}
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		for _, match := range element.FindAllStringSubmatch(string(content), -1) {
			parts := tdec.FindStringSubmatch(match[2])
			if parts == nil {
				continue
			}
			minimum, _ := strconv.Atoi(parts[1])
			maximum := minimum
			if parts[2] != "" {
				maximum, _ = strconv.Atoi(parts[2])
			}
			if !slices.Contains(out[match[1]], casas{minimum, maximum}) {
				out[match[1]] = append(out[match[1]], casas{minimum, maximum})
			}
		}
	}
	return out
}

// familyOf é a pasta de esquemas de cada pacote de documento.
var familyOf = map[string]string{"cte": "cte", "cteos": "cte", "ctesimp": "cte", "mdfe": "mdfe"}

// TestDecimalPlacesFollowTheSchema: a tag dec:"N" de cada campo decimal está dentro das casas que o esquema dá ao
// elemento (TDec_*). O pICMS com quatro casas, que a SEFAZ recusava, não volta.
func TestDecimalPlacesFollowTheSchema(t *testing.T) {
	families := map[string]map[string][]casas{}
	for pkg, folder := range familyOf {
		if families[folder] == nil {
			families[folder] = schemaDecimals(t, folder)
		}
		allowed := families[folder]
		checked := 0
		entries, err := os.ReadDir(filepath.Join(root, pkg))
		if err != nil {
			t.Fatal(err)
		}
		for _, entry := range entries {
			if !strings.HasSuffix(entry.Name(), ".go") || strings.HasSuffix(entry.Name(), "_test.go") {
				continue
			}
			relative := pkg + "/" + entry.Name()
			ast.Inspect(parse(t, relative, 0), func(node ast.Node) bool {
				field, ok := node.(*ast.Field)
				if !ok || field.Tag == nil {
					return true
				}
				tag, _ := strconv.Unquote(field.Tag.Value)
				decimals, has := reflect.StructTag(tag).Lookup("dec")
				name, _, _ := strings.Cut(reflect.StructTag(tag).Get("xml"), ",")
				ranges, known := allowed[name]
				if !has || !known {
					return true
				}
				checked++
				n, err := strconv.Atoi(decimals)
				if err != nil {
					t.Errorf("%s: %s com dec:%q", relative, name, decimals)
					return true
				}
				for _, r := range ranges {
					if n >= r.min && n <= r.max {
						return true
					}
				}
				t.Errorf("%s: %s com dec:\"%d\"; o esquema aceita %v casas", relative, name, n, ranges)
				return true
			})
		}
		if checked == 0 {
			t.Errorf("%s: nenhum campo decimal conferido; o gate não está lendo o pacote", pkg)
		}
	}
}

// chaveOuCNPJ casa os nomes de elemento que levam CNPJ ou chave de acesso: CNPJ, CNPJFab, chNFe, chCTe, refNFe,
// chave. Esses campos podem ter letras (CNPJ alfanumérico, NT 2025.001).
var chaveOuCNPJ = regexp.MustCompile(`^(CNPJ[A-Za-z]*|chave|ch[A-Z][A-Za-z]*|ref(NFe|NFeSig|CTe|Cte|CTeCanc|CteAnu))$`)

// elementosAlfanumericos lê, dos esquemas de uma família, os elementos cujo tipo aceita letras nas posições do CNPJ
// (TCnpj, TCnpjOpc, TChDFe, TChNFe e os padrões em linha com [A-Z0-9]{12}).
func elementosAlfanumericos(t *testing.T, folder string) map[string]bool {
	t.Helper()
	tipos := regexp.MustCompile(`<xs:simpleType name="([A-Za-z0-9_]+)">(?s:.*?)</xs:simpleType>`)
	elemento := regexp.MustCompile(`<xs:element name="([A-Za-z0-9]+)"([^>]*)>`)
	alfa := map[string]bool{}
	out := map[string]bool{}
	files, err := filepath.Glob(filepath.Join(root, "xsd", "esquemas", folder, "*.xsd"))
	if err != nil || len(files) == 0 {
		t.Fatalf("esquemas de %s: %v", folder, err)
	}
	var conteudos []string
	for _, file := range files {
		content, err := os.ReadFile(file)
		if err != nil {
			t.Fatal(err)
		}
		conteudos = append(conteudos, string(content))
		for _, m := range tipos.FindAllStringSubmatch(string(content), -1) {
			if strings.Contains(m[0], "[A-Z0-9]{12}") {
				alfa[m[1]] = true
			}
		}
	}
	for _, content := range conteudos {
		for _, m := range elemento.FindAllStringSubmatchIndex(content, -1) {
			name, attrs := content[m[2]:m[3]], content[m[4]:m[5]]
			if tipo := regexp.MustCompile(`type="([A-Za-z0-9_]+)"`).FindStringSubmatch(attrs); tipo != nil {
				if alfa[tipo[1]] {
					out[name] = true
				}
				continue
			}
			// Tipo em linha: o padrão vem logo depois, antes do fim do elemento.
			fim := strings.Index(content[m[1]:], "</xs:element>")
			if fim > 0 && strings.Contains(content[m[1]:m[1]+fim], "[A-Z0-9]{12}") {
				out[name] = true
			}
		}
	}
	return out
}

// TestCNPJAndKeysKeepLetters: campo de CNPJ ou de chave de acesso não usa norm:"num", que apagaria as letras do CNPJ
// alfanumérico; usa norm:"alnum". Vale para os nomes conhecidos em todo pacote e, onde há esquema embutido, para todo
// elemento cujo tipo aceita letras.
func TestCNPJAndKeysKeepLetters(t *testing.T) {
	esquema := map[string]map[string]bool{"cte": elementosAlfanumericos(t, "cte"), "mdfe": elementosAlfanumericos(t, "mdfe")}
	familia := map[string]string{"cte": "cte", "cteos": "cte", "ctesimp": "cte", "mdfe": "mdfe"}
	conferidos := 0
	goFiles(t, func(relative string, test bool) {
		if test {
			return
		}
		pkg, _, _ := strings.Cut(relative, "/")
		ast.Inspect(parse(t, relative, 0), func(node ast.Node) bool {
			field, ok := node.(*ast.Field)
			if !ok || field.Tag == nil {
				return true
			}
			tag, _ := strconv.Unquote(field.Tag.Value)
			name, _, _ := strings.Cut(reflect.StructTag(tag).Get("xml"), ",")
			if !chaveOuCNPJ.MatchString(name) && !esquema[familia[pkg]][name] {
				return true
			}
			conferidos++
			if slices.Contains(strings.Split(reflect.StructTag(tag).Get("norm"), ","), "num") {
				t.Errorf("%s: %s com norm:\"num\" apaga as letras do CNPJ alfanumérico; use norm:\"alnum\"", relative, name)
			}
			return true
		})
	})
	if conferidos < 80 {
		t.Errorf("só %d campos de CNPJ ou chave conferidos; o gate não está lendo os pacotes", conferidos)
	}
}
