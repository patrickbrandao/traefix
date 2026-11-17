# Keep canary routers at their location priority on ingress-nginx provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13789](https://github.com/traefik/traefik/pull/13789) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | lazerg |
| Aberto em | 2026-08-27 (última atualização: 2026-10-12) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +176/-0 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Routers de canary herdam a regra de location com a condição de canary e ganham prioridade pelo tamanho da regra. Assim, um canary em / pode vencer um path mais específico como /data/ e mandar requisições do cookie/header para o backend errado. O patch fixa a prioridade em location + 1.

## Riscos e situação no upstream

Base v3.7, precisa de adaptação. Pequeno. Corrige #13785. Status/2-needs-review. O autor aponta um caso residual: canary em / fica uma posição acima de PathPrefix('/'), empatando com Exact.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `pkg/provider/kubernetes/ingress-nginx/translator.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13789/head:upstream-pr-13789`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
