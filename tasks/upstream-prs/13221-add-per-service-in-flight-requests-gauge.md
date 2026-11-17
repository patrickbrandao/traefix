# Add per-service in-flight requests gauge

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13221](https://github.com/traefik/traefik/pull/13221) |
| Tipo | Recurso |
| Área | middlewares/metrics, observability/metrics |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | eugenetaranov |
| Aberto em | 2026-05-21 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +297/-31 em 9 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/middleware |
| Status | pendente |

## O que faz

Adiciona o gauge traefik_service_inflight_requests, com labels service, method e protocol, em todos os backends de métricas (Prometheus, StatsD, Datadog, InfluxDB2 e OTLP). Dá visibilidade de carga por serviço HTTP, complementando o issue #11006. Respeita a flag AddServicesLabels.

## Riscos e situação no upstream

Label needs-design-review e maintainer pediu análise de impacto. Amplia a superfície de métricas em 5 backends, com risco de aumento de cardinalidade, mitigado pela flag AddServicesLabels.

## Principais arquivos

- `pkg/middlewares/metrics/metrics.go`
- `pkg/middlewares/metrics/inflight_requests_test.go`
- `pkg/observability/metrics/prometheus.go`
- `pkg/observability/metrics/otel.go`
- `pkg/observability/metrics/metrics.go`
- `docs/content/reference/install-configuration/observability/metrics.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13221/head:upstream-pr-13221`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
