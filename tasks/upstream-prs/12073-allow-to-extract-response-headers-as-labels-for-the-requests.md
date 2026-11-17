# Allow to extract response headers as labels for the requests_total metrics

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12073](https://github.com/traefik/traefik/pull/12073) |
| Tipo | Recurso |
| Área | middleware/metrics |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | jybp |
| Aberto em | 2025-09-16 (última atualização: 2026-03-12) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +306/-53 em 18 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/middleware/metrics |
| Status | pendente |

## O que faz

Adiciona responseHeaderLabels às métricas requests_total do Prometheus, espelhando headerLabels existente para headers de requisição. Útil para labels como X-Cache-Status ou headers próprios das aplicações.

## Riscos e situação no upstream

Toca 18 arquivos, boa parte testes e config. Label status/2-needs-review, sem reviews. Cuidado com cardinalidade alta se o operador escolher headers variáveis.

## Principais arquivos

- `pkg/middlewares/metrics/metrics.go`
- `pkg/observability/metrics/headers.go`
- `pkg/observability/metrics/prometheus.go`
- `pkg/observability/types/metrics.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12073/head:upstream-pr-12073`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
