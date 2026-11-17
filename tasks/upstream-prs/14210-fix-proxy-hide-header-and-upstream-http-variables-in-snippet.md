# Fix proxy_hide_header and upstream_http variables in snippets

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14210](https://github.com/traefik/traefik/pull/14210) |
| Tipo | Correção |
| Área | middleware/ingressnginx-snippet |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | locker95 |
| Aberto em | 2026-11-03 (última atualização: 2026-11-16) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +72/-14 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

proxy_hide_header em snippets do ingress-nginx nunca removia nada, porque o hook era registrado só para um tipo de writer que não chegava. Agora ele atua sobre os headers do upstream antes do add_header. $upstream_http_* resolve contra os headers do upstream, e add_header é pulado quando o valor fica vazio.

## Riscos e situação no upstream

Muda a ordem de aplicação de headers em snippets (hide antes de add). Status: needs-review. Corrige CORS duplicado em migração de ingress-nginx.

## Principais arquivos

- `pkg/middlewares/ingressnginx/snippet/action.go`
- `pkg/middlewares/ingressnginx/snippet/collector.go`
- `pkg/middlewares/ingressnginx/snippet/snippet.go`
- `pkg/middlewares/ingressnginx/snippet/snippet_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14210/head:upstream-pr-14210`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
