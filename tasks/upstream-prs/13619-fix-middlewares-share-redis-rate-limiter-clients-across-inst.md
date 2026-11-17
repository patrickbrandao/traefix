# fix(middlewares): share redis rate-limiter clients across instances

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13619](https://github.com/traefik/traefik/pull/13619) |
| Tipo | Correção |
| Área | middleware/ratelimiter (backend Redis, vazamento de goroutines) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | funkypenguin |
| Aberto em | 2026-08-01 (última atualização: 2026-08-20) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +168/-1 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/middleware, ai/review |
| Status | pendente |

## O que faz

Cada reload da configuração criava um novo cliente go-redis por instância do rate limiter, e os antigos nunca eram fechados, o que faz goroutines e memória crescerem sem parar. A proposta é um cache de clientes indexado pela config Redis e um fingerprint dos materiais TLS, para que certificados rotacionados gerem cliente novo. Resolve #13618.

## Riscos e situação no upstream

Alternativa à #13676 (que fecha clientes em vez de compartilhá-los). Os dois PRs tocam o mesmo arquivo e não devem ser portados juntos sem decidir. Status review, com review COMMENTED.

## Principais arquivos

- `pkg/middlewares/ratelimiter/redis_limiter.go`
- `pkg/middlewares/ratelimiter/redis_limiter_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13619/head:upstream-pr-13619`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
