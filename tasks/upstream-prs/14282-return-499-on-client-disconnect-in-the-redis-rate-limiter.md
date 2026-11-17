# Return 499 on client disconnect in the Redis rate limiter

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14282](https://github.com/traefik/traefik/pull/14282) |
| Tipo | Correção |
| Área | middleware/ratelimit (Redis) |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | kxxil01 |
| Aberto em | 2026-11-13 (última atualização: 2026-11-16) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +69/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | pendente |

## O que faz

Quando o cliente desconecta antes de o Redis responder, o rate limiter respondia 500 e gerava log de erro. Agora retorna 499 sem corpo e log de debug, como o ForwardAuth. Timeouts continuam em 500.

## Riscos e situação no upstream

Base é v3.7, precisa de port. Muda só o ramo de erro do Allow e o log, então o efeito é em métricas e logs. Sem review de maintainer ainda (status/2-needs-review).

## Principais arquivos

- `pkg/middlewares/ratelimiter (ramo de erro do Allow no Redis)`
- `pkg/middlewares/ratelimiter (testes)`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14282/head:upstream-pr-14282`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
