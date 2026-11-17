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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

### Achados

- Sem review de maintainer; mudança pequena (closes #12891). A motivação declarada pelo autor é fraca ("conseguir commits em repos populares"), mas o código é simples e o comportamento faz sentido.
- Mudança de comportamento: `traceparent`/`tracestate` recebidos do cliente são sobrescritos no `req.Header` antes dos middlewares (o mapa é compartilhado com o request original). Middlewares e o access log passam a ver o contexto do span do Traefik; o upstream já sobrescreve no RoundTripper de saída, então o backend não muda.
- A injeção vinha antes do `CaptureServerRequest`, então os headers capturados no span do entrypoint (`tracing.capturedRequestHeaders`) registravam o traceparent do próprio Traefik em vez do enviado pelo cliente.
- Só ocorre com tracing habilitado; nada de dado sensível novo em headers.

### Ajustes

- No entrypoint, a injeção foi movida para depois do `CaptureServerRequest`, preservando no span os headers originais do cliente.
- Comentários reescritos para explicar o porquê.

### Testes

- `go build ./...`: ok.
- `go test ./pkg/middlewares/observability/`: ok (inclui os testes novos do PR).
- `golangci-lint run ./pkg/middlewares/observability/...`: 0 issues.
