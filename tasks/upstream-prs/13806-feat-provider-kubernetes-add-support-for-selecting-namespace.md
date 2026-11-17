# feat(provider-kubernetes): add support for selecting namespaces based on a label selector opposed to a hard coded list

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13806](https://github.com/traefik/traefik/pull/13806) |
| Tipo | Recurso |
| Área | provider/kubernetes (ingress, crd, gateway, knative) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | HowDoCode |
| Aberto em | 2026-08-30 (última atualização: 2026-09-17) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +441/-19 em 11 arquivos |
| Labels | status/0-needs-triage, area/provider/k8s/ingress, size/L, area/provider/k8s/crd, area/provider/k8s, area/provider/k8s/gatewayapi, area/provider/k8s/knative |
| Status | pendente |

## O que faz

Permite escolher os namespaces observados por label selector, em vez de listar namespaces fixos. Replica a implementação do ingress-nginx e carrega os namespaces por um helper novo, sem mudar a assinatura de WatchAll. Não adiciona inclusão ou remoção dinâmica de namespaces.

## Riscos e situação no upstream

441 linhas em 11 arquivos, nos providers ingress, crd, gateway e knative. Status status/0-needs-triage, sem review. O autor disse que a versão dinâmica fica para o #13713, então os dois PRs se sobrepõem em conceito.

## Principais arquivos

- `pkg/provider/kubernetes/crd/client.go`
- `pkg/provider/kubernetes/crd/client_test.go`
- `pkg/provider/kubernetes/crd/kubernetes.go`
- `pkg/provider/kubernetes/gateway/client.go`
- `pkg/provider/kubernetes/gateway/client_test.go`
- `pkg/provider/kubernetes/gateway/kubernetes.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13806/head:upstream-pr-13806`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
