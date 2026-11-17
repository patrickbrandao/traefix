# Set http.route on root server span and update span name to include route

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13504](https://github.com/traefik/traefik/pull/13504) |
| Tipo | Recurso |
| Área | middleware/tracing |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | newdogwang-netizen |
| Aberto em | 2026-07-10 (última atualização: 2026-07-13) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +169/-21 em 4 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/M, area/middleware, area/middleware/tracing |
| Status | importado com ajustes |

## O que faz

Quando um router casa a requisição, o span server raiz recebe o atributo http.route com o caminho extraído da regra e passa a se chamar '{método} {rota}', conforme a semconv do OTel. Antes o nome era só o método. Fixes #11230.

## Riscos e situação no upstream

Status 1-needs-design-review, sem review. Mudar o nome do span pode quebrar dashboards e alertas existentes. Para regras sem Path, o fallback usa a regra crua.

## Principais arquivos

- `pkg/middlewares/observability/entrypoint.go`
- `pkg/middlewares/observability/router.go`
- `pkg/middlewares/observability/router_test.go`
- `pkg/middlewares/observability/mock_tracing_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13504/head:upstream-pr-13504`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- Sem review de maintainer (fixes #11230). Sobrepõe-se ao #14073 (`http.route` nas métricas semconv); ver a nota abaixo.
- A extração do path por regex era incorreta em vários casos:
  - `Path(/a) || Path(/b)` nomeava todas as requisições como `/a` ("o primeiro vence" não é verdade para OR);
  - `!PathPrefix(/admin)` nomeava como `/admin` justamente as requisições que NÃO são `/admin`;
  - regras com aspas duplas (`PathPrefix("/api")`, sintaxe válida) não eram reconhecidas;
  - no fallback, a regra crua (com hosts e regex) virava o nome do span (`GET Host(...)`), o que foge da convenção OTel (`{method}` quando não há rota).
- A regex rodava a cada requisição, em vez de uma vez na construção do router.
- Com routers aninhados e tracing detalhado, `trace.SpanFromContext` devolve o span `Router` do router pai, não o span do servidor: o PR renomearia o span errado.

### Ajustes

- A rota é calculada uma vez em `newRouter`; só é usada quando a regra tem exatamente um `Path`/`PathPrefix` (com crase ou aspas duplas) e nenhum `||` ou `!`. Caso contrário o span do servidor continua só com `{method}` e sem `http.route`.
- O entrypoint guarda o span do servidor no contexto (`serverSpanKey`), e o router enriquece exatamente esse span.
- O span `Router` continua com a regra crua em `http.route`, como antes.
- Testes: casos do PR ajustados e `TestExtractRouteFromRule` (OR, negação, aspas, PathRegexp, múltiplos paths).

### Nota sobre o #14073

O #14073 põe `http.route` nas métricas `http.server.request.duration` usando a regra crua (consistente com o span `Router`, cardinalidade limitada ao número de routers). Os dois foram importados: o span do servidor usa o path extraído só quando ele é inequívoco; a métrica usa a regra crua. Unificar exigiria decidir um formato único no upstream.

### Testes

- `go vet ./pkg/...`: ok.
- `go test ./pkg/middlewares/observability/`: ok.
- `golangci-lint run ./pkg/middlewares/observability/...`: 0 issues.
