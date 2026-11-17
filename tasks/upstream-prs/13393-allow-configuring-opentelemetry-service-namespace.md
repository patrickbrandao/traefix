# Allow configuring OpenTelemetry service namespace

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13393](https://github.com/traefik/traefik/pull/13393) |
| Tipo | Recurso |
| Área | observability/otel |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | nickmnt |
| Aberto em | 2026-06-24 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +67/-25 em 16 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/logs, size/S, area/middleware/tracing |
| Status | pendente |

## O que faz

Adiciona a opção serviceNamespace para logs, access logs, métricas e tracing OpenTelemetry, propagada ao atributo service.namespace junto do service.name, que já tem opção própria. Fixes #13293.

## Riscos e situação no upstream

Antes era possível via resourceAttributes genérico, então o ganho é de ergonomia. Sem review; maintainers disseram que está no radar, sem ETA. Toca config estática e docs.

## Principais arquivos

- `pkg/config/static/static_config.go`
- `pkg/observability/metrics/otel.go`
- `pkg/observability/tracing/tracing.go`
- `pkg/observability/logs/otel_test.go`
- `docs/content/reference/install-configuration/configuration-options.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13393/head:upstream-pr-13393`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
