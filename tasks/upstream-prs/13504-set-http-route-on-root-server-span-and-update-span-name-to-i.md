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
| Status | pendente |

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
