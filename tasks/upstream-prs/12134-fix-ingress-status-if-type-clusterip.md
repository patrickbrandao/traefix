# Fix ingress status if type ClusterIP

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12134](https://github.com/traefik/traefik/pull/12134) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | walnuss0815 |
| Aberto em | 2025-10-07 (última atualização: 2026-09-22) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +2/-2 em 2 arquivos |
| Labels | status/1-needs-design-review, area/provider/k8s/ingress, kind/bug/fix, size/S |
| Status | rejeitado (superado por #13357) |

## O que faz

Quando o Service do Traefik é ClusterIP, o status do Ingress usa spec.clusterIPs em vez de externalIPs, que não servem para isso. Corrige o caso em que o ArgoCD não consegue verificar Ingresses.

## Riscos e situação no upstream

Pequeno (2 arquivos, +2/-2). Base é v3.7, precisa de port. Autor já rebaseou em v3.6 e cobra revisão. Status needs-design-review, sem reviews.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/fixtures/Published-Service-ClusterIP.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12134/head:upstream-pr-12134`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados**

- Troca `spec.externalIPs` por `spec.clusterIPs` no caso `ClusterIP`: quem configurou `externalIPs` num Service ClusterIP (caso suportado e documentado) passaria a ver IPs internos no status do Ingress, quebrando External-DNS e similares. É uma regressão do comportamento padrão.
- O próprio upstream aponta a sobreposição com #13357, que resolve o mesmo problema (ArgoCD preso em "Progressing") sem essa regressão.

**Decisão**

Rejeitado em favor de #13357, que mantém `externalIPs` e só cai para o ClusterIP quando não há `externalIPs`. A ideia útil deste PR (usar `clusterIPs`, cobrindo dual-stack) foi incorporada no ajuste de #13357.

**Testes**

Nenhum (nada importado).
