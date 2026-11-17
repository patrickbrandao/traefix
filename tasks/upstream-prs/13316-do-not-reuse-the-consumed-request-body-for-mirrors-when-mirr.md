# Do not reuse the consumed request body for mirrors when mirrorBody is false

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13316](https://github.com/traefik/traefik/pull/13316) |
| Tipo | Correção |
| Área | middleware/mirroring |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | JATINKUMAR1008 |
| Aberto em | 2026-06-06 (última atualização: 2026-09-23) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +77/-3 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | pendente |

## O que faz

Com mirrorBody: false, os clones espelhados passam a ter http.NoBody, ContentLength zero, TransferEncoding limpo e sem Content-Length, em vez de reusar o body já consumido e fechado pelo handler principal. Corrige 502 falso no mirror ('invalid Read on closed Body') que polui métricas e traces. Fixes #13292.

## Riscos e situação no upstream

PR base v3.7 (portar para master). Review APPROVED. Testes de regressão para body de tamanho conhecido e chunked. Escopo pequeno.

## Principais arquivos

- `pkg/server/service/loadbalancer/mirror/mirror.go`
- `pkg/server/service/loadbalancer/mirror/mirror_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13316/head:upstream-pr-13316`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
