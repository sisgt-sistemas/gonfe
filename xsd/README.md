# Esquemas oficiais (pacote `xsd`)

Os arquivos de `esquemas/` são os pacotes de liberação publicados pela SEFAZ, sem alteração, só os `.xsd`. Baixados em
2026-11-13 do Portal Nacional do CT-e, página "Esquemas XML"
(<https://www.cte.fazenda.gov.br/portal/listaConteudo.aspx?tipoConteudo=0xlG1bdBass=>).

| Pasta | Pacote | Publicação | sha256 do .zip |
|---|---|---|---|
| `esquemas/cte` | Schemas XML CT-e, NT 2026.004, RTC, v1.01 (`PL_CTe_400_NT2026.004 RTC_1.01`): CT-e, CT-e Simplificado, CT-e OS, GTV-e, eventos e modais | republicado em 11/11/2026 | `88f3e1f94e7c7d26b3e47367d273c303fe2b4643bf2bf01c4cdbf5f05b4e7714` |
| `esquemas/mdfe` | Schemas XML MDF-e, Pacote de Liberação 3.00b, "Schemas NT 2025.001 v 1.04" (`PL_MDFe_300b_NT012025_1.04.zip`, cuja pasta interna se chama `PL_MDFe_300b_NT012025_1.05`), baixado em 2026-11-13 do Portal do MDF-e (<https://dfe-portal.svrs.rs.gov.br/Mdfe/Documentos>) | 25/04/2026 | `fc53c880bd757d5b03b79de15dd7687259dfbfa38fed85e9ebb9c638b93854cc` |

Para atualizar (uma nota técnica nova):

1. Baixe o pacote, troque os `.xsd` da pasta e atualize esta tabela.
2. Rode `make ci`. O gate `TestModelFollowsTheSchema` (`internal/archtest`) compara cada caminho de elemento e de
   atributo do esquema, da raiz às folhas, com as tags `xml` do modelo Go, no modal rodoviário: lista o que o esquema
   ganhou e o modelo não tem, e o que o modelo tem e o esquema tirou ou mudou de lugar.
3. Trabalhe a lista: modele o campo novo (com `Validar()` e, se o manual pede, o documento auxiliar) ou registre a
   falta em `semCampo`, com o motivo. Exceção que deixa de ser necessária reprova o gate.
4. Os testes de `cte`, `cteos`, `ctesimp` e `mdfe` validam os documentos de exemplo contra os esquemas novos.

A validação é só de teste e precisa do `xmllint` (pacote `libxml2-utils` no Debian e no Ubuntu). Sem ele, os testes
que validam contra o esquema são pulados.
