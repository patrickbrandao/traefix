# Track the Kubernetes origin of dynamic configuration objects

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14278](https://github.com/traefik/traefik/pull/14278) |
| Tipo | Recurso |
| Área | provider/kubernetes (todos os providers k8s), observability |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | Neffi42 |
| Aberto em | 2026-11-12 (última atualização: 2026-11-13) |
| Draft | não |
| Branch-alvo | `v3.8` |
| Tamanho | +25895/-8032 em 51 arquivos |
| Labels | kind/enhancement, status/0-needs-triage, area/provider/k8s/ingress, size/L, area/middleware, area/provider/k8s/crd, area/provider/k8s, area/service, area/provider/k8s/gatewayapi, area/accesslogs, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Os providers Kubernetes (Ingress, ingress-nginx, CRD e Gateway API) passam a gravar em cada router, service, middleware e servers transport quais recursos Kubernetes os geraram, com group, version, kind, namespace, name, resourceVersion e JSON pointer. Ajuda a rastrear a origem de cada objeto de configuração.

## Riscos e situação no upstream

Enorme (+25895/-8032, 51 arquivos) e transversal a todos os providers e middlewares, com alto risco de conflito com o fork. Adiciona campo metadata à config dinâmica, o que muda a saída da API e os dados de integração. Status 0-needs-triage, sem review ainda.

## Principais arquivos

- `pkg/config/dynamic/http_config.go`
- `pkg/config/dynamic/metadata.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/config/dynamic/tcp_config.go`
- `pkg/config/dynamic/udp_config.go`
- `pkg/config/dynamic/tcp_middlewares.go`
- `integration/testdata/rawdata-*.json`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14278/head:upstream-pr-14278`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
