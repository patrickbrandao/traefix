# 0002 — Apply the upstream vhost after the auth middleware

- **Status:** aplicada (`6fbc69724`, 2026-11-23)
- **Tipo:** bugfix
- **Prioridade:** baixa
- **Upstream:** traefik/traefik#13787 — commits `fab837419` (branch v3.7)
- **Aplica limpo:** não (arquivos em conflito: `pkg/provider/kubernetes/ingress-nginx/translator.go`, `pkg/middlewares/ingressnginx/snippet/snippet_test.go`)
- **Conflito com Traefix:** não (conflito textual só com PRs do oficial já importados)

## O que mudou no oficial
Provider ingress-nginx: o middleware `<router>-vhost` (anotação `upstream-vhost`) passa a entrar na cadeia depois
dos middlewares de autenticação (`-snippet`, que carrega o `auth-url`) e antes do `-retry`. Antes ele vinha antes
dos limites e da autenticação, então o `auth-signin` recebia o host já reescrito. Testes novos em
`translator_test.go` e `snippet_test.go`.

## Impacto no Traefix
`translator.go` foi alterado pelos imports do #13789 (`b536fe0eb`: a função virou `buildMiddlewareChain` com o
parâmetro `withLimits`, para os routers canary) e do #13801 (`1b38839c1`). `snippet_test.go` foi alterado pelos
imports #14210, #14058, #14044, #13770 e #13754. Nenhuma alteração própria do Traefix (fingerprint, malha, WAF)
toca nessas áreas. No merge `ae7f2047f`, o oficial resolveu o mesmo conflito da mesma forma, já com o
`withLimits`.

## Plano de implementação
1. `git cherry-pick -x fab837419`
2. Resolver `translator.go` em `buildMiddlewareChain`: manter o bloco `if withLimits { ... }` do Traefix,
   apagar o bloco `-vhost` que fica antes dele e deixar só o bloco novo do oficial, depois de `loc.SnippetAuth` e
   antes de `loc.Retry` (o `-vhost` continua fora do `withLimits`, valendo para os routers canary e os
   não-canary).
3. Resolver os imports de `snippet_test.go`: manter `ingressnginx` e `ingressnginx/rewritetarget` e acrescentar
   `ingressnginx/upstreamvhost`.
4. Testes: `go test ./pkg/middlewares/ingressnginx/... ./pkg/provider/kubernetes/ingress-nginx/...` (passaram
   no worktree de teste com essa resolução).

## Critério de pronto
- `-vhost` vem depois de `-snippet` e antes de `-retry` nas cadeias canary e não-canary.
- Testes do ingress-nginx e `make lint` sem erros.
