# Emit recovery panic message and stack trace as a single log event

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13971](https://github.com/traefik/traefik/pull/13971) |
| Tipo | Correção |
| Área | middleware/recovery |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | mdabydeen |
| Aberto em | 2026-09-27 (última atualização: 2026-10-31) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +39/-2 em 2 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/S, area/middleware |
| Status | pendente |

## O que faz

Junta a mensagem de panic e o stack trace do middleware traefik-internal-recovery em um unico evento de log, em vez de dois. Evita que o trace se perca quando o processo termina logo apos o panic (ex.: shutdown com WebSocket).

## Riscos e situação no upstream

Escopo pequeno (2 arquivos, +39/-2) com teste de regressao. Label status/1-needs-design-review; maintainer pediu confirmacao do bug. Sem reviews formais ainda.

## Principais arquivos

- `pkg/middlewares/recovery/recovery.go`
- `pkg/middlewares/recovery/recovery_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13971/head:upstream-pr-13971`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
