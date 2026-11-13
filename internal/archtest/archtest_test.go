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
