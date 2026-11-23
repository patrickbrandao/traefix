# 0011 — Support the ^~ location modifier in ingress-nginx server-snippet

- **Status:** aplicada (`f738ad60e`, 2026-11-23)
- **Tipo:** bugfix
- **Prioridade:** baixa
- **Upstream:** traefik/traefik#13904 — commits `4ee187fb4` (branch v3.7)
- **Aplica limpo:** não (arquivos em conflito: `pkg/middlewares/ingressnginx/snippet/collector.go`, `docs/content/reference/routing-configuration/kubernetes/ingress-nginx.md`)
- **Conflito com Traefix:** não (conflito textual só com PRs do oficial já importados)

Corresponde à task `tasks/upstream-prs/13904-support-the-location-modifier-in-ingress-nginx-server-snippe.md`
(P3, status "pendente", nunca importada). O PR foi mergeado no oficial, e esta task a substitui.

## O que mudou no oficial
`buildLocationMatcher` em `collector.go`: um `location ^~ /caminho` do `server-snippet` passa a ser tratado como
prefixo, sem o modificador (antes virava o prefixo literal `^~/caminho` e nunca casava). A precedência de seleção de
location do NGINX continua não implementada; a documentação diz isso. Três testes novos em `snippet_test.go`.

## Impacto no Traefix
`collector.go` foi alterado pelos imports #14210 (`bcf93fdac`), #14058 (`8231a3514`) e #13770 (`dd4a1e4c5`); no ponto
do conflito, o Traefix tira as aspas do padrão (`trimQuote`, import #14044) e compara com
`ingressnginx.OriginalPath(req)` (import #13770). A linha da documentação foi reescrita pelos imports #14058
(`grpc_set_header`, `$authHeaderN`) e #14124 (`server-alias`).

## Plano de implementação
1. `git cherry-pick -x 4ee187fb4`
2. Resolver `collector.go` combinando os dois lados: `TrimSpace`, depois tirar o `^~` com `strings.CutPrefix` (e
   `TrimSpace` de novo), depois `trimQuote`, mantendo o `HasPrefix(ingressnginx.OriginalPath(req), pathPattern)` do
   Traefix. Assim `location ^~ '/x'` também funciona.
3. Resolver a documentação mantendo as linhas do Traefix (`server-alias`, `server-snippet` e
   `configuration-snippet` com `grpc_set_header` e `$authHeaderN`) e acrescentando à linha do `server-snippet` a
   frase do oficial: "Location blocks are evaluated independently in declaration order. The `^~` modifier is
   supported as a prefix match, without NGINX location-selection precedence."
4. Testes: `go test ./pkg/middlewares/ingressnginx/...` (passou no worktree de teste com essa resolução).
5. Marcar a task 13904 de `tasks/upstream-prs/` como absorvida.

## Critério de pronto
- `location ^~ /internal` casa `/internal/...`; os testes novos do oficial e os dos imports passam.
- A documentação descreve o `^~` sem perder as linhas do Traefix.
