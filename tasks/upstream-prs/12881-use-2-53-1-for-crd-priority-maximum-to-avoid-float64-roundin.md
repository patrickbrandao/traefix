# Use 2^53-1 for CRD priority maximum to avoid float64 rounding

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12881](https://github.com/traefik/traefik/pull/12881) |
| Tipo | Correção |
| Área | provider/kubernetes-crd |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | boinger |
| Aberto em | 2026-03-26 (última atualização: 2026-09-25) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +8/-8 em 6 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/provider/k8s/crd |
| Status | pendente |

## O que faz

Reduz o máximo de priority em IngressRoute e IngressRouteTCP para 2^53-1, valor que sobrevive ao arredondamento float64. Sem isso, ferramentas como ArgoCD veem o CRD sempre fora de sincronia. Corrige a issue #12693.

## Riscos e situação no upstream

Pequeno (6 arquivos, só valores de CRD). Precisa regenerar os schemas com make generate-crd. Status review sem maintainer comentando.

## Principais arquivos

- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/ingressroute.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/ingressroutetcp.go`
- `docs/content/reference/dynamic-configuration/traefik.io_ingressroutes.yaml`
- `docs/content/reference/dynamic-configuration/traefik.io_ingressroutetcps.yaml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12881/head:upstream-pr-12881`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
