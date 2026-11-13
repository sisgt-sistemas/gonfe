# Esquemas oficiais (pacote `xsd`)

Os arquivos de `esquemas/` são os pacotes de liberação publicados pela SEFAZ, sem alteração, só os `.xsd`. Baixados em
2026-11-13 do Portal Nacional do CT-e, página "Esquemas XML"
(<https://www.cte.fazenda.gov.br/portal/listaConteudo.aspx?tipoConteudo=0xlG1bdBass=>).

| Pasta | Pacote | Publicação | sha256 do .zip |
|---|---|---|---|
| `esquemas/cte` | Schemas XML CT-e, NT 2026.004, RTC, v1.01 (`PL_CTe_400_NT2026.004 RTC_1.01`): CT-e, CT-e Simplificado, CT-e OS, GTV-e, eventos e modais | republicado em 11/11/2026 | `88f3e1f94e7c7d26b3e47367d273c303fe2b4643bf2bf01c4cdbf5f05b4e7714` |
| `esquemas/mdfe` | Schemas XML MDF-e, Pacote de Liberação 3.00b, "Schemas NT 2025.001 v 1.04" (`PL_MDFe_300b_NT012025_1.04.zip`, cuja pasta interna se chama `PL_MDFe_300b_NT012025_1.05`), baixado em 2026-11-13 do Portal do MDF-e (<https://dfe-portal.svrs.rs.gov.br/Mdfe/Documentos>) | 25/04/2026 | `fc53c880bd757d5b03b79de15dd7687259dfbfa38fed85e9ebb9c638b93854cc` |

Para atualizar: baixe o pacote novo, troque os `.xsd` da pasta, atualize esta tabela e rode os testes (`cte`, `mdfe`
e `xsd`), que validam os documentos de exemplo contra os esquemas.

A validação é só de teste e precisa do `xmllint` (pacote `libxml2-utils` no Debian e no Ubuntu). Sem ele, os testes
que validam contra o esquema são pulados.
