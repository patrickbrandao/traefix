# Add router-level open connections metric for TCP routers

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12685](https://github.com/traefik/traefik/pull/12685) |
| Tipo | Recurso |
| Área | observability/metrics/tcp |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | pavanreddy07385 |
| Aberto em | 2026-02-14 (última atualização: 2026-11-16) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +1118/-213 em 25 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/middleware/metrics, area/tcp |
| Status | pendente |

## O que faz

Restaura a métrica traefik_router_open_connections para routers TCP, com labels router e service, que existia na v2 e foi perdida na v3. Permite atribuir conexões a um backend específico, útil para autoscaling (KEDA, HPA) e detectar backends sobrecarregados.

## Riscos e situação no upstream

Grande (25 arquivos, +1118/-213). Altera a interface Registry e os 5 backends de métricas. Status design-review. Há comentários de revisão de docs pendentes.

## Principais arquivos

- `pkg/middlewares/tcp/metrics/metrics.go`
- `pkg/observability/metrics/metrics.go`
- `pkg/observability/metrics/prometheus.go`
- `cmd/traefik/traefik.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12685/head:upstream-pr-12685`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
