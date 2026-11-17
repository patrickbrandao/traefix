# fix(provider/docker): don't drop container config when an unreferenced auto-created service can't resolve a port

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13561](https://github.com/traefik/traefik/pull/13561) |
| Tipo | Correção |
| Área | provider/docker |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | RahulRingane |
| Aberto em | 2026-07-23 (última atualização: 2026-07-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +113/-0 em 2 arquivos |
| Labels | status/1-needs-design-review, area/provider/docker, kind/bug/fix, size/M |
| Status | importado com ajustes |

## O que faz

Se o serviço criado automaticamente para o container não consegue resolver porta, hoje o container inteiro é descartado, mesmo com routers que não usam esse serviço (ex.: api@internal). A falha só deixa de ser fatal quando o serviço foi inventado pelo Traefik e todos os routers já têm serviço explícito. Resolve #13532, causado por macvlan não preencher NetworkSettings.Ports.

## Riscos e situação no upstream

Lógica condicional nova que depende de reproduzir a condição de auto-wire do BuildRouterConfiguration. Status needs-design-review, sem review de maintainer ainda.

## Principais arquivos

- `pkg/provider/docker/config.go`
- `pkg/provider/docker/config_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13561/head:upstream-pr-13561`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- Bug real: quando o container não declara serviço, o provider cria um serviço padrão. Se a porta dele não pode ser resolvida (macvlan, sem `NetworkSettings.Ports`), o erro descarta **toda** a configuração HTTP do container, inclusive routers que apontam para outro serviço (por exemplo `api@internal`).
- A correção só ignora o erro quando o serviço foi criado automaticamente e nenhum router pode usá-lo. Isso espelha a regra de `BuildRouterConfiguration`: sem routers, cria-se o router padrão; um router sem `service` é ligado ao único serviço.
- **Brecha no critério original:** um router que referencia o serviço automático **pelo nome** (`service=<nome-do-container>`) era considerado "não dependente". O serviço seria apagado e o router ficaria apontando para um serviço inexistente, com o erro aparecendo só em runtime e não mais como "port is missing".
- Os testes usavam mensagens longas nas asserções, contra a convenção do projeto.
- Sem risco de segurança: só remove um serviço que ninguém usa. Complementa o #13404 (fallback para `ExposedPorts`), que resolve a maioria dos casos de macvlan com `EXPOSE`.

**Ajustes**

- `defaultServiceRequired` virou função pura, recebe o nome do serviço e também o considera necessário quando algum router o referencia explicitamente.
- Comentários reduzidos ao *porquê*.
- Os dois testes viraram uma tabela com 4 casos (serviço não usado; sem routers; router sem serviço; router referenciando o serviço automático), com asserções sem mensagens.

**Testes:** `go build ./pkg/...` ok; `go test ./pkg/provider/docker/` ok; `golangci-lint run ./pkg/provider/docker/...` com 0 issues.
