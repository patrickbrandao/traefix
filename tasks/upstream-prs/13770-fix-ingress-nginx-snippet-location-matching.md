# Fix ingress-nginx snippet location matching

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13770](https://github.com/traefik/traefik/pull/13770) |
| Tipo | Correção |
| Área | middleware/snippet, provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | parrahex |
| Aberto em | 2026-08-25 (última atualização: 2026-08-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +115/-7 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/middleware, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Dois ajustes para igualar ao ingress-nginx em snippets de location: operandos entre aspas deixam de incluir as aspas no padrão, e o match usa o path original da requisição e não o path já reescrito por rewrite-target. Corrige o #13690.

## Riscos e situação no upstream

Base v3.7, precisa de adaptação. Mudança de comportamento no middleware de rewrite e de snippet, o que pode alterar configs existentes que dependiam do path reescrito. Status/2-needs-review. Autor validou os testes contra o base sem o patch.

## Principais arquivos

- `pkg/middlewares/ingressnginx/request.go`
- `pkg/middlewares/ingressnginx/rewritetarget/rewrite_target.go`
- `pkg/middlewares/ingressnginx/snippet/collector.go`
- `pkg/middlewares/ingressnginx/snippet/snippet_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13770/head:upstream-pr-13770`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
