# Add denyOnError option to the Redis rate limiter

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13916](https://github.com/traefik/traefik/pull/13916) |
| Tipo | Recurso |
| Área | middleware/ratelimit-redis |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | deyanp |
| Aberto em | 2026-09-16 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +221/-10 em 16 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/middleware |
| Status | pendente |

## O que faz

Adiciona a opção denyOnError ao rate limiter com Redis. Com true (padrão, como hoje) erros do Redis geram 500; com false, o erro é logado e a requisição passa sem limite. Evita que uma falha do store derrube toda a API quando o rate limit serve para conter abuso.

## Riscos e situação no upstream

Retoma o PR #13529 (commits do autor original preservados). Corrige default no provider REST, que não chama SetDefaults. Toca 16 arquivos, incluindo CRDs gerados (exige make generate-crd). Status needs-review, sem review ainda.

## Principais arquivos

- `pkg/middlewares/ratelimiter/rate_limiter.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/middleware.go`
- `docs/content/reference/routing-configuration/http/middlewares/ratelimit.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13916/head:upstream-pr-13916`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
