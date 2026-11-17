# Add configurable metrics temporality for OTLP exporter

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12789](https://github.com/traefik/traefik/pull/12789) |
| Tipo | Recurso |
| Área | observability/metrics/otlp |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | JensPauwels |
| Aberto em | 2026-03-10 (última atualização: 2026-03-12) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +64/-23 em 4 arquivos |
| Labels | status/2-needs-review, size/S, area/middleware/metrics |
| Status | pendente |

## O que faz

Adiciona metrics.otlp.temporality (cumulative, delta ou lowmemory), para escolher a temporalidade das métricas OTLP por configuração do Traefik. Útil para backends como o Datadog, que preferem delta. O padrão continua cumulative.

## Riscos e situação no upstream

Pequeno (4 arquivos, +64/-23) e com testes e docs. Status review, sem maintainer comentando.

## Principais arquivos

- `pkg/observability/metrics/otel.go`
- `pkg/observability/metrics/otel_test.go`
- `pkg/observability/types/metrics.go`
- `docs/content/reference/install-configuration/observability/metrics.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12789/head:upstream-pr-12789`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
