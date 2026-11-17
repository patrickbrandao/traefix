# Add support for ServiceExternalTrafficPolicyTypeLocal

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12048](https://github.com/traefik/traefik/pull/12048) |
| Tipo | Recurso |
| Área | provider/kubernetes/crd |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | deveshk0 |
| Aberto em | 2025-09-08 (última atualização: 2026-08-31) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +495/-9 em 7 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/provider/k8s/crd, area/provider/k8s |
| Status | pendente |

## O que faz

Quando o Service tem externalTrafficPolicy Local, limita os nós do balanceamento aos que têm endpoints prontos, nos casos HTTP, TCP e UDP em NodePort. Usa EndpointSlice para saber a prontidão.

## Riscos e situação no upstream

Afeta provider CRD para HTTP/TCP/UDP (7 arquivos, +495/-9). Maintainers reconhecem como feature desejada, mas sem ETA e sem review ainda. Fixtures e testes incluídos.

## Principais arquivos

- `pkg/provider/kubernetes/crd/kubernetes_http.go`
- `pkg/provider/kubernetes/crd/kubernetes_tcp.go`
- `pkg/provider/kubernetes/crd/kubernetes_udp.go`
- `pkg/provider/kubernetes/crd/kubernetes_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12048/head:upstream-pr-12048`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
