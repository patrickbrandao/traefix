# 0001 — Keep the OCSP staple cache janitor alive

- **Status:** aplicada (`35c665896`, 2026-11-23)
- **Tipo:** bugfix
- **Prioridade:** média
- **Upstream:** traefik/traefik#14257 — commits `ed7bd614b` (branch v3.7)
- **Aplica limpo:** sim
- **Conflito com Traefix:** não

## O que mudou no oficial
`pkg/tls/ocsp.go`: o campo `cache` do `ocspStapler` passa de `cache.Cache` (cópia do valor devolvido por
`cache.New`) para `*cache.Cache`. O go-cache para a goroutine de limpeza (janitor) com um finalizer no ponteiro
devolvido por `cache.New`; ao guardar uma cópia, o ponteiro original era coletado, o janitor parava e os
grampos OCSP vencidos nunca saíam da memória (vazamento lento).

## Impacto no Traefix
O Traefix alterou `pkg/tls/ocsp.go` no import do #14213 (`2531051d8`: emissor intermediário do grampo, `sync.RWMutex`
em `ocspEntry`, troca da entrada quando o emissor muda). As mudanças não se sobrepõem: o oficial só mexe na
declaração do campo e no construtor. Nenhum outro lugar constrói `ocspStapler` com o valor do cache.

## Plano de implementação
1. `git cherry-pick -x ed7bd614b`
2. Nenhum ajuste: o import do #14213 continua igual.
3. Testes: `go test ./pkg/tls/...` (passou no worktree de teste).

## Critério de pronto
- `ocspStapler.cache` é `*cache.Cache` e `newOCSPStapler` guarda o ponteiro de `cache.New`.
- `go test ./pkg/tls/...` e `make lint` sem erros.
