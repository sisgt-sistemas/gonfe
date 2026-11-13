package cte

import "github.com/sisgt-sistemas/gonfe/tipos"

// Grupos da reforma tributária do consumo (RTC) no leiaute 4.00, pelo esquema da NT 2026.004 RTC. Valem para o CT-e,
// o CT-e OS e o CT-e Simplificado. A biblioteca carrega os valores que quem emite calcula e confere a coerência entre
// eles (Validar); não calcula imposto.
//
// As alíquotas (TDec_0302_04RTC) aceitam de duas a quatro casas e saem com quatro; os valores (TDec1302RTC), com
// duas.

// IBSCBS é o grupo do IBS e da CBS (TTribCTe).
type IBSCBS struct {
	// CST é o código de situação tributária do IBS e da CBS, três dígitos.
	CST string `xml:"CST" norm:"num"`
	// CClassTrib é o código de classificação tributária, seis dígitos.
	CClassTrib string `xml:"cClassTrib" norm:"num"`
	// IndDoacao é "1" quando a operação é doação.
	IndDoacao string `xml:"indDoacao,omitempty"`
	// GIBSCBS traz bases, alíquotas e valores; ausente quando a classificação não tributa.
	GIBSCBS *GIBSCBS `xml:"gIBSCBS,omitempty"`
	// GEstornoCred é o estorno de crédito.
	GEstornoCred *EstornoCred `xml:"gEstornoCred,omitempty"`
}

// GIBSCBS são a base, o IBS da UF e do município e a CBS (TCIBS).
type GIBSCBS struct {
	VBC     tipos.Decimal `xml:"vBC" dec:"2"`
	GIBSUF  IBSUF         `xml:"gIBSUF"`
	GIBSMun IBSMun        `xml:"gIBSMun"`
	// VIBS é a soma do IBS da UF e do município.
	VIBS tipos.Decimal `xml:"vIBS" dec:"2"`
	GCBS CBS           `xml:"gCBS"`
	// GTribRegular é a tributação regular, quando a operação tem tratamento diferenciado.
	GTribRegular *TribRegular `xml:"gTribRegular,omitempty"`
	// GTribCompraGov é a tributação na compra governamental.
	GTribCompraGov *TribCompraGov `xml:"gTribCompraGov,omitempty"`
}

// IBSUF é o IBS da unidade da federação.
type IBSUF struct {
	PIBSUF   tipos.Decimal `xml:"pIBSUF" dec:"4"`
	GDif     *Diferimento  `xml:"gDif,omitempty"`
	GDevTrib *DevTributo   `xml:"gDevTrib,omitempty"`
	GRed     *Reducao      `xml:"gRed,omitempty"`
	VIBSUF   tipos.Decimal `xml:"vIBSUF" dec:"2"`
}

// IBSMun é o IBS do município.
type IBSMun struct {
	PIBSMun  tipos.Decimal `xml:"pIBSMun" dec:"4"`
	GDif     *Diferimento  `xml:"gDif,omitempty"`
	GDevTrib *DevTributo   `xml:"gDevTrib,omitempty"`
	GRed     *Reducao      `xml:"gRed,omitempty"`
	VIBSMun  tipos.Decimal `xml:"vIBSMun" dec:"2"`
}

// CBS é a contribuição sobre bens e serviços.
type CBS struct {
	PCBS     tipos.Decimal `xml:"pCBS" dec:"4"`
	GDif     *Diferimento  `xml:"gDif,omitempty"`
	GDevTrib *DevTributo   `xml:"gDevTrib,omitempty"`
	GRed     *Reducao      `xml:"gRed,omitempty"`
	// GALCZFMCBS é a CBS nas áreas de livre comércio e na Zona Franca de Manaus.
	GALCZFMCBS *ALCZFMCBS    `xml:"gALCZFMCBS,omitempty"`
	VCBS       tipos.Decimal `xml:"vCBS" dec:"2"`
}

// Diferimento é o diferimento do tributo (TDif).
type Diferimento struct {
	PDif tipos.Decimal `xml:"pDif" dec:"4"`
	VDif tipos.Decimal `xml:"vDif" dec:"2"`
}

// DevTributo é a devolução de tributo (TDevTrib).
type DevTributo struct {
	PDevTrib *tipos.Decimal `xml:"pDevTrib,omitempty" dec:"4"`
	VDevTrib tipos.Decimal  `xml:"vDevTrib" dec:"2"`
}

// Reducao é a redução de alíquota (TRed).
type Reducao struct {
	PRedAliq  tipos.Decimal `xml:"pRedAliq" dec:"4"`
	PAliqEfet tipos.Decimal `xml:"pAliqEfet" dec:"4"`
}

// ALCZFMCBS é a CBS nas áreas de livre comércio e na Zona Franca de Manaus (TALCZFMCBS).
type ALCZFMCBS struct {
	PAliqEfetRegCBS tipos.Decimal `xml:"pAliqEfetRegCBS" dec:"4"`
	VTribRegCBS     tipos.Decimal `xml:"vTribRegCBS" dec:"2"`
}

// TribRegular é a tributação regular de referência (TTribRegular).
type TribRegular struct {
	CSTReg             string        `xml:"CSTReg" norm:"num"`
	CClassTribReg      string        `xml:"cClassTribReg" norm:"num"`
	PAliqEfetRegIBSUF  tipos.Decimal `xml:"pAliqEfetRegIBSUF" dec:"4"`
	VTribRegIBSUF      tipos.Decimal `xml:"vTribRegIBSUF" dec:"2"`
	PAliqEfetRegIBSMun tipos.Decimal `xml:"pAliqEfetRegIBSMun" dec:"4"`
	VTribRegIBSMun     tipos.Decimal `xml:"vTribRegIBSMun" dec:"2"`
	PAliqEfetRegCBS    tipos.Decimal `xml:"pAliqEfetRegCBS" dec:"4"`
	VTribRegCBS        tipos.Decimal `xml:"vTribRegCBS" dec:"2"`
}

// TribCompraGov é a tributação na compra governamental (TTribCompraGov).
type TribCompraGov struct {
	PAliqIBSUF  tipos.Decimal `xml:"pAliqIBSUF" dec:"4"`
	VTribIBSUF  tipos.Decimal `xml:"vTribIBSUF" dec:"2"`
	PAliqIBSMun tipos.Decimal `xml:"pAliqIBSMun" dec:"4"`
	VTribIBSMun tipos.Decimal `xml:"vTribIBSMun" dec:"2"`
	PAliqCBS    tipos.Decimal `xml:"pAliqCBS" dec:"4"`
	VTribCBS    tipos.Decimal `xml:"vTribCBS" dec:"2"`
}

// EstornoCred é o estorno de crédito de IBS e CBS (TEstornoCred).
type EstornoCred struct {
	VIBSEstCred tipos.Decimal `xml:"vIBSEstCred" dec:"2"`
	VCBSEstCred tipos.Decimal `xml:"vCBSEstCred" dec:"2"`
}

// CompraGov identifica a compra governamental (TCompraGovReduzido), no grupo ide.
type CompraGov struct {
	// TpEnteGov é o tipo do ente governamental: 1 a 6.
	TpEnteGov string `xml:"tpEnteGov"`
	// PRedutor é o percentual de redução de alíquota da compra governamental.
	PRedutor tipos.Decimal `xml:"pRedutor" dec:"4"`
	// TpOperGov é o tipo da operação com o ente governamental: 1 a 4.
	TpOperGov string `xml:"tpOperGov"`
	// RefDFeAnt são as chaves dos documentos anteriores relacionados, até 99.
	RefDFeAnt []string `xml:"refDFeAnt,omitempty" norm:"alnum"`
}

// PagAntecipado aponta o documento do pagamento antecipado (gPagAntecipado).
type PagAntecipado struct {
	ChDFePagAnt string `xml:"chDFePagAnt" norm:"alnum"`
}

// MultaJuros aponta o CT-e de referência da multa e dos juros (gMultaJuros).
type MultaJuros struct {
	ChCTeRef string `xml:"chCTeRef" norm:"alnum"`
}

// PgtoVinc são os pagamentos vinculados ao documento (pgtoVinc), de 1 a 99.
type PgtoVinc struct {
	Pgto []PagamentoRTC `xml:"pgto"`
}

// PagamentoRTC é um pagamento vinculado (TPagamentoRTC).
type PagamentoRTC struct {
	// NPag é o número sequencial do pagamento, de 1 a 990.
	NPag int `xml:"nPag,attr"`
	// IdTransacao é a identificação da transação no meio de pagamento, até 35 caracteres.
	IdTransacao string `xml:"idTransacao,attr"`
	// TpMeioPgto é o meio de pagamento, dois dígitos.
	TpMeioPgto string `xml:"tpMeioPgto" norm:"num"`
	// CNPJReceb é o CNPJ de quem recebe o pagamento.
	CNPJReceb string `xml:"CNPJReceb" norm:"alnum"`
	// CNPJBasePSP é a raiz do CNPJ do prestador do serviço de pagamento (oito posições).
	CNPJBasePSP string `xml:"CNPJBasePSP" norm:"alnum"`
}

// InfSolicNFF é a solicitação da Nota Fiscal Fácil.
type InfSolicNFF struct {
	XSolic string `xml:"xSolic" norm:"-"`
}

// InfPAA identifica o Provedor de Assinatura e Autorização e a assinatura dele (infPAA).
type InfPAA struct {
	CNPJPAA      string       `xml:"CNPJPAA" norm:"alnum"`
	PAASignature PAASignature `xml:"PAASignature"`
}

// PAASignature é a assinatura RSA do PAA, com a chave pública do emitente, em base64.
type PAASignature struct {
	SignatureValue string      `xml:"SignatureValue" norm:"-"`
	RSAKeyValue    RSAKeyValue `xml:"RSAKeyValue"`
}

// RSAKeyValue é a chave pública RSA (TRSAKeyValueType), em base64.
type RSAKeyValue struct {
	Modulus  string `xml:"Modulus" norm:"-"`
	Exponent string `xml:"Exponent" norm:"-"`
}

// IBSCBSTot é o total de IBS e CBS do CT-e Simplificado (TIBSCBSTot), soma dos itens.
type IBSCBSTot struct {
	VBCIBSCBS    tipos.Decimal `xml:"vBCIBSCBS" dec:"2"`
	GIBS         IBSTot        `xml:"gIBS"`
	GCBS         TotalTributo  `xml:"gCBS"`
	GEstornoCred *EstornoCred  `xml:"gEstornoCred,omitempty"`
}

// IBSTot é o total do IBS, da UF e do município.
type IBSTot struct {
	GIBSUF  TotalIBSUF    `xml:"gIBSUF"`
	GIBSMun TotalIBSMun   `xml:"gIBSMun"`
	VIBS    tipos.Decimal `xml:"vIBS" dec:"2"`
}

// TotalIBSUF é o total do IBS da UF: diferido, devolvido e devido.
type TotalIBSUF struct {
	VDif     tipos.Decimal `xml:"vDif" dec:"2"`
	VDevTrib tipos.Decimal `xml:"vDevTrib" dec:"2"`
	VIBSUF   tipos.Decimal `xml:"vIBSUF" dec:"2"`
}

// TotalIBSMun é o total do IBS do município.
type TotalIBSMun struct {
	VDif     tipos.Decimal `xml:"vDif" dec:"2"`
	VDevTrib tipos.Decimal `xml:"vDevTrib" dec:"2"`
	VIBSMun  tipos.Decimal `xml:"vIBSMun" dec:"2"`
}

// TotalTributo é o total da CBS.
type TotalTributo struct {
	VDif     tipos.Decimal `xml:"vDif" dec:"2"`
	VDevTrib tipos.Decimal `xml:"vDevTrib" dec:"2"`
	VCBS     tipos.Decimal `xml:"vCBS" dec:"2"`
}
