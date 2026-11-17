# fix(metrics): prune stale OpenTelemetry gauge entries to stop unbounded memory growth

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13888](https://github.com/traefik/traefik/pull/13888) |
| Tipo | Correção |
| Área | observability/metrics-otel |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | mikegraycodes |
| Aberto em | 2026-09-11 (última atualização: 2026-09-11) |
| Draft | sim |
| Branch-alvo | `v3.7` |
| Tamanho | +421/-16 em 3 arquivos |
| Labels | status/0-needs-triage, size/L |
| Status | pendente |

## O que faz

Corrige vazamento de memória no coletor de gauges do OpenTelemetry, que mantinha uma entrada por combinação de atributos para sempre (traefik_open_connections e traefik_service_server_up). Replica a poda por diff que o Prometheus já faz, aplicada a cada atualização de configuração.

## Riscos e situação no upstream

PR em draft, +421/-16, sem review. Lógica de remoção com marcadores entre gauges independentes exige cuidado e precisa de testes de concorrência. Status 0-needs-triage. Vinculado a #12232.

## Principais arquivos

- `pkg/observability/metrics/otel.go`
- `pkg/observability/metrics/otel_test.go`
- `pkg/observability/metrics/prometheus.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13888/head:upstream-pr-13888`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
