# fix(healthcheck): run first check immediately instead of waiting one interval

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13223](https://github.com/traefik/traefik/pull/13223) |
| Tipo | Correção |
| Área | healthcheck |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | filip2mac |
| Aberto em | 2026-05-21 (última atualização: 2026-06-17) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +60/-6 em 3 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/S, area/healthcheck |
| Status | pendente |

## O que faz

Cada reload de configuração recria os load balancers com todos os servers marcados como UP, e o primeiro health check só roda após um intervalo inteiro. Isso gera 502s para servers que já estão DOWN. O PR troca o NewTicker por NewTimer(0) para a primeira checagem rodar logo após o Launch, nos checkers HTTP e TCP. Relata o issue #13218.

## Riscos e situação no upstream

Label needs-design-review, com a análise pedida pelo maintainer. Mudança pequena (+60/-6) em healthcheck.go e healthcheck_tcp.go. Em clusters grandes com reloads frequentes, pode gerar pico de checagens; vale testar.

## Principais arquivos

- `pkg/healthcheck/healthcheck.go`
- `pkg/healthcheck/healthcheck_tcp.go`
- `pkg/healthcheck/healthcheck_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13223/head:upstream-pr-13223`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
