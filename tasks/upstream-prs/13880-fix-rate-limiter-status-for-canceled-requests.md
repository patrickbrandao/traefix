# Fix rate limiter status for canceled requests

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13880](https://github.com/traefik/traefik/pull/13880) |
| Tipo | Correção |
| Área | middleware/ratelimit |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | lansman |
| Aberto em | 2026-09-09 (última atualização: 2026-10-12) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +66/-1 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | pendente |

## O que faz

Quando a espera de delay do rate limiter é interrompida por context.Canceled, responde 499 em vez de registrar 500. Afeta access log e métricas de status, não a resposta ao cliente que já cancelou. Segue o tratamento já existente no ForwardAuth.

## Riscos e situação no upstream

Mudança localizada com teste de regressão determinístico. Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/middlewares/ratelimiter/rate_limiter.go`
- `pkg/middlewares/ratelimiter/rate_limiter_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13880/head:upstream-pr-13880`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
