# feat: add health track to ratelimit middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12042](https://github.com/traefik/traefik/pull/12042) |
| Tipo | Recurso |
| Área | middleware/ratelimit |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | david-garcia-garcia |
| Aberto em | 2025-09-05 (última atualização: 2025-09-08) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +485/-1 em 5 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/middleware |
| Status | adiado |

## O que faz

Adiciona backoff ao ratelimit com Redis: após falhas repetidas no backend, o limitador fica desligado por um período e as requisições passam, evitando que uma queda do Redis derrube o middleware. Novas opções backoffDuration, backoffTimeout e backoffThreshold.

## Riscos e situação no upstream

Muda o caminho crítico de ratelimit (5 arquivos, +485/-1). Comportamento de falha (fail-open temporário) é decisão de segurança a validar. Status needs-design-review, sem comentários de maintainers.

## Principais arquivos

- `pkg/middlewares/ratelimiter/rate_limiter.go`
- `pkg/middlewares/ratelimiter/health_tracker.go`
- `pkg/config/dynamic/middlewares.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12042/head:upstream-pr-12042`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: adiado.**

Achados no PR original:

- Data race real em `healthTracker.isShutdownNow`: lê `isShutdown` e `shutdownUntil` sem lock
  ("fast path lockless"), enquanto `recordFailure` escreve sob lock. O race detector acusa isso e o
  comportamento não é garantido pelo modelo de memória do Go.
- As três opções novas ficam em `RateLimit` (e não em `Redis`), embora só o backend Redis possa falhar;
  não há suporte no CRD do Kubernetes nem documentação.
- O PR está em `status/1-needs-design-review` no upstream, sem retorno dos maintainers.
- Sobreposição com #13916 (`denyOnError`), importado neste mesmo grupo: os dois tratam a falha do
  Redis. O #13916 cobre o caso principal (fail-open configurável) com bem menos superfície. O ganho
  que este PR traz a mais é um circuit breaker que evita pagar o timeout do Redis em toda requisição
  durante a queda; isso pode ser feito depois sobre o `denyOnError`, mas pede desenho próprio.

Testes: nenhum (não importado).

Motivo: risco de concorrência no caminho crítico, design pendente no upstream e sobreposição com o
#13916, que foi o escolhido para o comportamento de falha do Redis.
