# fix(gateway): resolve TLSOptions references on Gateway API listeners

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13864](https://github.com/traefik/traefik/pull/13864) |
| Tipo | Recurso |
| Área | provider/kubernetes-gateway, tls |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | sk-sanjay01 |
| Aberto em | 2026-09-08 (última atualização: 2026-09-10) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +792/-34 em 13 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/tls, area/provider/k8s/gatewayapi |
| Status | pendente |

## O que faz

Faz o provider Gateway API ler a extensão de TLSOptions de cada listener (tls.traefik.io/tlsoptions.name e .namespace) e resolvê-la para uma referência a TLSOption CRD, com autorização via ReferenceGrant. Antes, todo router TLS ficava com Options vazio e caía no TLSOption default.

## Riscos e situação no upstream

Diff grande (+792/-34, 13 arquivos) que toca as quatro rotas (HTTP, gRPC, TCP, TLS). Label status/1-needs-design-review; maintainers aplicaram design-review e não há review formal ainda. Pode exigir ajustes de RBAC (kubernetes-gateway-rbac.yml).

## Principais arquivos

- `pkg/provider/kubernetes/gateway/client.go`
- `pkg/provider/kubernetes/gateway/httproute.go`
- `pkg/provider/kubernetes/gateway/grpcroute.go`
- `pkg/provider/kubernetes/gateway/tcproute.go`
- `pkg/provider/kubernetes/gateway/tlsroute.go`
- `pkg/provider/kubernetes/gateway/kubernetes.go`
- `docs/content/reference/dynamic-configuration/kubernetes-gateway-rbac.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13864/head:upstream-pr-13864`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
