# Make rewrite-target replace the whole path

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14063](https://github.com/traefik/traefik/pull/14063) |
| Tipo | Correção |
| Área | provider/ingress-nginx, middleware/ingressnginx/rewritetarget |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | lazerg |
| Aberto em | 2026-10-13 (última atualização: 2026-10-22) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +41/-17 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Corrige rewrite-target do ingress-nginx para substituir o path inteiro pelo template expandido, como o rewrite do nginx, em vez de trocar so o trecho casado. Sem isso, /api/v1 com rewrite-target / virava //v1 e recebia 400. Resolve #14062.

## Riscos e situação no upstream

Pequeno (+41/-17). Muda comportamento para ingresses que dependiam do sufixo ser mantido (ex.: /foo/x indo para /bar/x passa a ir para /bar); o autor documenta isso como correto segundo o ingress-nginx. Um maintainer aprovou (APPROVED).

## Principais arquivos

- `pkg/middlewares/ingressnginx/rewritetarget/rewrite_target.go`
- `pkg/middlewares/ingressnginx/rewritetarget/rewrite_target_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14063/head:upstream-pr-14063`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
