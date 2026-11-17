# Add a scope option to the rate limit middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13823](https://github.com/traefik/traefik/pull/13823) |
| Tipo | Recurso |
| Área | middleware/ratelimit |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | deyanp |
| Aberto em | 2026-09-02 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +403/-14 em 18 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/docker, size/L, area/middleware, area/provider/k8s/crd, area/provider/k8s |
| Status | pendente |

## O que faz

Novo campo scope em rateLimit (router ou middleware) define em que escopo os buckets são compartilhados. Resolve a divergência entre memória e Redis apontada no #13706, sem mudar o comportamento padrão de nenhum dos dois. Redis rejeita scope router.

## Riscos e situação no upstream

O próprio autor diz que o design não está fechado: pode virar alinhamento direto da memória com o Redis. Dois reviews COMMENTED. Toca CRD, docs e fixtures. Dependência de aprovação de CI.

## Principais arquivos

- `pkg/config/dynamic/middlewares.go`
- `pkg/middlewares/ratelimiter/bucket_registry.go`
- `pkg/middlewares/ratelimiter/rate_limiter.go`
- `pkg/middlewares/ratelimiter/rate_limiter_test.go`
- `pkg/provider/kubernetes/crd/generated/applyconfiguration/traefikio/v1alpha1/ratelimit.go`
- `pkg/provider/kubernetes/crd/kubernetes.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13823/head:upstream-pr-13823`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
