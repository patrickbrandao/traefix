# feat(metrics): export Go runtime metrics via OTLP

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13126](https://github.com/traefik/traefik/pull/13126) |
| Tipo | Recurso |
| Área | observability/otel |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | ElfoLiNk |
| Aberto em | 2026-05-07 (última atualização: 2026-10-02) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +13/-0 em 3 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/middleware/metrics, area/otel |
| Status | adiado |

## O que faz

Registra a instrumentação de runtime do Go do OpenTelemetry no MeterProvider OTLP, para que métricas process.runtime.go.* (goroutines, heap, GC) saiam pelo mesmo exporter OTLP. Útil quando o Prometheus está desligado e tudo vai para um OTel Collector.

## Riscos e situação no upstream

Código pequeno (9 linhas em otel.go), mas traz bump de dependências em go.mod e go.sum. O branch ficou 251 commits atrás e teve conflito repetido nesses arquivos, então o rebase precisa ser refeito. Sem revisão de maintainer.

## Principais arquivos

- `pkg/observability/metrics/otel.go`
- `go.mod`
- `go.sum`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13126/head:upstream-pr-13126`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- Adiciona dependência nova `go.opentelemetry.io/contrib/instrumentation/runtime v0.71.0`. É do projeto oficial OpenTelemetry (mesma família de `otelhttp`, já usado), então não é de terceiro desconhecido, mas é uma mudança em `go.mod`/`go.sum`.
- O módulo não está no cache local, e o brief proíbe baixar qualquer coisa além do diff/metadados: não dá para compilar nem testar aqui.
- As métricas de runtime ficam sempre ligadas quando OTLP está habilitado, sem opção para desligar: muda o volume exportado para todos os usuários de OTLP, ao contrário do Prometheus, onde o GoCollector é esperado. Seria melhor uma opção explícita (ex.: `metrics.otlp.addRuntimeMetrics`).
- Sem testes nem documentação; sem review de maintainer.

### Ajustes

Nenhum.

### Testes

Não aplicável.

### Motivo

Adiado: exige baixar uma dependência nova (fora do que o brief permite neste ciclo) e precisa de opt-in antes de importar. Reavaliar fazendo o `go get` de forma controlada, alinhando a versão do contrib com as que já estão no `go.mod`.
