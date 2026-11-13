# Contribuindo

Obrigado pelo interesse. Este documento diz o que é útil contribuir e como.

## O que ajuda mais

**Rejeições reais.** Se a SEFAZ rejeitou uma nota por algo que a validação local
devia ter pego, isso é ouro. Abra uma issue com o código de rejeição, o motivo
devolvido e o trecho do XML — sem dados reais de clientes e sem certificado.

**Endereços de serviço divergentes.** Os estados mudam endereços sem aviso. Se o
que está na tabela não bate com o Portal da NF-e, avise; a correção é de uma
linha.

**Cenários tributários.** A biblioteca foi exercitada em venda comum, Simples
Nacional e substituição tributária. Diferimento, partilha interestadual,
exportação e regimes específicos de UF têm menos rodagem.

**Os itens do roteiro.** Eventos, distribuição de DF-e, CT-e, MDF-e e DANFE.
Antes de começar algo grande, abra uma issue para combinarmos a abordagem — a
infraestrutura comum já existe e vale reaproveitar.

## Ambiente

Para **usar** a biblioteca basta Go 1.23 ou mais recente: ela não usa CGO nem
programa externo.

Para **desenvolver**, tudo roda no container de `docker/dev` (Go,
golangci-lint, govulncheck e o `xmllint`), pelo `Makefile`. Só Docker e `make`
no host:

```bash
git clone https://github.com/sisgt-sistemas/gonfe
cd gonfe
make image            # uma vez
make ci               # o gate: formatação, vet, golangci-lint e testes
make help             # o resto: cover, vuln, mutate, shell
```

O `make ci` é o mesmo que a CI roda no Linux e o que o fluxo exige antes de
integrar. Sem o container, `go test ./...` também funciona; os testes contra os
esquemas oficiais são pulados quando não há `xmllint`.

## Convenções do código

**Português.** Nomes de tipo e de função, comentários e mensagens de erro são em
português. Os nomes dos campos dos documentos fiscais seguem o XSD exatamente —
`VProd`, `CMunFG`, `ICMSSN102` — para que a conferência contra o Manual de
Orientação do Contribuinte seja direta. Não traduza nem "melhore" esses nomes.

**Nada de `float64` em valor fiscal.** Use `tipos.Decimal` sempre. Se você
precisar de um construtor novo, ele não deve receber `float64`.

**Comentários explicam o porquê.** O código já diz o quê. Comentário bom é o que
registra a regra do leiaute, a razão de uma decisão ou a armadilha que a
próxima pessoa cairia. Comentário que repete a assinatura da função é ruído.

**Erros com contexto.** Embrulhe com `%w` e diga qual campo, qual arquivo, qual
endereço. `fmt.Errorf("sefaz: falha na comunicação com %s: %w", endereco, err)`
economiza meia hora de quem for depurar.

**Formatação.** `gofmt` resolve. A CI reprova o que não estiver formatado.

## Leiaute

**O campo nasce do XSD.** Nome, ordem, obrigatoriedade, tamanho e casas
decimais vêm do esquema oficial da versão em vigor (em `xsd/esquemas`, com a
origem e o sha256 no README da pasta), não de exemplo nem de outra biblioteca.
O Manual de Orientação explica; o esquema decide.

**Todo documento passa no esquema.** CT-e, CT-e OS e MDF-e têm teste que monta o
documento, assina e confere com o pacote `xsd`. Campo novo entra no exemplo
desse teste. Esquema novo entra em `xsd/esquemas` byte a byte, com o sha256 no
README.

**Os gates em `internal/archtest`** conferem o que o compilador não vê:

- a biblioteca não usa CGO;
- só o pacote `xsd` roda programa externo;
- todo pacote de documento tem teste contra o esquema;
- a tag `dec:"N"` de cada campo decimal cabe nas casas do tipo `TDec_*` do
  esquema.

Um gate não se afrouxa para passar uma mudança: se a regra estiver errada,
corrija a regra com o motivo no pull request.

## Testes

Toda mudança de comportamento precisa de teste. Alguns padrões que o projeto
segue e que vale manter:

- **Vetores conferíveis.** Onde há algoritmo — dígito verificador, hash,
  canonicalização —, prefira casos que um leitor consiga verificar à mão, ou
  contraprova por uma implementação independente dentro do próprio teste. Um
  teste que só confirma o que o código faz não prova nada.
- **Casos oficiais.** A canonicalização é conferida contra os casos 3.2 e 3.3 da
  especificação Canonical XML 1.0 do W3C. Se você tocar nela, esses testes
  precisam continuar passando.
- **Sem rede.** Os testes não acessam a internet. Para exercitar o cliente
  SEFAZ, use `httptest.Server` e `sefaz.Config.HTTP`, como em
  `sefaz/cliente_test.go`.
- **Sem certificado real.** Use `internal/certtest` para gerar certificados
  sintéticos.
- **Mensagens em português**, dizendo o que se esperava e o que veio:
  `t.Errorf("vProd = %s, queria %s", got, want)`.
- **Regra nova traz teste, e o teste é visto falhar.** Antes de abrir o pull
  request, quebre uma vez o código que a regra protege e confira que o teste
  reprova; diga no pull request como sabotou.

## Documentação

A documentação de referência é o próprio GoDoc — comentários nos tipos e funções
exportados. Os guias em `docs/` cobrem o que não cabe em GoDoc: o porquê das
decisões e os caminhos completos.

Para ver o site localmente:

```bash
pip install -r docs/requirements.txt
mkdocs serve
```

## Enviando a mudança

1. Abra uma issue antes de mudanças grandes, para combinarmos a abordagem.
2. Trabalhe em um branch `feat/…` ou `fix/…` a partir de `develop`. O `main`
   só guarda versões lançadas.
3. Deixe o `make ci` verde. A CI roda também `-race`, `staticcheck`,
   `govulncheck` e a matriz de sistemas e versões de Go.
4. Escreva o commit em [Conventional Commits](https://www.conventionalcommits.org/pt-br/):
   `<tipo>(<escopo>): <assunto>`, com o assunto em inglês, no imperativo, até
   72 caracteres, e o porquê no corpo. Se o commit corrige uma rejeição da
   SEFAZ, cite o código. A versão sai dos commits: `feat` sobe o minor, `tipo!:`
   ou `BREAKING CHANGE:` sobe o major, o resto sobe o patch.
5. Anote a mudança que quem usa a biblioteca vê na seção **Não lançado** do
   `CHANGELOG.md`, em português; quebra de API leva **Incompatível**. As
   versões já lançadas e o `VERSION` só mudam no commit de release, que
   renomeia a seção para a versão.
6. Abra o pull request para `develop`, descrevendo o problema que ele resolve.

Os ganchos de `.githooks` conferem a mensagem e o gate antes do push; ative com
`git config core.hooksPath .githooks`.

## Segurança

Se você encontrar uma vulnerabilidade — algo que permita forjar uma assinatura,
vazar a chave privada ou o CSC —, não abra issue pública. Veja
[SECURITY.md](SECURITY.md).

## Licença

Ao contribuir, você concorda em licenciar sua contribuição sob a
[licença MIT](LICENSE) do projeto.
