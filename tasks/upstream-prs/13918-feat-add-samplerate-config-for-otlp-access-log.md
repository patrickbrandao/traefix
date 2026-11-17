# Feat: Add sampleRate config for otlp access log

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13918](https://github.com/traefik/traefik/pull/13918) |
| Tipo | Recurso |
| Área | logs/accesslog-otlp |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | Thanhphan1147 |
| Aberto em | 2026-09-16 (última atualização: 2026-11-13) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +262/-4 em 9 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/logs, size/L, area/middleware |
| Status | importado com ajustes |

## O que faz

Adiciona accesslog.sampleRate (0.0 a 1.0, padrão 1.0) para reduzir o volume de access logs exportados ao OpenTelemetry Collector, mantendo cada registro com probabilidade igual ao rate. A amostragem envolve o log processor antes do batch processor e só vale para accesslog.otlp.

## Riscos e situação no upstream

Amostragem aleatória simples, sem determinismo; o autor deixou aberto se for necessário. Status needs-review, sem review. Toca 9 arquivos, incluindo observability/logs/otel.go.

## Principais arquivos

- `pkg/middlewares/accesslog/logger.go`
- `pkg/observability/logs/otel.go`
- `pkg/config/static/static_config.go`
- `docs/content/reference/install-configuration/observability/logs-and-accesslogs.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13918/head:upstream-pr-13918`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- Reviews só automatizadas (agente de IA, Aikido) e do próprio autor; sem maintainer humano. PR feito com assistência de IA, declarado.
- Amostragem aleatória (`math/rand/v2`, seguro para concorrência) via wrapper do processor OTLP; só afeta a exportação OTLP do access log, não o arquivo/stdout.
- Armadilha: `SampleRate` é `float64` com default 1.0 via `SetDefaults`. Um `AccessLog` montado sem `SetDefaults` (código, testes) tem `SampleRate == 0` e, no PR, isso descartava TODO o access log OTLP em silêncio. O próprio teste do PR tratava 0 como "descarta tudo".
- Conflito com os campos de rotação (#12752) e anonimização (#13526) já importados em `AccessLog`.

### Ajustes

- `NewLoggerProvider` só envolve o processor quando `0 < sampleRate < 1`; 0 é tratado como "não definido" (exporta tudo). Documentado na descrição do campo e na doc.
- Teste `TestAccessLogSampling` ajustado: taxa ínfima (1e-12) não exporta; 0 exporta.
- Conflito resolvido mantendo `MaxSize/MaxAge/MaxBackups/Compress` e `Anonymization`; `go generate`.
- Esta assinatura nova de `NewLoggerProvider` (com `sampleRate`) é a base para #13993/#13994.

### Testes

- `go vet ./pkg/... ./cmd/...`: ok.
- `go test ./pkg/middlewares/accesslog/ ./pkg/observability/types/ ./pkg/config/static/ ./pkg/observability/logs/`: ok.
- `golangci-lint run ./pkg/middlewares/accesslog/... ./pkg/observability/... ./pkg/config/static/...`: 0 issues.
