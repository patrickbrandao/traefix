# 0008 — Bump go.opentelemetry.io/otel/sdk/log to v1.47.0

- **Status:** aplicada (`956481d4f`, 2026-11-23)
- **Tipo:** deps
- **Prioridade:** baixa
- **Upstream:** traefik/traefik#14315 — commits `08964addb` (branch v3.7)
- **Aplica limpo:** sim
- **Conflito com Traefix:** não

## O que mudou no oficial
`go.mod`/`go.sum`: `go.opentelemetry.io/otel/sdk/log` de `v0.22.0` para `v1.47.0` (primeira versão estável) e os
exportadores `otlplog/otlploggrpc` e `otlplog/otlploghttp` de `v0.22.0` para `v0.23.0` (`sdk/log/logtest` vai a
`v0.23.0`). O resto do otel já estava na v1.47.0. Não é correção de segurança.

O PR irmão #14316 (`4bc6e92a9`, "Bump go.opentelemetry.io/otel to v1.47.0") é do branch v2.11 e não tem efeito no v3:
o merge `ece978d30` ficou com o `go.mod` do v3.7, que já estava na v1.47.0. Ele não tem task (ver relatório da
sincronização).

## Impacto no Traefix
O Traefix alterou o `go.mod` (Coraza, jsonschema, miniredis, gjson, bloco `replace` dos forks `traefix-*`), mas não as
linhas do otel. O bump **não toca** `github.com/quic-go/quic-go`, `github.com/quic-go/qpack` nem `golang.org/x/net`
(nenhuma linha dessas no `go.sum` do commit), então não passa pelos agentes dos forks. No worktree de teste,
`go build ./...`, `go mod verify`, `go mod tidy -diff` e `scripts/traefix/check-forks.sh` passaram.

## Plano de implementação
1. `git cherry-pick -x 08964addb`
2. `scripts/traefix/check-forks.sh` (deve continuar passando) e `go mod tidy -diff` (sem diferença).
3. Testes: `go test ./pkg/observability/...`

## Critério de pronto
- `sdk/log` em v1.47.0 e `otlplog*` em v0.23.0 no `go.mod`; bloco `replace` dos forks sem mudança.
- `check-forks.sh`, `go build ./...` e os testes de `pkg/observability` sem erros.
