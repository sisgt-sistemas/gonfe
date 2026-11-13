# Registro de mudanças

Segue o formato do [Keep a Changelog](https://keepachangelog.com/pt-BR/1.1.0/) e
o [versionamento semântico](https://semver.org/lang/pt-BR/).

## [Não lançado]

## [1.1.0] — 2026-11-13

### Adicionado

- DACTE em meia folha (`Opcoes.Papel: danfe.PapelA5`): A5 paisagem, que o
  Manual do DACTE admite (§2.15), com os quadros do A4 mais baixos e o código de
  barras e o QR Code no tamanho mínimo. Sai sem canhoto, a menos que
  `Opcoes.CanhotoNaMeiaFolha` o peça; o que não cabe vai para a folha seguinte,
  que repete o cabeçalho. `make amostras` grava `dacte-meia-folha.pdf`.
- Duas meias folhas por A4 (`danfe.GerarDACTEsEmA4` e `danfe.DACTEsEmA4`): os
  DACTEs em meia folha um em cima do outro, em tamanho real, com borda em cada um
  e a linha de corte no meio. `make amostras` grava
  `dacte-meia-folha-duas-por-a4.pdf`.

### Corrigido

- O código de barras da chave no DACTE, no DACTE OS e no do Simplificado fica
  centrado na faixa: a conta da posição era feita em inteiros e o deixava
  0,5 mm acima do centro.

## [1.0.2] — 2026-11-13

### Alterado

- O CI fica mais leve: roda só no push em `develop` (o PR de release e o `main`
  recebem o mesmo commit), com 4 jobs em vez de 10. Saem o macOS, que ficava na
  fila sem runner, e os jobs que repetiam o `make ci`; corrida, cobertura e
  `govulncheck` dividem um job.

## [1.0.1] — 2026-11-13

### Alterado

- A documentação não é mais publicada no GitHub Pages: os guias ficam em
  `docs/` no repositório, e `mkdocs serve` continua servindo-os localmente.

### Corrigido

- `make release` podia recusar uma seção "Não lançado" cheia como vazia: com
  `pipefail`, o `grep -q` saía no primeiro casamento e o `awk` tomava SIGPIPE.
  A falha era intermitente e derrubou o gate do CI na v1.0.0.
- Os testes dos scripts do fluxo (`internal/flowtest`) pulam no Windows, onde
  os scripts bash não rodam.

## [1.0.0] — 2026-11-13

Fork mantido pela SISGT Sistemas (`github.com/sisgt-sistemas/gonfe`).

### Adicionado

- Tributos federais do CT-e OS (`cteos.InfTribFed`: PIS, COFINS, IR, INSS e
  CSLL), que o esquema prevê no `imp`, conferidos no `Validar` e impressos no
  DACTE OS.
- DACTE do CT-e Simplificado (`danfe.GerarDACTESimp`, `danfe.DACTESimp`): o
  cabeçalho do Manual do DACTE e a tabela das entregas com os documentos de
  cada uma, paginada.
- QR Code no DACTE, no DACTE OS e no DAMDFE, a partir do `qrCodCTe` e do
  `qrCodMDFe`, codificado com o `rsc.io/qr` (Go puro, sem CGO).
- `make amostras` grava os PDFs de exemplo em `tmp/amostras`.
- `cteos.Ide.InfPercurso`: as UF do percurso do veículo, que o esquema do CT-e
  OS prevê.
- Reforma tributária no CT-e, no CT-e OS e no CT-e Simplificado (NT 2026.004
  RTC), com os tipos no pacote `cte`: `IBSCBS` (IBS da UF e do município e CBS,
  com diferimento, devolução, redução, tributação regular, compra governamental
  e estorno de crédito), `vTotDFe`, `vTPrestLiq` e `vPrestLiq`, compra
  governamental, pagamento antecipado, multa e juros, pagamentos vinculados,
  Nota Fiscal Fácil e Provedor de Assinatura e Autorização, onde o esquema de
  cada documento os prevê. A biblioteca não calcula o imposto: `Validar`
  confere a coerência e os padrões, e o `Preparar` do Simplificado soma o
  `IBSCBSTot` das prestações.
- `cte.TipoEmissao.Valida` e `cte.EmissaoRegimeEspecialNFF` (3).
- CNPJ alfanumérico (Nota Técnica Conjunta 2025.001) em toda a biblioteca:
  `chave` monta, valida e decompõe a chave com letras nas posições do CNPJ, com
  o dígito verificador pelo ASCII menos 48; a regra de normalização
  `norm:"alnum"` guarda as letras em todo campo de CNPJ e de chave (NF-e, CT-e,
  CT-e OS, CT-e Simplificado, MDF-e e eventos); o certificado, a distribuição
  de DF-e e a consulta de MDF-e não encerrados aceitam o CNPJ com letras; o
  código de barras do DANFE, do DACTE e do DAMDFE alterna do Code 128 C para o
  A nas letras (`pdf.ValoresCode128`). Uma chave só de dígitos sai igual.
- `chave.CNPJValido`, para o padrão do CNPJ nos leiautes.
- Pacote `ctesimp`: CT-e Simplificado (modelo 57, raiz `CTeSimp`), com várias
  prestações de um tomador por documento. Modelo na ordem do esquema,
  `Preparar` (numera os itens e soma componentes e totais), `Validar()`,
  assinatura, leitura e montagem do `cteSimpProc` e envio síncrono. Modal
  rodoviário. Os exemplos
  são validados contra o esquema oficial nos testes.
- `cte.EmissaoContingenciaOffline` (tpEmis 2), que o leiaute 4.00 aceita.
- Transmissão do CT-e Simplificado: `sefaz.ClienteCTe.Autorizar` reconhece o
  `<CTeSimp>` e o envia à recepção própria (`CTeRecepcaoSimpV4`, método
  `cteRecepcaoSimp`, NT 2024.002), com os endereços de produção e homologação
  de todos os autorizadores conferidos no Portal do CT-e; a resposta
  (`sefaz.RetCTeSimp`) volta como `RetCTe`, com o protocolo. Antes, o
  documento era montado como CT-e e recusado sem explicação.
- Pacote `xsd`: valida CT-e, CT-e Simplificado e MDF-e contra os esquemas
  oficiais (CT-e 4.00 da NT 2026.004 RTC v1.01; MDF-e 3.00b), embutidos, pelo
  `xmllint`. É ferramenta de teste e não usa CGO; sem `xmllint`, devolve
  `xsd.ErrSemXmllint`. Os exemplos de CT-e e de MDF-e passam a ser validados
  contra o esquema nos testes.

### Alterado

- **Incompatível:** o grupo de impostos do CT-e OS passa a ser `cteos.Imp`, e
  não mais `cte.Imp`, para levar o `infTribFed`; os campos comuns têm os mesmos
  nomes.
- DACTE, DACTE OS e DAMDFE refeitos pelos manuais oficiais (MOC CT-e 4.00 e
  MDF-e 3.00b, Anexo II): código de barras de 15 mm com margem clara, consulta
  impressa, telefone do emitente, país e telefone das partes, expedidor e
  recebedor, CT-e globalizado, fluxo da carga, documentos originários em duas
  colunas, percurso do veículo no DACTE OS; o DAMDFE no leiaute do manual, com
  vale-pedágio, RNTRC por veículo e o aviso de contingência no lugar do
  protocolo. A homologação sai como texto na área variável, sem tarja por cima
  dos valores, e a paginação mede o espaço, sem deixar meia folha vazia.
- **Incompatível:** `Opcoes.Mensagem` e o lembrete de encerramento não saem
  mais no DACTE, no DACTE OS e no DAMDFE: os manuais só admitem o que está no
  XML. A coluna "ICMS ST" do DACTE, que trazia a partilha, saiu; a partilha vai
  para as informações adicionais.
- **Incompatível:** DACTE, DACTE OS, DACTE do Simplificado e DAMDFE saem sempre
  em A4 retrato (`Opcoes.Orientacao` é ignorada neles: em paisagem os quadros
  fixos não cabem). Quando os quadros fixos não cabem nem em retrato
  (componentes ou quantidades demais), a geração devolve erro com
  `danfe.ErrNaoCabe`, em vez de cortar o conteúdo.
- **Incompatível:** `cte.EmissaoContingencia` (5) sai, porque o leiaute 4.00
  não tem o código, e `cte.EmissaoFSDA` vira `cte.EmissaoRegimeEspecialNFF`: no
  leiaute atual o 3 é o Regime Especial da Nota Fiscal Fácil. O `Validar` recusa
  a forma de emissão fora do leiaute de cada documento: 1, 2, 3, 4, 7 e 8 no
  CT-e e no Simplificado; 1, 2, 7 e 8 no CT-e OS. O 3 deixa de contar como
  contingência (não pede data nem justificativa).
- **Incompatível:** `cteos.InfSolicNFF` e o campo `InfCte.InfSolicNFF` do CT-e
  OS saem: o esquema do CT-e OS não tem o grupo, e o documento que o levava era
  recusado.
- Esquemas do MDF-e atualizados para o pacote "Schemas NT 2025.001 v 1.04"
  (25/04/2026): o processo de emissão aceita o 4, emissão por Provedor de
  Assinatura e Autorização (`mdfe.EmissaoPAA`), e o CNPJ do contratante deixa
  de aceitar vazio. O `Id` do manifesto continua só numérico, então o MDF-e de
  emitente com CNPJ alfanumérico segue recusado pelo esquema.
- **Incompatível:** `chave.Limpar` preserva as letras (em maiúsculas) e tira
  só a pontuação, o prefixo do atributo Id (`NFe`, `CTe`, `MDFe`) e um rótulo
  de letras antes da chave (`Chave: …`); antes devolvia só os dígitos. Outro
  texto misturado à chave agora a invalida. `chave.Validar` aceita a chave com
  CNPJ alfanumérico.
- **Incompatível:** o `Preparar` dos documentos deixa de apagar as letras dos
  campos de CNPJ e de chave; ele as põe em maiúsculas.
- `sefaz.ClienteMDFe.NaoEncerrados` aceita o CNPJ alfanumérico, que o esquema
  da consulta já aceitava.
- O caminho do módulo passa a ser `github.com/sisgt-sistemas/gonfe`.
- **Incompatível:** `cte.Emit.EnderEmit` passa a ser `cte.EnderecoEmitente`
  (`TEndeEmi`: sem `cPais` e `xPais`, com `fone`), como o esquema pede; vale
  também para o CT-e OS.
- **Incompatível:** o tipo do proprietário do CT-e OS (`tpProp`) segue o leiaute:
  `ProprietarioTACAgregado` (0), `ProprietarioTACIndependente` (1) e
  `ProprietarioOutros` (2). Antes os códigos tinham o significado trocado e
  havia um 3, que o esquema recusa.
- **Incompatível:** o veículo do CT-e OS perde o `CInt`, que não existe no
  esquema do modal rodoviário de outros serviços.

### Corrigido

- `validacao.ValidarCPFouCNPJ` decidia pelo número de dígitos: um CNPJ
  alfanumérico com três letras (11 dígitos) caía na validação de CPF. Agora
  entrada com letra vai para o CNPJ, e `ValidarCPF` recusa letra.
- O `CRT` do emitente do CT-e e do CT-e OS é obrigatório e conferido em
  `Validar()` (TCRT: 1 a 4); antes saía omitido e o esquema recusava.
- As alíquotas do ICMS do CT-e (`pICMS`, `pRedBC`, `pICMSSTRet`,
  `pICMSOutraUF`, `pRedBCOutraUF`, `pICMSUFFim`, `pICMSInter`, `pFCPUFFim`) saem
  com duas casas (`TDec_0302`); com quatro, o esquema recusava.
- CT-e OS: a raiz leva a `versao`, o `indIEToma` vem antes do `cMunIni` e o
  veículo segue a ordem do esquema (placa, RENAVAM, prop, UF). O exemplo passa
  a ser validado contra o esquema nos testes.
- A tarifa do modal dutoviário do CT-e (`vTar`) sai com seis casas
  (`TDec_0906Opc`); com quatro, o esquema recusava.
- `Validar()` confere o nome do componente (até 15) no CT-e e no CT-e OS e a
  descrição do serviço do CT-e OS (até 30).
- Ler um `cteProc`, `mdfeProc` ou evento com o protocolo aberto e não fechado
  (arquivo cortado) devolve erro, e não mais o documento como "ainda não
  autorizado".

## [0.5.0] — 2026-08-17

Fecha os ciclos de vida. Todo documento que a biblioteca emite agora pode
também ser corrigido, cancelado e impresso.

### Adicionado

- Eventos do CT-e, que faltavam: um conhecimento era autorizado mas nunca podia
  ser cancelado. `cte.NovoCancelamento` (110111), `cte.NovaCartaCorrecao`
  (110110) e `cte.NovoDesacordo` (610110), com `sefaz.ClienteCTe.EnviarEvento`.
- A carta de correção do CT-e é mais estrita que a da NF-e: em vez de texto
  livre, cada correção aponta grupo, campo e valor. O `xCondUso`, que a SEFAZ
  compara caractere a caractere, é preenchido a partir de
  `cte.CondicaoDeUsoCCe`.
- O registro de prestação em desacordo é do **tomador**, não do emitente — é a
  via que o contratante tem para dizer que o serviço divergiu do contratado.
- DACTE OS em PDF: `danfe.GerarDACTEOS` e `danfe.DACTEOS`. Onde o DACTE descreve
  carga e documentos transportados, ele descreve o tomador, o serviço em texto
  livre, o veículo e os documentos referenciados — bilhetes de passagem e GTV-e
  na mesma tabela. Não tem canhoto: não há volumes a receber.
- O comando `exemplos/danfe` reconhece o CT-e OS pelo elemento raiz, testando o
  modelo 67 antes do 57 porque `<CTe` é prefixo de `<CTeOS`.
- Teste fixando que os eventos do pacote `cte` servem ao modelo 67: o elemento
  raiz é `eventoCTe` nos dois, então não há — nem deve haver — um pacote de
  eventos do CT-e OS.
- O modo `-amostra` do `exemplos/danfe` passa a gerar os cinco documentos
  auxiliares.

### Notas

- Os documentos auxiliares do transporte seguem a estrutura de blocos dos
  manuais, não uma reprodução milimétrica do formulário, e nenhum passou por
  homologação visual em SEFAZ. Imprima uma amostra e confira.

## [0.4.0] — 2026-08-17

Fecha o transporte: o CT-e OS, os documentos auxiliares que faltavam e a ponta
que transmite o MDF-e.

### Adicionado

- Pacote `cteos`: CT-e Outros Serviços, modelo 67, no leiaute 4.00 — transporte
  de pessoas, transporte de valores e excesso de bagagem. Raiz `<CTeOS>`, um
  único grupo `toma` no lugar do par toma3/toma4, serviço descrito em texto no
  lugar da carga e o modal `rodoOS`, que traz o veículo de volta ao documento
  porque nessas prestações não há MDF-e para carregá-lo.
- Tudo o que os dois modelos de CT-e têm em comum vem do pacote `cte` e não é
  redefinido: grupos de ICMS, endereço, emitente, valor da prestação,
  informações complementares, cobrança, responsável técnico e protocolo.
- `sefaz.ClienteCTe.Autorizar` reconhece o modelo pelo elemento raiz e atende
  aos dois pelo mesmo serviço de recepção síncrona.
- `sefaz.ClienteMDFe`, que faltava: o manifesto era montado, validado e
  assinado, mas não havia como transmiti-lo. Cobre status do serviço,
  autorização síncrona, consulta por chave, recepção de evento e a consulta de
  manifestos não encerrados. O MDF-e é centralizado — todas as UFs vão para a
  Sefaz Virtual do Rio Grande do Sul.
- `RetConsSitMDFe.Encerrado` e `RetEventoMDFe.Registrado`, que já levam em conta
  o código 132: o encerramento é aceito com um `cStat` diferente do 135 dos
  demais eventos, e conferir só o 135 daria o encerramento como recusado.
- `RetConsMDFeNaoEnc.Chaves` lista os manifestos em aberto. A rejeição que a
  SEFAZ devolve na autorização não diz qual manifesto está pendente; esta
  consulta diz.
- `mdfe.MontarEnvioSincrono`, a compressão gzip e base64 da recepção do 3.00.
- DACTE do CT-e e DAMDFE do MDF-e em PDF, no mesmo pacote `danfe` e com a mesma
  paginação automática dos demais: `GerarDACTE`, `DACTE`, `GerarDAMDFE` e
  `DAMDFE`.
- No DACTE, o tomador apontado pelo `toma3` é resolvido para a parte
  correspondente, e o CNPJ, a série e o número dos documentos originários saem
  da própria chave de acesso.
- O comando `exemplos/danfe` reconhece o tipo do documento pelo elemento raiz do
  XML, então o mesmo `-xml` atende a NF-e, CT-e e MDF-e. O modo `-amostra` passa
  a gerar os quatro documentos auxiliares.
- Teste que confere que nada é desenhado abaixo da borda inferior da página, em
  todos os documentos auxiliares e nos casos que enchem a primeira folha. As
  constantes de espaço reservado que dividem os itens entre as páginas são uma
  estimativa; sem essa contraprova, uma estimativa otimista faria o rodapé
  escorregar para fora da folha em silêncio.

### Alterado

- As actions do CI e da documentação foram atualizadas; as versões anteriores
  ainda pediam Node 20, que o runner agora força para Node 24.
- A página de pacotes da documentação estava parada na 0.1.0: o mapa não
  conhecia `cte`, `mdfe`, `danfe`, `dfe` nem `evento`. Foi refeito.

### Notas

- O `cteos` é novo e tem menos rodagem em campo que o `cte`. Homologue antes de
  emitir com valor fiscal.
- O DACTE OS em PDF não está implementado.
- As tabelas de endereço de CT-e e MDF-e continuam sendo reproduções do que os
  portais publicam; confira antes de produção e sobreponha o que divergir.

## [0.3.0] — 2026-08-17

Quatro frentes novas: distribuição de DF-e, documentos auxiliares em PDF, CT-e e
MDF-e.

### Adicionado

**Distribuição de DF-e**

- Pacote `dfe`: consulta por último NSU, por NSU específico e por chave;
  descompactação dos `docZip`; interpretação dos quatro schemas que a fila
  entrega — resumo de NF-e, NF-e completa, resumo de evento e evento completo.
- `Cliente.DistribuicaoDFe` e `Cliente.ConsumirDFe`. O laço respeita um minuto
  entre consultas, porque a Receita bloqueia o CNPJ por uma hora quando o
  consumo é considerado indevido, e devolve o NSU do último documento
  processado com sucesso para que a retomada não pule nada.
- O envelope SOAP da distribuição tem um nível a mais que o dos demais
  serviços: o `nfeDadosMsg` vai dentro de um `nfeDistDFeInteresse`.

**DANFE e cupom em PDF**

- Pacote `danfe`: DANFE da NF-e em A4 com os blocos do manual e paginação
  automática, e cupom da NFC-e em bobina com altura calculada pelo conteúdo.
- Escritor PDF próprio, em Go puro, com as fontes base-14 e as métricas reais
  da Helvetica. Sem biblioteca gráfica, sem CGO, sem dependência nova.
- Code 128 modo C para o código de barras da chave de acesso.
- O QR Code não é codificado pela biblioteca: passe a matriz pronta em
  `Opcoes.QRCode`. Sem ela, o cupom sai com a URL em texto e avisa.

**CT-e**

- Pacote `cte`: modelo 57 no leiaute 4.00, com o modal rodoviário completo e as
  estruturas dos modais aéreo, aquaviário, ferroviário, dutoviário e
  multimodal.
- `sefaz.ClienteCTe` com tabela de endereços própria e a recepção síncrona do
  4.00, que recebe o documento comprimido em gzip.

**MDF-e**

- Pacote `mdfe`: modelo 58 no leiaute 3.00, com o modal rodoviário completo —
  veículo de tração, reboques, condutores e dados da ANTT.
- Documentos agrupados por município de descarregamento, com contagem
  automática de NF-e, CT-e e MDF-e no grupo de totais.
- Eventos de encerramento de viagem, cancelamento e inclusão de condutor.

**Outros**

- Exemplos executáveis `exemplos/danfe`, com modo `-amostra` que gera os dois
  documentos auxiliares sem precisar de certificado nem de XML.
- Guias de distribuição de DF-e, DANFE e transporte na documentação.

### Notas

- O CT-e OS (modelo 67) tem raiz e estrutura próprias e não está implementado.
- O DACTE e o DAMDFE em PDF não estão implementados.
- As tabelas de endereço de CT-e e MDF-e são novas e têm menos verificação em
  campo que a da NF-e; confira-as antes de produção.

## [0.2.0] — 2026-08-17

Eventos e inutilização, fechando o ciclo de vida do documento depois da
autorização.

### Adicionado

- Pacote `evento` com os eventos da NF-e e da NFC-e: carta de correção
  (110110), cancelamento (110111), cancelamento por substituição (110112) e as
  quatro manifestações do destinatário (210200, 210210, 210220 e 210240).
- Inutilização de faixas de numeração (`inutNFe`), com Id de 43 caracteres,
  assinatura sobre o grupo `infInut` e montagem do comprovante.
- Montagem de lote `envEvento`, do arquivo de distribuição `procEventoNFe` e do
  `ProcInutNFe`, todos preservando os bytes assinados.
- `Cliente.EnviarEvento`, `Cliente.EnviarLoteDeEventos` e `Cliente.Inutilizar`,
  com os endereços de `NFeRecepcaoEvento4` e `NFeInutilizacao4` de todas as UFs.
- Roteamento automático das manifestações do destinatário para o Ambiente
  Nacional, que é outro servidor. Lotes que misturam destinos são recusados
  antes da transmissão, com a explicação do conflito.
- `nfe.AjustarParaHomologacao`, que aplica a razão social obrigatória do
  destinatário na NF-e e a descrição obrigatória do primeiro item na NFC-e.
- Guia de testes no ambiente de homologação, com roteiro em cinco passos,
  tabela de códigos de retorno e as armadilhas de consumo indevido.
- Guia de eventos na documentação e o exemplo executável `exemplos/eventos`.

### Corrigido

- Os atalhos de ícone da documentação (`:octicons-…:`) apareciam como texto
  cru: faltava a extensão `pymdownx.emoji` apontando para o índice do Material.

### Removido

- `evento.Tipo` não implementa mais `fmt.Stringer`. Um `String()` que devolvia
  o código mais a descrição corrompia silenciosamente qualquer formatação com
  `%s` — inclusive a montagem do atributo `Id` do evento, onde o defeito foi
  encontrado. Para a forma legível, use `Tipo.Rotulo()`.

### Alterado

- Os tipos `RetEvento`, `InfRetEvento` e `ProcEventoNFe` passaram a ser
  definidos no pacote `evento`; `sefaz` os reexporta como alias, então o código
  existente continua compilando. `ProcEventoNFe` agora carrega também o evento,
  não só o retorno.

## [0.1.0] — 2026-08-16

Primeira versão. Cobre o ciclo completo de emissão de NF-e e NFC-e no leiaute
4.00.

### Adicionado

**Documentos fiscais**

- Modelo de dados completo da NF-e modelo 55 e da NFC-e modelo 65, leiaute 4.00,
  espelhando o XSD campo a campo: identificação, emitente, destinatário, itens
  com todos os grupos de tributos, totais, transporte, cobrança, pagamento,
  informações adicionais, exportação, compra, cana e responsável técnico.
- Todas as variações do grupo ICMS — CST 00 a 90, `ICMSPart`, `ICMSST` e os
  CSOSN do Simples Nacional —, além de IPI, II, PIS, COFINS, ISSQN e a partilha
  `ICMSUFDest`.
- `NFe.Preparar`: normalização de texto, ajuste da escala decimal de cada campo
  conforme o leiaute, cálculo de totais e montagem da chave de acesso.
- `NFe.Validar`: validação estrutural com o caminho do campo em cada
  inconsistência, incluindo dígitos verificadores, coerência de somatórios e as
  regras específicas da NFC-e.
- `NFe.CalcularTotais` seguindo as regras do grupo W do Manual de Orientação.
- Montagem de lote `enviNFe` e do arquivo de distribuição `nfeProc`, preservando
  os bytes assinados.

**NFC-e**

- QR Code versão 2 com hash SHA-1 sobre o CSC, e URL de consulta por chave.
- Tabela de endereços de QR Code e de consulta das 27 unidades da federação.
- `ConferirQRCode` para validar um cupom recebido ou a própria configuração.

**Segurança**

- Carregamento de certificados A1 em PKCS#12, com suporte aos algoritmos
  modernos (AES-256 com PBES2) e aos antigos (3DES com SHA-1).
- Extração de CNPJ e CPF dos OIDs da ICP-Brasil no `subjectAltName`, com recurso
  ao nome comum quando a extensão não está presente.
- Assinatura e verificação XML-DSig no perfil da SEFAZ, preservando byte a byte
  os dados originais do documento.
- Canonicalização C14N 1.0 própria, conferida contra os casos 3.2 e 3.3 da
  especificação do W3C.
- Interface `xmldsig.Assinante` para plugar A3, HSM ou assinatura remota sem
  trazer CGO para o núcleo.

**Comunicação**

- Cliente SOAP 1.2 com autenticação mútua TLS, sem biblioteca externa de SOAP.
- Operações: status do serviço, autorização de lote, consulta de recibo com
  espera pelo processamento, consulta de nota pela chave e consulta de cadastro.
- Tabela de endereços por UF, modelo e ambiente, com os autorizadores próprios,
  SVAN, SVRS e as Sefaz Virtuais de Contingência; sobreponível serviço a
  serviço.

**Base**

- `tipos.Decimal`: decimal de precisão fixa, sem `float64` em nenhum ponto do
  caminho fiscal, com arredondamento comercial.
- `tipos.DataHora` e `tipos.Data` nos formatos do leiaute, com fuso explícito.
- Chave de acesso de 44 dígitos: montagem, dígito verificador módulo 11,
  validação e decomposição.
- Validação de CPF e de CNPJ, inclusive o alfanumérico da IN RFB 2.229/2024.
- Códigos do IBGE, nomes e fusos horários das 27 unidades da federação.

### Notas

- Requer Go 1.23 ou mais recente.
- Uma única dependência externa: `software.sslmate.com/src/go-pkcs12`.
- Eventos, distribuição de DF-e, CT-e, MDF-e e DANFE ainda não estão
  implementados; veja o roteiro no README.

[Não lançado]: https://github.com/sisgt-sistemas/gonfe/compare/v0.5.0...HEAD
[0.5.0]: https://github.com/sisgt-sistemas/gonfe/compare/v0.4.0...v0.5.0
[0.4.0]: https://github.com/sisgt-sistemas/gonfe/compare/v0.3.0...v0.4.0
[0.3.0]: https://github.com/sisgt-sistemas/gonfe/compare/v0.2.0...v0.3.0
[0.2.0]: https://github.com/sisgt-sistemas/gonfe/compare/v0.1.0...v0.2.0
[0.1.0]: https://github.com/sisgt-sistemas/gonfe/releases/tag/v0.1.0
