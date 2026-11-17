# Do not reject allowed requests from disconnected clients in rateLimit

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14174](https://github.com/traefik/traefik/pull/14174) |
| Tipo | Correção |
| Área | middleware/ratelimit |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | sarankumar1325 |
| Aberto em | 2026-10-29 (última atualização: 2026-11-11) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +74/-6 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | importado com ajustes |

## O que faz

O middleware rateLimit só espera quando há delay maior que zero. Antes, uma requisição permitida com cliente já desconectado respondia 500 context canceled e nunca chegava ao serviço. Com delay real e cliente que sai, a resposta passa a ser 499 em vez de 500.

## Riscos e situação no upstream

Escopo pequeno (rate_limiter.go e testes). Status: needs-review. Afeta métricas de erro e o código de status em desconexões.

## Principais arquivos

- `pkg/middlewares/ratelimiter/rate_limiter.go`
- `pkg/middlewares/ratelimiter/rate_limiter_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14174/head:upstream-pr-14174`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados: correto. Com delay zero (caso normal abaixo do limite), o `select` entre `ctx.Done()` e
`time.After(0)` escolhia aleatoriamente o cancelamento quando o cliente já tinha ido embora, e a
requisição permitida virava `500 context canceled`. Agora só espera se `delay > 0`, e o cancelamento
durante a espera responde `499` (`httputil.StatusClientClosedRequest`, já usado pelo proxy). Sem ciclo de
import (`pkg/proxy/httputil` não depende do ratelimiter). Nada suspeito no diff.

Ajustes: estendido ao caminho de erro do limiter, combinado com o #13916: se a chamada ao Redis falha
com `context.Canceled` porque o cliente desconectou, responde `499` em vez de contar como falha do Redis
(500 com `denyOnError: true`, ou encaminhar sem limite com `false`). Novo caso em
`TestRateLimit_CanceledRequest` cobre isso.

Testes: `go test -count=5` de `TestRateLimit_CanceledRequest` e `TestRateLimiterDenyOnError` ok,
`golangci-lint` 0 issues.
