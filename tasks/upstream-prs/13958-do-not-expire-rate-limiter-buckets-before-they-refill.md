# Do not expire rate limiter buckets before they refill

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13958](https://github.com/traefik/traefik/pull/13958) |
| Tipo | Correção |
| Área | middleware/ratelimit |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | rasoanaivo-r |
| Aberto em | 2026-09-24 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +87/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/middleware |
| Status | pendente |

## O que faz

Corrige o TTL dos buckets do rate limiter, que expirava em cerca de 1/rate e recriava o bucket cheio. Com burst alto e rate baixo, pausas curtas davam rajadas completas repetidas. O TTL passa a ser max(ttl atual, ceil(burst/rate)), e o Redis herda a mesma correção.

## Riscos e situação no upstream

Aumenta a vida dos buckets (memória limitada por maxSources). Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/middlewares/ratelimiter/rate_limiter.go`
- `pkg/middlewares/ratelimiter/rate_limiter_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13958/head:upstream-pr-13958`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
