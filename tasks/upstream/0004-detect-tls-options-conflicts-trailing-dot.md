# 0004 — Detect TLS options conflicts on hosts with a trailing dot

- **Status:** aplicada (`e3f72b006`, 2026-11-23)
- **Tipo:** bugfix
- **Prioridade:** média
- **Upstream:** traefik/traefik#14192 — commits `7d22f5cfe` (branch v2.11, entrou no v3 pelo merge `ece978d30`)
- **Aplica limpo:** sim
- **Conflito com Traefix:** não

## O que mudou no oficial
`pkg/server/aggregator.go` (`findConflictingRouters`): o domínio é normalizado com `strings.TrimSuffix(domain, ".")`
antes de agrupar por host e opção TLS. Os matchers `Host` e `HostSNI` ignoram o ponto final, então `example.com.` e
`example.com` são o mesmo host, e um router com `example.com.` e outras opções TLS escapava da detecção de
conflito (e do fallback para as opções `default`). Teste novo em `aggregator_test.go`.

## Impacto no Traefix
Nenhuma alteração local em `aggregator.go`. Combina com o import do #14154 (`HostRegexp` com ponto final). Os
routers `@mesh` criados pela malha passam pela mesma agregação, então a checagem também vale para eles, o que é o
comportamento desejado.

O commit é do branch v2.11, mas o diff só acrescenta o import `strings` e as três linhas; o cherry-pick no v3 não
traz import `traefik/v2` (conferido).

## Plano de implementação
1. `git cherry-pick -x 7d22f5cfe`
2. Conferir que nenhum import `github.com/traefik/traefik/v2` entrou (`git grep traefik/v2 -- pkg/server`).
3. Testes: `go test ./pkg/server/`

## Critério de pronto
- Hosts com e sem ponto final caem no mesmo grupo de conflito de opções TLS.
- `go test ./pkg/server/` e `make lint` sem erros.
