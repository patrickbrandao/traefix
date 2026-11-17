# Add UDPRoute support to Kubernetes Gateway API provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12472](https://github.com/traefik/traefik/pull/12472) |
| Tipo | Recurso |
| Área | provider/kubernetes-gateway-api |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | fmauNeko |
| Aberto em | 2025-12-28 (última atualização: 2026-08-06) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +22493/-89 em 27 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/provider/k8s/gatewayapi |
| Status | pendente |

## O que faz

Adiciona suporte a UDPRoute (Gateway API v1, canal Standard) no provider de Gateway API, com reconciliação de rotas, backends, ReferenceGrant e status. Inclui teste de conformidade do perfil GATEWAY-UDP.

## Riscos e situação no upstream

Enorme (27 arquivos, +22493/-89, muitos são CRDs e relatórios de conformidade). Ainda usa v1alpha2 e precisa migrar para v1, o que o autor aceitou. Conflita com a master atual e exige RBAC novo no Helm chart. Sem review de maintainer.

## Principais arquivos

- `pkg/provider/kubernetes/gateway/kubernetes.go`
- `docs/content/reference/routing-configuration/kubernetes/gateway-api.md`
- `docs/content/reference/install-configuration/providers/kubernetes/kubernetes-gateway.md`
- `integration/fixtures/k8s-gateway/04-udproute.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12472/head:upstream-pr-12472`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
