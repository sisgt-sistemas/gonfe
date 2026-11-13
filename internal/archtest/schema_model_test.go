package archtest

import (
	"encoding"
	"encoding/xml"
	"os"
	"path/filepath"
	"reflect"
	"slices"
	"strings"
	"testing"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/cteos"
	"github.com/sisgt-sistemas/gonfe/ctesimp"
	"github.com/sisgt-sistemas/gonfe/mdfe"
)

// O modelo de cada documento espelha o esquema oficial: cada caminho de elemento e de atributo que o esquema admite,
// da raiz às folhas, tem campo no modelo, e cada tag xml do modelo existe no esquema. É o que prepara o projeto para
// uma nota técnica: troca-se o .xsd, e este gate lista o que o modelo ainda não tem.
//
// O infModal é xs:any no esquema do documento, e cada modal tem esquema próprio: o gate enxerta no lugar do any a raiz
// do modal rodoviário, o único que o projeto emite (decisão de 2026-11-13); os grupos dos outros modais no modelo
// ficam fora do escopo e não são conferidos.
//
// As exceções têm motivo e cobrem a subárvore: semCampo é o que o esquema tem e o modelo ainda não, semEsquema o que
// o modelo tem e o esquema não (e Validar recusa). Uma exceção que não cobre mais nada reprova, para a lista não
// envelhecer.

// raizDoEsquema é um documento: o esquema, o elemento raiz, o tipo Go que o modela e os modais enxertados no
// infModal.
type raizDoEsquema struct {
	nome         string
	esquema      string // relativo a xsd/esquemas
	elemento     string
	modelo       reflect.Type
	modais       []modalDoEsquema
	foraDoEscopo []string          // grupos do modelo que o gate não confere
	semCampo     map[string]string // caminho → por que o modelo não o tem
	semEsquema   map[string]string // caminho → por que o modelo o tem, se o esquema não
}

type modalDoEsquema struct {
	esquema, elemento string
}

// Os motivos das exceções.
const (
	assinatura = "a assinatura é inserida pelo xmldsig ao assinar"
	aModelar   = "a modelar (tms#116)"
	naDCe      = "entra com a DC-e (tms#114)"
	herdadoOS  = "tipo compartilhado com o CT-e; o leiaute do OS não o tem, e o Validar do OS o recusa"
)

// faltas monta as exceções a partir de um caminho base, com o mesmo motivo.
func faltas(motivo, base string, caminhos ...string) map[string]string {
	m := map[string]string{}
	for _, c := range caminhos {
		m[base+"/"+c] = motivo
	}
	return m
}

// juntarExcecoes une os mapas de exceções.
func juntarExcecoes(mapas ...map[string]string) map[string]string {
	m := map[string]string{}
	for _, mapa := range mapas {
		for chave, motivo := range mapa {
			m[chave] = motivo
		}
	}
	return m
}

// icmsAModelar são o cBenef e o vICMSDeson dos grupos do ICMS e o ICMS previsto no pagamento antecipado.
func icmsAModelar(imp string, grupos ...string) map[string]string {
	m := faltas(aModelar, imp+"/ICMS", "gICMSPrevistoPagtoAntecip")
	for _, g := range grupos {
		m[imp+"/ICMS/"+g+"/cBenef"] = aModelar
		m[imp+"/ICMS/"+g+"/vICMSDeson"] = aModelar
	}
	return m
}

// outrosModais são os grupos dos modais que o projeto não emite, no infModal do CT-e e do MDF-e.
func outrosModais(infModal string, modais ...string) []string {
	caminhos := make([]string, 0, len(modais))
	for _, modal := range modais {
		caminhos = append(caminhos, infModal+"/"+modal)
	}
	return caminhos
}

func raizesDosEsquemas() []raizDoEsquema {
	modaisCTe := []string{"aereo", "aquav", "ferrov", "duto", "multimodal"}
	return []raizDoEsquema{
		{
			nome: "CT-e", esquema: "cte/cte_v4.00.xsd", elemento: "CTe", modelo: reflect.TypeFor[cte.CTe](),
			modais:       []modalDoEsquema{{"cte/cteModalRodoviario_v4.00.xsd", "rodo"}},
			foraDoEscopo: outrosModais("CTe/infCte/infCTeNorm/infModal", modaisCTe...),
			semCampo: juntarExcecoes(
				map[string]string{"CTe/Signature": assinatura, "CTe/infCte/emit/ISUFEmit": aModelar},
				icmsAModelar("CTe/infCte/imp", "ICMS20", "ICMS45", "ICMS60", "ICMS90", "ICMSOutraUF"),
				faltas(naDCe, "CTe/infCte/infCTeNorm/infDoc", "infDCe"),
				faltas(aModelar, "CTe/infCte/infCTeNorm/infDoc",
					"infNF/infUnidCarga", "infNF/infUnidTransp", "infNFe/infUnidCarga", "infNFe/infUnidTransp",
					"infOutros/infUnidCarga", "infOutros/infUnidTransp"),
			),
		},
		{
			nome: "CT-e OS", esquema: "cte/cteOS_v4.00.xsd", elemento: "CTeOS", modelo: reflect.TypeFor[cteos.CTeOS](),
			modais: []modalDoEsquema{{"cte/cteModalRodoviarioOS_v4.00.xsd", "rodoOS"}},
			semCampo: juntarExcecoes(
				map[string]string{
					"CTeOS/Signature": assinatura, "CTeOS/infCte/emit/ISUFEmit": aModelar,
					"CTeOS/infCte/infCTeNorm/infGTVe/Comp/xComp": aModelar,
				},
				icmsAModelar("CTeOS/infCte/imp", "ICMS20", "ICMS45", "ICMS90", "ICMSOutraUF"),
			),
			semEsquema: faltas(herdadoOS, "CTeOS/infCte",
				"compl/Entrega", "compl/destCalc", "compl/fluxo", "compl/origCalc", "emit/CPF", "imp/ICMS/ICMS60",
				"infCTeNorm/infCteSub/indAlteraToma"),
		},
		{
			nome: "CT-e Simplificado", esquema: "cte/cteSimp_v4.00.xsd", elemento: "CTeSimp",
			modelo:       reflect.TypeFor[ctesimp.CTeSimp](),
			modais:       []modalDoEsquema{{"cte/cteModalRodoviario_v4.00.xsd", "rodo"}},
			foraDoEscopo: outrosModais("CTeSimp/infCte/infModal", modaisCTe...),
			semCampo: juntarExcecoes(
				map[string]string{"CTeSimp/Signature": assinatura},
				icmsAModelar("CTeSimp/infCte/imp", "ICMS20", "ICMS45", "ICMS60", "ICMS90", "ICMSOutraUF"),
			),
		},
		{
			nome: "MDF-e", esquema: "mdfe/mdfe_v3.00.xsd", elemento: "MDFe", modelo: reflect.TypeFor[mdfe.MDFe](),
			modais:       []modalDoEsquema{{"mdfe/mdfeModalRodoviario_v3.00.xsd", "rodo"}},
			foraDoEscopo: outrosModais("MDFe/infMDFe/infModal", "aereo", "aquav", "ferrov"),
			semCampo: juntarExcecoes(
				map[string]string{"MDFe/Signature": assinatura},
				faltas(aModelar, "MDFe/infMDFe",
					"infDoc/infMunDescarga/infCTe/indPrestacaoParcial", "infDoc/infMunDescarga/infCTe/infEntregaParcial",
					"infDoc/infMunDescarga/infCTe/infNFePrestParcial", "infModal/rodo/infANTT/infPag/indAntecipaAdiant",
					"infModal/rodo/infANTT/infPag/tpAntecip", "infPAA", "infSolicNFF", "prodPred/NCM", "prodPred/cEAN"),
			),
		},
	}
}

func TestModelFollowsTheSchema(t *testing.T) {
	for _, raiz := range raizesDosEsquemas() {
		t.Run(raiz.nome, func(t *testing.T) {
			doEsquema := caminhosDoEsquema(t, raiz)
			doModelo := map[string]bool{raiz.elemento: true}
			caminhosDoModelo(raiz.modelo, raiz.elemento, doModelo, 0)

			var faltam, sobram, excecoesVelhas []string
			for _, grupo := range raiz.foraDoEscopo {
				if len(cobertos(grupo, doModelo)) == 0 {
					excecoesVelhas = append(excecoesVelhas, grupo+" (fora do escopo)")
				}
				for _, caminho := range cobertos(grupo, doModelo) {
					delete(doModelo, caminho)
				}
			}
			for caminho := range doEsquema {
				if !doModelo[caminho] && !excecao(raiz.semCampo, caminho) {
					faltam = append(faltam, caminho)
				}
			}
			for caminho := range doModelo {
				if !doEsquema[caminho] && !excecao(raiz.semEsquema, caminho) {
					sobram = append(sobram, caminho)
				}
			}
			for chave := range raiz.semCampo {
				if !slices.ContainsFunc(cobertos(chave, doEsquema), func(c string) bool { return !doModelo[c] }) {
					excecoesVelhas = append(excecoesVelhas, chave)
				}
			}
			for chave := range raiz.semEsquema {
				if !slices.ContainsFunc(cobertos(chave, doModelo), func(c string) bool { return !doEsquema[c] }) {
					excecoesVelhas = append(excecoesVelhas, chave)
				}
			}
			relatar(t, "o esquema tem e o modelo não", faltam)
			relatar(t, "o modelo tem e o esquema não (tag com nome ou lugar errado)", sobram)
			relatar(t, "exceção que não é mais necessária", excecoesVelhas)
		})
	}
}

// cobertos devolve os caminhos do conjunto que a chave cobre: ela mesma e a subárvore.
func cobertos(chave string, conjunto map[string]bool) []string {
	var achados []string
	for caminho := range conjunto {
		if caminho == chave || strings.HasPrefix(caminho, chave+"/") {
			achados = append(achados, caminho)
		}
	}
	return achados
}

// excecao diz se alguma chave do mapa cobre o caminho.
func excecao(excecoes map[string]string, caminho string) bool {
	for chave := range excecoes {
		if caminho == chave || strings.HasPrefix(caminho, chave+"/") {
			return true
		}
	}
	return false
}

func relatar(t *testing.T, oQue string, caminhos []string) {
	t.Helper()
	if len(caminhos) == 0 {
		return
	}
	slices.Sort(caminhos)
	t.Errorf("%s (%d):\n  %s", oQue, len(caminhos), strings.Join(caminhos, "\n  "))
}

// noXSD é um nó qualquer de um esquema, com os atributos e os filhos.
type noXSD struct {
	XMLName xml.Name
	Attrs   []xml.Attr `xml:",any,attr"`
	Filhos  []noXSD    `xml:",any"`
}

func (n noXSD) attr(nome string) string {
	for _, a := range n.Attrs {
		if a.Name.Local == nome {
			return a.Value
		}
	}
	return ""
}

// esquemaLido junta um arquivo e os que ele inclui (o mesmo namespace): os tipos complexos e os elementos globais.
type esquemaLido struct {
	tipos     map[string]noXSD
	elementos map[string]noXSD
}

func lerEsquema(t *testing.T, arquivo string) esquemaLido {
	t.Helper()
	e := esquemaLido{tipos: map[string]noXSD{}, elementos: map[string]noXSD{}}
	lidos := map[string]bool{}
	var ler func(caminho string)
	ler = func(caminho string) {
		if lidos[caminho] {
			return
		}
		lidos[caminho] = true
		conteudo, err := os.ReadFile(caminho)
		if err != nil {
			t.Fatal(err)
		}
		var schema noXSD
		if err := xml.Unmarshal(conteudo, &schema); err != nil {
			t.Fatalf("%s: %v", caminho, err)
		}
		for _, filho := range schema.Filhos {
			switch filho.XMLName.Local {
			case "include":
				ler(filepath.Join(filepath.Dir(caminho), filho.attr("schemaLocation")))
			case "complexType":
				e.tipos[filho.attr("name")] = filho
			case "element":
				e.elementos[filho.attr("name")] = filho
			}
		}
	}
	ler(filepath.Join(root, "xsd", "esquemas", arquivo))
	return e
}

// caminhosDoEsquema percorre o esquema a partir da raiz e devolve cada caminho de elemento e de atributo (@nome).
func caminhosDoEsquema(t *testing.T, raiz raizDoEsquema) map[string]bool {
	t.Helper()
	caminhos := map[string]bool{}
	documento := lerEsquema(t, raiz.esquema)
	var anys []string
	documento.percorrerElemento(documento.elementos[raiz.elemento], "", caminhos, &anys)
	if len(caminhos) == 0 {
		t.Fatalf("%s: o elemento %s não está no esquema", raiz.esquema, raiz.elemento)
	}
	for _, lugar := range anys {
		if !strings.HasSuffix(lugar, "/infModal") {
			t.Fatalf("%s: xs:any em %s, e o gate só sabe enxertar o infModal", raiz.esquema, lugar)
		}
		for _, modal := range raiz.modais {
			esquema := lerEsquema(t, modal.esquema)
			elemento, ok := esquema.elementos[modal.elemento]
			if !ok {
				t.Fatalf("%s: o elemento %s não está no esquema", modal.esquema, modal.elemento)
			}
			var semAny []string
			esquema.percorrerElemento(elemento, lugar, caminhos, &semAny)
			if len(semAny) > 0 {
				t.Fatalf("%s: xs:any em %v", modal.esquema, semAny)
			}
		}
	}
	return caminhos
}

func (e esquemaLido) percorrerElemento(elemento noXSD, pai string, caminhos map[string]bool, anys *[]string) {
	if ref := elemento.attr("ref"); ref != "" {
		// Só a assinatura vem por referência, de outro namespace: o caminho entra, o conteúdo não.
		caminhos[juntar(pai, local(ref))] = true
		return
	}
	caminho := juntar(pai, elemento.attr("name"))
	caminhos[caminho] = true
	if tipo, ok := e.tipos[local(elemento.attr("type"))]; ok {
		e.percorrerTipo(tipo, caminho, caminhos, anys)
	}
	for _, filho := range elemento.Filhos {
		if filho.XMLName.Local == "complexType" {
			e.percorrerTipo(filho, caminho, caminhos, anys)
		}
	}
}

func (e esquemaLido) percorrerTipo(tipo noXSD, caminho string, caminhos map[string]bool, anys *[]string) {
	for _, filho := range tipo.Filhos {
		switch filho.XMLName.Local {
		case "sequence", "choice":
			e.percorrerTipo(filho, caminho, caminhos, anys)
		case "element":
			e.percorrerElemento(filho, caminho, caminhos, anys)
		case "attribute":
			caminhos[caminho+"/@"+filho.attr("name")] = true
		case "any":
			*anys = append(*anys, caminho)
		case "complexContent", "simpleContent":
			e.percorrerTipo(filho, caminho, caminhos, anys)
		case "extension", "restriction":
			if base, ok := e.tipos[local(filho.attr("base"))]; ok {
				e.percorrerTipo(base, caminho, caminhos, anys)
			}
			e.percorrerTipo(filho, caminho, caminhos, anys)
		}
	}
}

func juntar(pai, nome string) string {
	if pai == "" {
		return nome
	}
	return pai + "/" + nome
}

func local(nome string) string {
	if i := strings.LastIndex(nome, ":"); i >= 0 {
		return nome[i+1:]
	}
	return nome
}

var (
	marshalerXML  = reflect.TypeFor[xml.Marshaler]()
	marshalerText = reflect.TypeFor[encoding.TextMarshaler]()
)

// caminhosDoModelo percorre as tags xml do tipo Go e devolve os caminhos que ele produz, no mesmo formato do esquema.
// Um tipo que se serializa sozinho (Marshaler) ou que não é struct é folha.
func caminhosDoModelo(tipo reflect.Type, caminho string, caminhos map[string]bool, profundidade int) {
	for tipo.Kind() == reflect.Pointer || tipo.Kind() == reflect.Slice {
		tipo = tipo.Elem()
	}
	if tipo.Kind() != reflect.Struct || profundidade > 30 ||
		tipo.Implements(marshalerXML) || reflect.PointerTo(tipo).Implements(marshalerXML) ||
		tipo.Implements(marshalerText) || reflect.PointerTo(tipo).Implements(marshalerText) {
		return
	}
	for i := range tipo.NumField() {
		campo := tipo.Field(i)
		if !campo.IsExported() || campo.Name == "XMLName" {
			continue
		}
		tag := campo.Tag.Get("xml")
		if tag == "-" {
			continue
		}
		nome, opcoes, _ := strings.Cut(tag, ",")
		if i := strings.LastIndex(nome, " "); i >= 0 { // "namespace nome"
			nome = nome[i+1:]
		}
		if campo.Anonymous && nome == "" {
			caminhosDoModelo(campo.Type, caminho, caminhos, profundidade+1)
			continue
		}
		if nome == "" {
			nome = campo.Name
		}
		if slices.Contains(strings.Split(opcoes, ","), "attr") {
			caminhos[caminho+"/@"+nome] = true
			continue
		}
		filho := juntar(caminho, nome)
		caminhos[filho] = true
		caminhosDoModelo(campo.Type, filho, caminhos, profundidade+1)
	}
}
