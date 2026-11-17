# Add TCP health check support for ExternalName services

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12290](https://github.com/traefik/traefik/pull/12290) |
| Tipo | Recurso |
| Área | healthcheck/tcp, provider/k8s/crd |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | kyounghoonJang |
| Aberto em | 2025-11-19 (última atualização: 2026-10-22) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +461/-2 em 11 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/healthcheck, area/tcp, area/provider/k8s/crd |
| Status | pendente |

## O que faz

Adiciona health check TCP para serviços ExternalName em IngressRouteTCP, paridade com o que já existe no IngressRoute HTTP. Resolve a issue 12181.

## Riscos e situação no upstream

11 arquivos (+461/-2). Docs não atualizadas. Status needs-review, sem reviews.

## Principais arquivos

- `pkg/provider/kubernetes/crd/kubernetes_tcp.go`
- `pkg/config/dynamic/tcp_config.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/ingressroutetcp.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12290/head:upstream-pr-12290`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
