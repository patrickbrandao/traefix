# Warn when ingressEndpoint.ip is not an IP address

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13857](https://github.com/traefik/traefik/pull/13857) |
| Tipo | Recurso |
| Área | provider/kubernetes-ingress |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | datrixlab |
| Aberto em | 2026-09-07 (última atualização: 2026-09-10) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +73/-0 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/k8s/ingress, size/M |
| Status | pendente |

## O que faz

Emite warning no startup quando providers.kubernetesingress.ingressendpoint.ip não é um IPv4 ou IPv6 válido. Hoje o valor vai direto para status.loadBalancer de todos os Ingress sem validação, e a falha (flag sem valor que engole o próximo argumento) é difícil de diagnosticar.

## Riscos e situação no upstream

Só adiciona log; sem mudança de comportamento. Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/kubernetes_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13857/head:upstream-pr-13857`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
