# ForwardAuth: Don't treat client-canceled context as error in logs/tracing

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13655](https://github.com/traefik/traefik/pull/13655) |
| Tipo | Correção |
| Área | middleware/forwardauth (logs e tracing) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | djfinnoy |
| Aberto em | 2026-08-07 (última atualização: 2026-08-11) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +26/-6 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware, area/authentication |
| Status | importado com ajustes |

## O que faz

Cancelamento de contexto pelo cliente era logado e marcado como erro no tracing do ForwardAuth, poluindo o log e ocultando erros reais. Agora usa log de debug, inclui o status HTTP nos logs de transporte e não marca o span como erro.

## Riscos e situação no upstream

Muda nível de log e marcação de span, o que afeta alertas que dependem desses sinais. Status needs-review, sem review ainda. PR base v3.7.

## Principais arquivos

- `pkg/middlewares/auth/forward.go`
- `pkg/middlewares/auth/forward_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13655/head:upstream-pr-13655`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

Veredito: **importado com ajustes**.

Achados:
- PR aberto, sem review de maintainer. Só houve review do Copilot, e o comentário dele (marcar o span como erro) já foi corrigido pelo autor. A mudança é pequena e não afeta a segurança: só o nível de log e o status do span mudam quando o cliente cancela. O código de resposta (499) e a decisão de autenticação continuam iguais.
- O teste do PR era vazio. Ele verificava o status de `mockTracer.spans[1]` (o span `AuthRequest`), mas `observability.SetStatusErrorf` marca o span do contexto da requisição (`spans[0]`). Por isso o teste passava mesmo sem a correção.
- Efeito colateral a conhecer: alertas baseados no log `error` "Error calling" deixam de disparar quando o cliente cancela. Esse é o objetivo do PR.

Ajustes:
- O teste passa a verificar o span da requisição (`spans[0]`) e agora falha sem a correção.
- Novo teste `TestForwardAuthForwardErrorTracing`: um erro real de transporte (conexão recusada) continua logado em `error`, com `statusCode` 500, e marca o span como erro.
- Comentário de porquê no ramo do cancelamento.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/middlewares/auth/`: ok.
- `golangci-lint run ./pkg/middlewares/auth/...`: 0 issues.
