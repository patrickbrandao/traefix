# Support new connections to terminating endpoints for zero-downtime

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12419](https://github.com/traefik/traefik/pull/12419) |
| Tipo | Correção |
| Área | server/loadbalancer/wrr, provider/kubernetes |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | PartlyFluked |
| Aberto em | 2025-12-13 (última atualização: 2025-12-15) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +306/-15 em 4 arquivos |
| Labels | status/1-needs-design-review, size/L, area/server, area/provider/k8s |
| Status | pendente |

## O que faz

Quando todos os endpoints do Kubernetes estão terminando mas ainda servindo (serving=true), o WRR passa a mandar novas conexões para eles em vez de responder 503. Isso cobre deploys com réplica única ou estratégia Recreate, onde hoje há downtime.

## Riscos e situação no upstream

Muda o comportamento de roteamento do WRR no caminho de falha. Status design-review (maintainers vão checar impacto antes de seguir). Toca docs de Kubernetes CRD e Ingress.

## Principais arquivos

- `pkg/server/service/loadbalancer/wrr/wrr.go`
- `pkg/server/service/loadbalancer/wrr/wrr_test.go`
- `docs/content/routing/providers/kubernetes-crd.md`
- `docs/content/routing/providers/kubernetes-ingress.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12419/head:upstream-pr-12419`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
