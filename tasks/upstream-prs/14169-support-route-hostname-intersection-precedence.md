# Support route hostname intersection precedence

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14169](https://github.com/traefik/traefik/pull/14169) |
| Tipo | Recurso |
| Área | provider/kubernetes/gateway |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | rtribotte |
| Aberto em | 2026-10-28 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `v3.8` |
| Tamanho | +1090/-286 em 19 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/provider/k8s/gatewayapi |
| Status | pendente |

## O que faz

Implementa a feature estendida GatewayRouteHostnameIntersectionPrecedence da Gateway API, que define a precedência entre rotas anexadas ao mesmo listener quando os hostnames se cruzam. Inclui atualização dos relatórios de conformidade e docs.

## Riscos e situação no upstream

Grande (+1090/-286, 19 arquivos), mexe no provider Gateway API e em relatórios de conformidade gerados. Base v3.8, então compatível com o fork. Status: needs-review. Só vale para quem usa Gateway API.

## Principais arquivos

- `pkg/provider/kubernetes/gateway/features.go`
- `pkg/provider/kubernetes/gateway/fixtures/httproute/with_same_hostname_intersection.yml`
- `integration/gateway_api_conformance_merged_test.go`
- `docs/content/migrate/v3.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14169/head:upstream-pr-14169`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
