# Feature/add gateway sticky session support

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12537](https://github.com/traefik/traefik/pull/12537) |
| Tipo | Recurso |
| Área | provider/kubernetes-gateway-api |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | americancode |
| Aberto em | 2026-01-09 (última atualização: 2026-10-09) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +3238/-61 em 34 arquivos |
| Labels | kind/enhancement, status/2-needs-review, contributor/waiting-for-corrections, size/L, area/provider/k8s/gatewayapi |
| Status | pendente |

## O que faz

Adiciona sessionPersistence (cookie ou header) em HTTPRoute, com sticky sessions pela Gateway API sem CRD específico do Traefik. Também consome o XBackendTrafficPolicy experimental e define a precedência com TraefikService.

## Riscos e situação no upstream

Muito grande (34 arquivos, +3238/-61, CRDs regenerados). Um maintainer avisou que a Gateway API v1.7.0-rc.1 deprecia sessionPersistence no HTTPRoute e no XBackendTrafficPolicy, então o desenho pode precisar de reescrita. Conflitos com master. Status contributor/waiting-for-corrections.

## Principais arquivos

- `pkg/provider/kubernetes/gateway/client.go`
- `pkg/config/dynamic/http_config.go`
- `docs/content/reference/routing-configuration/kubernetes/gateway-api.md`
- `docs/content/reference/install-configuration/providers/kubernetes/kubernetes-gateway.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12537/head:upstream-pr-12537`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
