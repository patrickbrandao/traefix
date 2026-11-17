# NodePortLB support in kubernetes gateway provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12175](https://github.com/traefik/traefik/pull/12175) |
| Tipo | Recurso |
| Área | provider/kubernetes-gateway |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | jorisvergeer |
| Aberto em | 2025-10-21 (última atualização: 2025-10-30) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +820/-47 em 17 arquivos |
| Labels | status/1-needs-design-review, size/L, area/provider/k8s/gatewayapi |
| Status | pendente |

## O que faz

Adiciona balanceamento NodePort ao provider Gateway API, como já existe nos providers CRD e Ingress. Inclui opção para desligar a leitura de recursos Node, caso o RBAC não permita.

## Riscos e situação no upstream

Muda RBAC do Gateway API e a documentação de migração (17 arquivos, +820/-47). Autor pede segunda revisão do RBAC. Status needs-design-review, com dois comentários de review.

## Principais arquivos

- `pkg/provider/kubernetes/gateway/kubernetes.go`
- `pkg/provider/kubernetes/gateway/client.go`
- `pkg/provider/kubernetes/gateway/annotations.go`
- `docs/content/reference/dynamic-configuration/kubernetes-gateway-rbac.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12175/head:upstream-pr-12175`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
