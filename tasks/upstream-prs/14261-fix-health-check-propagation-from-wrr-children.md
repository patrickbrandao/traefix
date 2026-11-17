# Fix health check propagation from WRR children

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14261](https://github.com/traefik/traefik/pull/14261) |
| Tipo | Correção |
| Área | provider/kubernetes-crd (WRR, health check) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | rnikhori |
| Aberto em | 2026-11-10 (última atualização: 2026-11-10) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +160/-4 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/crd |
| Status | pendente |

## O que faz

Quando um IngressRoute tem vários services (ou TraefikService Weighted), o WRR pai nunca recebia o status dos filhos com healthCheck, então um backend DOWN continuava recebendo tráfego. O PR ativa HealthCheck no WRR gerado quando todos os filhos são Services Kubernetes com healthCheck.

## Riscos e situação no upstream

Base é v3.7, precisa de port para o master. Só altera o comportamento quando todos os filhos são Services Kubernetes com healthCheck, o que limita o escopo. Sem review de maintainer ainda (status/2-needs-review).

## Principais arquivos

- `pkg/provider/kubernetes/crd (geração do WRR pai)`
- `pkg/server/service/service.go (getWRRServiceHandler)`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14261/head:upstream-pr-14261`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
