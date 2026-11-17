# Add allowCORSPreflight to BasicAuth middleware for CORS preflight

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14237](https://github.com/traefik/traefik/pull/14237) |
| Tipo | Recurso |
| Área | middleware/basicauth |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | k3a |
| Aberto em | 2026-11-06 (última atualização: 2026-11-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +121/-22 em 7 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/authentication |
| Status | pendente |

## O que faz

Adiciona a opção allowCORSPreflight ao BasicAuth, deixando passar sem credenciais os OPTIONS que têm Access-Control-Request-Method. Resolve o caso em que o preflight CORS de fetch com credenciais falha antes de chegar ao backend (issue #1019, antiga).

## Riscos e situação no upstream

Afeta a segurança: um OPTIONS sem autenticação passa pelo middleware quando o header de preflight existe, então a opção deve ser opt-in e revisada. Toca em config dinâmica e no provider CRD (kubernetes.go, middleware.go). Review: COMMENTED, sem aprovação.

## Principais arquivos

- `pkg/middlewares/auth/basic_auth.go`
- `pkg/middlewares/auth/basic_auth_test.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/middleware.go`
- `pkg/provider/kubernetes/crd/kubernetes.go`
- `docs/content/reference/routing-configuration/http/middlewares/basicauth.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14237/head:upstream-pr-14237`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
