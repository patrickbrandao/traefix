# Fix Redis rate limiter not limiting when hash field order changes

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14300](https://github.com/traefik/traefik/pull/14300) |
| Tipo | Correção |
| Área | middleware/ratelimit (Redis) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | vecz-akos |
| Aberto em | 2026-11-15 (última atualização: 2026-11-15) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +31/-23 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | pendente |

## O que faz

O script Lua do rate limiter Redis usava HGETALL e dependia da ordem dos campos do hash. Com encoding hashtable, o Redis pode devolver tokens antes de last, e aí o limitador nunca limita. A correção usa HMGET com os campos nomeados.

## Riscos e situação no upstream

Base é v3.7, precisa de port. Bug confirmado no master: pkg/middlewares/ratelimiter/lua.go linha 34 usa hgetall. Mudança pequena e sem efeito colateral conhecido. Sem review de maintainer ainda (status/2-needs-review).

## Principais arquivos

- `pkg/middlewares/ratelimiter/lua.go`
- `pkg/middlewares/ratelimiter/rate_limiter_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14300/head:upstream-pr-14300`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
