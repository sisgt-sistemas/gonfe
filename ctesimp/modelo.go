// Package ctesimp implementa o CT-e Simplificado, modelo 57, no leiaute 4.00.
//
// O CT-e Simplificado é o conhecimento que reúne, num documento só, várias
// prestações de um mesmo tomador: cada entrega é um item (det), com o seu
// trecho, o seu valor e os documentos que ela transporta. Por isso ele tem raiz
// própria, <CTeSimp>, sem remetente, expedidor, recebedor e destinatário: o
// tomador é declarado uma vez, e a carga é descrita no conjunto.
//
// # O que é compartilhado com o CT-e
//
// Do pacote [github.com/sisgt-sistemas/gonfe/cte] vêm o endereço, o grupo de
// impostos (ICMS e partilha), a carga, o modal rodoviário, a cobrança, os
// componentes do valor, o responsável técnico e o protocolo. A chave de acesso,
// a assinatura e o formato da mensagem de envio (gzip e base64) também são os
// mesmos, e os eventos (cancelamento, carta de correção) são os do pacote cte:
// a raiz é `eventoCTe`, e o que muda é só a chave referenciada. O serviço de
// recepção, não: o CT-e Simplificado tem o seu, com retorno próprio
// (retCTeSimp), que o pacote sefaz ainda não implementa.
//
// # O que fica de fora
//
// Os grupos e campos da reforma tributária (IBSCBS por item e IBSCBSTot,
// vPrestLiq, vTPrestLiq, vTotDFe, tpPagAnt, gCompraGov, pgtoVinc,
// gPagAntecipado, gMultaJuros, infPAA, infSolicNFF) ainda não são
// implementados, como no CT-e e no CT-e OS deste módulo. O esquema aceita os
// modais rodoviário, aéreo e aquaviário; este pacote implementa o rodoviário.
// O tomador no exterior (CNPJ vazio, UF "EX" no endereço) também fica para
// depois: o Validar exige CNPJ ou CPF.
package ctesimp

import (
	"encoding/xml"

	"github.com/sisgt-sistemas/gonfe/cte"
	"github.com/sisgt-sistemas/gonfe/tipos"
)

// Versao é a versão do leiaute implementada por este pacote.
const Versao = cte.Versao

// VersaoModal é a versão do grupo específico do modal.
const VersaoModal = cte.VersaoModal

// Espaco é o namespace XML, o mesmo do CT-e.
const Espaco = cte.Espaco

// TipoCTe é a finalidade do CT-e Simplificado (TFinCTeSimp). O modelo 57 comum
// usa os códigos de 0 a 3; o simplificado tem os seus.
type TipoCTe string

const (
	// Simplificado é o CT-e Simplificado normal.
	Simplificado TipoCTe = "5"
	// SubstituicaoSimplificado substitui um CT-e Simplificado, apontado em infCteSub.
	SubstituicaoSimplificado TipoCTe = "6"
)

// TipoUnidadeCarga é o código do tipo da unidade de carga (TtipoUnidCarga: 1 a 4). O esquema lista os códigos
// sem descrevê-los; o significado está no Manual de Orientação.
type TipoUnidadeCarga string

// TipoUnidadeTransporte é o código do tipo da unidade de transporte (TtipoUnidTransp: 1 a 7), como no MDF-e.
type TipoUnidadeTransporte string

// TipoPrestacao diz se a prestação anterior cobre todas as notas do documento anterior ou parte delas.
type TipoPrestacao string

const (
	PrestacaoTotal   TipoPrestacao = "1"
	PrestacaoParcial TipoPrestacao = "2"
)

// CTeSimp é o documento completo, correspondente ao elemento raiz <CTeSimp>.
type CTeSimp struct {
	XMLName xml.Name `xml:"http://www.portalfiscal.inf.br/cte CTeSimp"`

	// InfCte são as informações do conhecimento, o bloco assinado.
	InfCte InfCte `xml:"infCte"`
	// InfCTeSupl traz o QR Code do DACTE.
	InfCTeSupl *cte.InfCTeSupl `xml:"infCTeSupl,omitempty"`
}

// InfCte são as informações do conhecimento, na ordem do esquema.
type InfCte struct {
	Versao string `xml:"versao,attr"`
	Id     string `xml:"Id,attr"`

	Ide   Ide    `xml:"ide"`
	Compl *Compl `xml:"compl,omitempty"`
	Emit  Emit   `xml:"emit"`
	Toma  Toma   `xml:"toma"`
	// InfCarga descreve a carga do conjunto das prestações; o vCarga é obrigatório aqui.
	InfCarga cte.InfCarga `xml:"infCarga"`
	// Det são as prestações, de 1 a 999.
	Det        []Det           `xml:"det"`
	InfModal   cte.InfModal    `xml:"infModal"`
	Cobr       *cte.Cobr       `xml:"cobr,omitempty"`
	InfCteSub  *InfCteSub      `xml:"infCteSub,omitempty"`
	Imp        cte.Imp         `xml:"imp"`
	Total      Total           `xml:"total"`
	AutXML     []cte.AutXML    `xml:"autXML,omitempty"`
	InfRespTec *cte.InfRespTec `xml:"infRespTec,omitempty"`
}

// Ide é o grupo de identificação do conhecimento.
type Ide struct {
	// CUF é o código do IBGE da UF do emitente.
	CUF int `xml:"cUF"`
	// CCT é o código numérico de oito dígitos que compõe a chave de acesso.
	CCT string `xml:"cCT" norm:"num"`
	// CFOP é o código fiscal da prestação.
	CFOP string `xml:"CFOP" norm:"num"`
	// NatOp é a natureza da operação.
	NatOp string `xml:"natOp"`
	// Mod é o modelo do documento: 57, como o CT-e.
	Mod cte.Modelo `xml:"mod"`
	// Serie é a série do documento.
	Serie int `xml:"serie"`
	// NCT é o número do conhecimento.
	NCT int `xml:"nCT"`
	// DhEmi é a data e hora de emissão, no fuso da UF do emitente.
	DhEmi tipos.DataHora `xml:"dhEmi"`
	// TpImp é o formato de impressão do DACTE.
	TpImp cte.FormatoImpressao `xml:"tpImp"`
	// TpEmis é a forma de emissão.
	TpEmis cte.TipoEmissao `xml:"tpEmis"`
	// CDV é o dígito verificador da chave de acesso.
	CDV int `xml:"cDV"`
	// TpAmb distingue produção de homologação.
	TpAmb cte.Ambiente `xml:"tpAmb"`
	// TpCTe é a finalidade: 5 (simplificado) ou 6 (substituição).
	TpCTe TipoCTe `xml:"tpCTe"`
	// ProcEmi identifica o processo de emissão.
	ProcEmi cte.ProcessoEmissao `xml:"procEmi"`
	// VerProc é a versão do aplicativo emissor.
	VerProc string `xml:"verProc"`
	// CMunEnv é o código do IBGE do município de envio.
	CMunEnv int `xml:"cMunEnv"`
	// XMunEnv é o nome do município de envio.
	XMunEnv string `xml:"xMunEnv"`
	// UFEnv é a UF de envio.
	UFEnv string `xml:"UFEnv" norm:"upper"`
	// Modal é o meio de transporte.
	Modal cte.Modal `xml:"modal"`
	// TpServ é o tipo do serviço: 0 normal, 1 subcontratação, 2 redespacho.
	TpServ cte.TipoServico `xml:"tpServ"`
	// UFIni é a UF de início das prestações; os municípios ficam em cada det.
	UFIni string `xml:"UFIni" norm:"upper"`
	// UFFim é a UF de término.
	UFFim string `xml:"UFFim" norm:"upper"`
	// Retira diz se o recebedor retira a carga no destino.
	Retira cte.Retira `xml:"retira"`
	// XDetRetira detalha a retirada.
	XDetRetira string `xml:"xDetRetira,omitempty"`
	// DhCont é a data e hora de entrada em contingência.
	DhCont *tipos.DataHora `xml:"dhCont,omitempty"`
	// XJust é a justificativa da contingência.
	XJust string `xml:"xJust,omitempty"`
}

// Compl são as informações complementares. O grupo é mais curto que o do CT-e:
// sem emissor, previsão de entrega nem municípios de cálculo.
type Compl struct {
	XCaracAd  string         `xml:"xCaracAd,omitempty"`
	XCaracSer string         `xml:"xCaracSer,omitempty"`
	Fluxo     *cte.Fluxo     `xml:"fluxo,omitempty"`
	XObs      string         `xml:"xObs,omitempty"`
	ObsCont   []cte.ObsCampo `xml:"ObsCont,omitempty"`
	ObsFisco  []cte.ObsCampo `xml:"ObsFisco,omitempty"`
}

// Emit é o emitente — a transportadora. Difere do emitente do CT-e pela
// inscrição estadual opcional e pela inscrição na SUFRAMA.
type Emit struct {
	CNPJ      string               `xml:"CNPJ,omitempty" norm:"alnum"`
	CPF       string               `xml:"CPF,omitempty" norm:"num"`
	IE        string               `xml:"IE,omitempty" norm:"upper"`
	IEST      string               `xml:"IEST,omitempty" norm:"upper"`
	XNome     string               `xml:"xNome"`
	XFant     string               `xml:"xFant,omitempty"`
	EnderEmit cte.EnderecoEmitente `xml:"enderEmit"`
	// CRT é o regime tributário do emitente (TCRT): 1 a 4.
	CRT string `xml:"CRT"`
	// ISUFEmit é a inscrição do emitente na SUFRAMA.
	ISUFEmit string `xml:"ISUFEmit,omitempty" norm:"num"`
}

// Toma é o tomador de todas as prestações do documento.
type Toma struct {
	// Toma é o papel do tomador na prestação: 0 remetente a 4 outros.
	Toma cte.Tomador `xml:"toma"`
	// IndIEToma indica a situação do tomador quanto à inscrição estadual.
	IndIEToma cte.IndicadorIE `xml:"indIEToma"`
	CNPJ      string          `xml:"CNPJ,omitempty" norm:"alnum"`
	CPF       string          `xml:"CPF,omitempty" norm:"num"`
	IE        string          `xml:"IE,omitempty" norm:"upper"`
	XNome     string          `xml:"xNome"`
	// ISUF é a inscrição na SUFRAMA.
	ISUF      string       `xml:"ISUF,omitempty" norm:"num"`
	Fone      string       `xml:"fone,omitempty" norm:"num"`
	EnderToma cte.Endereco `xml:"enderToma"`
	Email     string       `xml:"email,omitempty"`
}

// Det é uma prestação do documento: um trecho, o seu valor e o que ele leva.
type Det struct {
	// NItem é o número do item, de 1 em diante; o [CTeSimp.Preparar] numera os que vierem zerados.
	NItem int `xml:"nItem,attr"`

	CMunIni int    `xml:"cMunIni"`
	XMunIni string `xml:"xMunIni"`
	CMunFim int    `xml:"cMunFim"`
	XMunFim string `xml:"xMunFim"`
	// VPrest é o valor da prestação; com componentes, o Preparar o soma deles.
	VPrest tipos.Decimal `xml:"vPrest" dec:"2"`
	// VRec é o valor a receber; zerado, o Preparar o iguala ao vPrest. Preenchido, fica como está: mudou os
	// componentes depois de um Preparar, zere o vRec para recalcular.
	VRec tipos.Decimal    `xml:"vRec" dec:"2"`
	Comp []cte.Componente `xml:"Comp,omitempty"`
	// InfNFe são as NF-e transportadas. Preencha este ou InfDocAnt, nunca os dois.
	InfNFe []InfNFe `xml:"infNFe,omitempty"`
	// InfDocAnt são os CT-e anteriores, na subcontratação e no redespacho.
	InfDocAnt []InfDocAnt `xml:"infDocAnt,omitempty"`
}

// InfNFe é uma NF-e transportada na prestação.
type InfNFe struct {
	ChNFe string      `xml:"chNFe" norm:"alnum"`
	PIN   string      `xml:"PIN,omitempty" norm:"num"`
	DPrev *tipos.Data `xml:"dPrev,omitempty"`
	// InfUnidCarga e InfUnidTransp são alternativos: informe um dos dois, ou nenhum.
	InfUnidCarga  []UnidadeCarga      `xml:"infUnidCarga,omitempty"`
	InfUnidTransp []UnidadeTransporte `xml:"infUnidTransp,omitempty"`
}

// UnidadeCarga é uma unidade de carga (TUnidCarga): contêiner, ULD, pallet.
type UnidadeCarga struct {
	TpUnidCarga  TipoUnidadeCarga `xml:"tpUnidCarga"`
	IdUnidCarga  string           `xml:"idUnidCarga" norm:"upper"`
	LacUnidCarga []Lacre          `xml:"lacUnidCarga,omitempty"`
	// QtdRat é a quantidade rateada (TDec_0302_0303: até três casas).
	QtdRat *tipos.Decimal `xml:"qtdRat,omitempty" dec:"3"`
}

// UnidadeTransporte é uma unidade de transporte (TUnidadeTransp): o veículo ou o reboque, com as unidades de carga.
type UnidadeTransporte struct {
	TpUnidTransp  TipoUnidadeTransporte `xml:"tpUnidTransp"`
	IdUnidTransp  string                `xml:"idUnidTransp" norm:"upper"`
	LacUnidTransp []Lacre               `xml:"lacUnidTransp,omitempty"`
	InfUnidCarga  []UnidadeCarga        `xml:"infUnidCarga,omitempty"`
	QtdRat        *tipos.Decimal        `xml:"qtdRat,omitempty" dec:"3"`
}

// Lacre é o número de um lacre.
type Lacre struct {
	NLacre string `xml:"nLacre"`
}

// InfDocAnt é um CT-e anterior, na subcontratação ou no redespacho.
type InfDocAnt struct {
	ChCTe   string        `xml:"chCTe" norm:"alnum"`
	TpPrest TipoPrestacao `xml:"tpPrest"`
	// InfNFeTranspParcial lista as NF-e transportadas, quando a prestação é parcial.
	InfNFeTranspParcial []NFeParcial `xml:"infNFeTranspParcial,omitempty"`
}

// NFeParcial é uma NF-e do documento anterior transportada nesta prestação.
type NFeParcial struct {
	ChNFe string `xml:"chNFe" norm:"alnum"`
}

// InfCteSub aponta o CT-e Simplificado substituído.
type InfCteSub struct {
	ChCte string `xml:"chCte" norm:"alnum"`
	// IndAlteraToma é "1" quando a substituição troca o tomador.
	IndAlteraToma string `xml:"indAlteraToma,omitempty"`
}

// Total são os valores somados das prestações.
type Total struct {
	VTPrest tipos.Decimal `xml:"vTPrest" dec:"2"`
	VTRec   tipos.Decimal `xml:"vTRec" dec:"2"`
}
