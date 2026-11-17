# Make use-regex redirects case insensitive

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14171](https://github.com/traefik/traefik/pull/14171) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | lazerg |
| Aberto em | 2026-10-28 (última atualização: 2026-10-28) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +3/-3 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Com use-regex ativo, o RedirectRegex de permanent-redirect e temporal-redirect ganha (?i), igual ao router. Antes, /DEV casava com o router, mas não com o redirect, e caía no backend. Agora o comportamento bate com o NGINX.

## Riscos e situação no upstream

Mudança de uma linha de regex em middleware.go, com teste. Status: needs-review. Docs não atualizadas, mas não há mudança de config.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/middleware.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14171/head:upstream-pr-14171`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
