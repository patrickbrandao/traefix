# 0006 — Remove duplicate Consul Catalog address check

- **Status:** aplicada (`1770b4b70`, 2026-11-23)
- **Tipo:** refactor
- **Prioridade:** baixa
- **Upstream:** traefik/traefik#14298 — commits `8d4921d71` (branch v3.7)
- **Aplica limpo:** sim
- **Conflito com Traefix:** não

## O que mudou no oficial
`pkg/provider/consulcatalog/config.go` (`addServerTCP`): apaga a segunda checagem `item.Address == ""`, repetida
depois da checagem da porta. Não muda o comportamento.

## Impacto no Traefix
O import do #13821 (`aaa4e59b1`, `allowEmptyServices`) alterou outra parte de `config.go`. Depois do cherry-pick,
a primeira checagem de endereço continua em `addServerTCP` (conferido no worktree de teste).

## Plano de implementação
1. `git cherry-pick -x 8d4921d71`
2. Nenhum ajuste.
3. Testes: `go test ./pkg/provider/consulcatalog/`

## Critério de pronto
- `addServerTCP` tem só uma checagem de endereço vazio; `go test ./pkg/provider/consulcatalog/` passa.
