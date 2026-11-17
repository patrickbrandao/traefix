# fix(ratelimiter): allow all requests when the Redis backend has rate limiting disabled (average 0)

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13512](https://github.com/traefik/traefik/pull/13512) |
| Tipo | Correção |
| Área | middleware/ratelimiter (backend Redis) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | TowyTowy |
| Aberto em | 2026-07-12 (última atualização: 2026-09-22) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +36/-1 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | importado |

## O que faz

Com average: 0 (padrão, documentado como sem limite), o backend Redis rejeitava todas as requisições com 429, enquanto o backend em memória deixava passar. O evaluateScript retornava nil no ramo rate.Inf, que é o sinal de rejeição. O fix faz o Redis liberar tudo, alinhando com o backend em memória e com a documentação.

## Riscos e situação no upstream

Correção pequena, com teste. Afeta qualquer configuração com redis e average 0. Status needs-review, sem review de maintainer ainda. PR base v3.7.

## Principais arquivos

- `pkg/middlewares/ratelimiter/redis_limiter.go`
- `pkg/middlewares/ratelimiter/rate_limiter_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13512/head:upstream-pr-13512`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado.**

Achados: bug confirmado no código atual. Com `average: 0` (`rate.Inf`), `redisLimiter.evaluateScript`
devolvia delay `nil`, que o `ServeHTTP` interpreta como rejeição, e o backend Redis respondia 429 a toda
requisição, enquanto o in-memory deixava tudo passar. A correção (delay zero) é mínima e não toca no Redis.
Nada suspeito no diff.

Ajustes: só o comentário, reescrito para dizer o porquê (delay `nil` significa rejeitar).

Testes: `go test ./pkg/middlewares/ratelimiter/` (novo `TestRedisRateLimitAverageZeroDisablesLimiting` ok),
`go build ./...` ok, `golangci-lint` 0 issues.
