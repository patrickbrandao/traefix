# Add ExternalName service support for KubernetesGateway provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13050](https://github.com/traefik/traefik/pull/13050) |
| Tipo | Recurso |
| Área | provider/kubernetes/gateway-api |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | aniskhalfallah |
| Aberto em | 2026-04-26 (última atualização: 2026-08-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +347/-25 em 4 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/provider/k8s/gatewayapi |
| Status | pendente |

## O que faz

Permite usar Services do tipo ExternalName como backend no provider KubernetesGateway, com o mesmo comportamento dos providers Ingress e CRD. Reutiliza a flag allowExternalNameServices, que é opt-in. Resolve o issue #12950.

## Riscos e situação no upstream

Label status/2-needs-review; usuários aguardam há meses e cobram atualização. A flag é opt-in e já existe nos outros providers, então o risco de segurança está contido. Toca apenas gateway/kubernetes.go e testes.

## Principais arquivos

- `pkg/provider/kubernetes/gateway/kubernetes.go`
- `pkg/provider/kubernetes/gateway/kubernetes_test.go`
- `pkg/provider/kubernetes/gateway/fixtures/httproute/with_external_name_http.yml`
- `pkg/provider/kubernetes/gateway/fixtures/httproute/with_external_name_https.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13050/head:upstream-pr-13050`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
