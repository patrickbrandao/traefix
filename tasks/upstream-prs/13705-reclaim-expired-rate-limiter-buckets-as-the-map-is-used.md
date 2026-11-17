# Reclaim expired rate limiter buckets as the map is used

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13705](https://github.com/traefik/traefik/pull/13705) |
| Tipo | Correção |
| Área | middleware/ratelimiter (ttlmap, uso de memória) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | deyanp |
| Aberto em | 2026-08-17 (última atualização: 2026-10-20) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +93/-5 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, area/middleware |
| Status | pendente |

## O que faz

O ttlmap só removia entradas expiradas quando a chave era consultada de novo ou quando o mapa enchia, então cada inMemoryRateLimiter crescia até 65536 entradas e ficava assim. Agora o Set remove entradas expiradas da raiz do heap, limitado a 16 por chamada, sem goroutine de fundo. Resolve #13704.

## Riscos e situação no upstream

Mudança pequena e localizada, com teste. Sem goroutine de fundo, para evitar vazamento a cada reload. Status needs-review, sem review ainda. PR base v3.7.

## Principais arquivos

- `pkg/middlewares/ratelimiter/ttlmap/ttlmap.go`
- `pkg/middlewares/ratelimiter/ttlmap/ttlmap_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13705/head:upstream-pr-13705`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
