# 0013 — Update Yaegi v2 and add fileMapping and mounts settings

- **Status:** aplicada (`97e317578`, 2026-11-23)
- **Tipo:** feature
- **Prioridade:** média
- **Upstream:** traefik/traefik#14303 — commits `3aefb4549` (branch v3.8)
- **Aplica limpo:** sim
- **Conflito com Traefix:** não

## O que mudou no oficial
- `go.mod`: `github.com/traefik/yaegi/v2` sai da pseudo-versão `v2.0.0-20261112132425-b76896158fba` (branch v2 ainda
  sem release) para `v2.0.0`.
- `pkg/plugins/yaegiv2.go`: importa `yaegi/v2/stdsyms/sideeffect` (dá aos plugins o pacote `expvar`, usado por
  `prometheus/client_golang`); `settings.mounts` passa a valer para plugins Yaegi v2 (`dir` ou `dir:ro`; `host:guest`
  só quando os dois caminhos são iguais) e, quando definido, limita os diretórios que o plugin alcança; sem
  `mounts`, o plugin continua vendo o sistema de arquivos inteiro.
- `pkg/plugins/types.go`: nova opção `settings.fileMapping` (flock e mmap nos próprios arquivos), que só vale com
  `useUnsafe`.
- Documentação: `experimental/plugins.md` e `configuration-options.md`; testes em `yaegiv2_test.go`.

## Impacto no Traefix
- `pkg/plugins/` foi alterado pelos imports #14134 (`f3b101453`, logger dos middlewares para plugins Yaegi) e #13006
  (`631ff7384`, cache do arquivo do plugin) e pela fase 4 (`5c011b2f3`: o middleware nativo `coraza` também é lido
  como `plugin.coraza` quando nenhum plugin com esse nome está declarado). Nada disso toca nas linhas da mudança.
  O Coraza é nativo e não passa pelo Yaegi.
- A dependência `traefik/yaegi/v2` não é um fork `traefix-*`, e o bump não toca quic-go, qpack nem `golang.org/x/net`.
  No worktree de teste, `go mod tidy -diff` não mostrou diferença e `scripts/traefix/check-forks.sh` passou.
- `configuration-options.md` tem muitas alterações do Traefix; o cherry-pick aplicou sem conflito.

## Plano de implementação
1. `git cherry-pick -x 3aefb4549`
2. `scripts/traefix/check-forks.sh` e `go mod tidy -diff`.
3. Testes: `go test ./pkg/plugins/...` (passou no worktree de teste).
4. Se a documentação de referência for gerada (`go run ./internal/`), conferir que `configuration-options.md` não muda.

## Critério de pronto
- `yaegi/v2` em `v2.0.0`; `mounts` e `fileMapping` funcionando para plugins Yaegi v2.
- `go test ./pkg/plugins/...`, `check-forks.sh` e `make lint` sem erros.
