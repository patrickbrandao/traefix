# feat(rate_limiter): add denyOnError option to Redis rate limiter

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13529](https://github.com/traefik/traefik/pull/13529) |
| Tipo | Recurso |
| Área | middleware/ratelimiter (backend Redis) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | ayoubbts |
| Aberto em | 2026-07-16 (última atualização: 2026-11-03) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +154/-10 em 11 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/middleware, contributor/waiting-for-feedback |
| Status | pendente |

## O que faz

Adiciona a opção denyOnError na configuração Redis do rate limiter. Com false, quando o Redis está indisponível o erro é logado e a requisição segue (fail-open). O default continua true (fail-closed, comportamento atual). Resolve a issue #12043.

## Riscos e situação no upstream

Muda a config dinâmica e os CRDs (inclui arquivos gerados). Fail-open é decisão de segurança. Maintainers pediram que #13916 (deyanp) supersede este PR, e o autor ainda não respondeu. Status contributor/waiting-for-feedback. Mesmo PR toca o mesmo rate limiter que #13619 e #13676.

## Principais arquivos

- `pkg/middlewares/ratelimiter/rate_limiter.go`
- `pkg/middlewares/ratelimiter/rate_limiter_test.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/middleware.go`
- `docs/content/reference/dynamic-configuration/traefik.io_middlewares.yaml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13529/head:upstream-pr-13529`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
