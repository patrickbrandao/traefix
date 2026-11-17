# Fix nil EndpointSlice ready condition

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14223](https://github.com/traefik/traefik/pull/14223) |
| Tipo | Correção |
| Área | provider/kubernetes/crd, provider/kubernetes/gateway |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | zl86790 |
| Aberto em | 2026-11-04 (última atualização: 2026-11-10) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +82/-3 em 5 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/crd, area/provider/k8s/gatewayapi |
| Status | pendente |

## O que faz

EndpointSlice com condição ready nula era tratado como não pronto nos providers TCP, UDP e Gateway API. Pela API do Kubernetes, nil significa desconhecido e deve ser tratado como pronto, como já fazem os providers HTTP. Corrige endpoints de Services sem seletor.

## Riscos e situação no upstream

Pequena mudança em três pontos de chamada, com testes. Status: needs-review, base v3.7. Pode mudar o conjunto de backends em EndpointSlices gerenciadas externamente.

## Principais arquivos

- `pkg/provider/kubernetes/crd/kubernetes_tcp.go`
- `pkg/provider/kubernetes/crd/kubernetes_udp.go`
- `pkg/provider/kubernetes/gateway/kubernetes.go`
- `pkg/provider/kubernetes/crd/kubernetes_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14223/head:upstream-pr-14223`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
