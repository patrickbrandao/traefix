# add source-IP sticky routing as a TCP load balancer option

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13264](https://github.com/traefik/traefik/pull/13264) |
| Tipo | Recurso |
| Área | tcp/load-balancer |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | KaanSimsek |
| Aberto em | 2026-06-01 (última atualização: 2026-06-06) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +491/-4 em 14 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, area/sticky-session, size/L, area/tcp, area/provider/k8s/crd, area/provider/k8s, area/service |
| Status | pendente |

## O que faz

Adiciona o campo strategy ao load balancer TCP (wrr, padrão, ou ip-hash), com Rendezvous Hashing para que conexões do mesmo IP de origem caiam sempre no mesmo backend, o que favorece o reuso de sessão TLS. Exposto também no CRD IngressRouteTCP. Closes #10265.

## Riscos e situação no upstream

Toca o CRD e código gerado (applyconfiguration, YAML de CRD, docs). Status 1-needs-design-review: maintainers pediram checagem de impacto antes de avançar. Sem review. Tamanho L em 14 arquivos.

## Principais arquivos

- `pkg/config/dynamic/tcp_config.go`
- `pkg/server/service/tcp/service.go`
- `pkg/tcp/handler.go`
- `pkg/provider/kubernetes/crd/kubernetes_tcp.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/ingressroutetcp.go`
- `pkg/provider/kubernetes/crd/generated/applyconfiguration/traefikio/v1alpha1/servicetcp.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13264/head:upstream-pr-13264`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
