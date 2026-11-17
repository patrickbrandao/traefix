# fix(rate_limiter): close Redis clients after cleanup

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13676](https://github.com/traefik/traefik/pull/13676) |
| Tipo | Correção |
| Área | middleware/ratelimiter (backend Redis, vazamento de goroutines) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | 0011001011 |
| Aberto em | 2026-08-11 (última atualização: 2026-08-11) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +34/-3 em 2 arquivos |
| Labels | status/0-needs-triage, size/M, area/middleware |
| Status | pendente |

## O que faz

Fecha o cliente Redis quando um middleware de rate limit aposentado deixa de ser alcançável, porque cada reload criava um cliente novo que nunca era fechado, acumulando conexões e goroutines. O fechamento segue o ciclo de vida do limiter, e runtime.KeepAlive evita fechar durante um comando em andamento.

## Riscos e situação no upstream

Alternativa à #13619, que resolve o mesmo vazamento compartilhando clientes. Não portar os dois sem decidir. Status 0-needs-triage, sem comentário de maintainer. PR base v3.7.

## Principais arquivos

- `pkg/middlewares/ratelimiter/redis_limiter.go`
- `pkg/middlewares/ratelimiter/rate_limiter_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13676/head:upstream-pr-13676`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
