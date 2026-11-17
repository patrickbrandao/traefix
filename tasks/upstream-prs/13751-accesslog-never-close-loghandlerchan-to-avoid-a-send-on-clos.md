# accesslog: never close logHandlerChan to avoid a send on closed channel

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13751](https://github.com/traefik/traefik/pull/13751) |
| Tipo | Correção |
| Área | middleware/accesslog, logs |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | euginfrancis |
| Aberto em | 2026-08-22 (última atualização: 2026-08-24) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +127/-6 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, area/logs, size/M, area/middleware |
| Status | pendente |

## O que faz

Corrige panic 'send on closed channel' no shutdown quando accessLog.bufferingSize > 0 e há uma conexão hijacked (WebSocket) ainda viva. O canal de log deixa de ser fechado; o autor reproduziu 485 panics em 3001 conexões no v3.7 e zero após o patch.

## Riscos e situação no upstream

Base v3.7, precisa de adaptação para master. Mudança pequena e bem reproduzida. Status/2-needs-review, review não verificado. Relacionado a PRs fechados como duplicatas (#13696, #13698).

## Principais arquivos

- `pkg/middlewares/accesslog/logger.go`
- `pkg/middlewares/accesslog/logger_close_race_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13751/head:upstream-pr-13751`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
