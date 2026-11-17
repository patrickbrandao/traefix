# Inject active span context into request headers for plugin middlewares

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13008](https://github.com/traefik/traefik/pull/13008) |
| Tipo | Correção |
| Área | observability/tracing, middlewares/plugins |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | JoeGruffins |
| Aberto em | 2026-04-20 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +98/-0 em 4 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/middleware |
| Status | pendente |

## O que faz

Injeta o span ativo (traceparent/tracestate) nos headers da requisição, no entrypoint de tracing e no wrapper de tracing por middleware. Assim, plugins que fazem chamadas outbound propagam o contexto e não geram traces órfãos. Fecha o issue #12891.

## Riscos e situação no upstream

Label status/2-needs-review sem revisão de maintainer; há um pedido de bump de outro usuário. Toca middlewares/observability (entrypoint.go e middleware.go), com baixo risco de conflito.

## Principais arquivos

- `pkg/middlewares/observability/entrypoint.go`
- `pkg/middlewares/observability/middleware.go`
- `pkg/middlewares/observability/entrypoint_test.go`
- `pkg/middlewares/observability/middleware_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13008/head:upstream-pr-13008`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
