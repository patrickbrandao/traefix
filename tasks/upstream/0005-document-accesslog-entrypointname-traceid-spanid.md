# 0005 — Document entryPointName, trace_id and span_id JSON access log fields

- **Status:** aplicada (`ceb036373`, 2026-11-23)
- **Tipo:** bugfix
- **Prioridade:** baixa
- **Upstream:** traefik/traefik#14302 — commits `0a248eccf` (branch v3.7)
- **Aplica limpo:** sim
- **Conflito com Traefix:** não

## O que mudou no oficial
Só documentação: `docs/content/reference/install-configuration/observability/logs-and-accesslogs.md` ganha
três linhas na tabela de campos do access log: `entryPointName`, `trace_id` e `span_id` (os dois últimos só
com tracing ligado). Os campos já existem no código.

## Impacto no Traefix
O Traefix alterou essa página (`44859b9ad`, `7cb3dbc7b`, `f11513bc0`, `c8004cba0`; inclui os campos
`Fingerprint*` da fase 3). As linhas novas entram em pontos da tabela que o Traefix não alterou. Os campos valem
para o access log do Traefix, inclusive o do entrypoint `traefix-mesh`.

## Plano de implementação
1. `git cherry-pick -x 0a248eccf`
2. Nenhum ajuste.
3. Conferir a página com `make docs-serve` (opcional).

## Critério de pronto
- A tabela de campos documenta `entryPointName`, `trace_id` e `span_id` e mantém os campos `Fingerprint*` do Traefix.
