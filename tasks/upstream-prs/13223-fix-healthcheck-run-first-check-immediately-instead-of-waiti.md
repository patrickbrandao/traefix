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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:

- O bug é real. Cada reload recria os balanceadores com todos os servidores UP, e a primeira checagem só acontece depois de um `interval` inteiro. Durante essa janela, servidores DOWN recebem tráfego (502/504). Se os reloads forem mais frequentes que o `interval` (Docker gerando eventos, por exemplo), um servidor morto nunca chega a ser checado.
- A troca de `time.Ticker` por `time.Timer(0)` + `Reset(interval)` está correta. Efeito colateral aceitável: o intervalo passa a contar a partir do fim de cada ciclo de checagem, e não do início. Na prática isso evita ciclos sobrepostos (o ticker já descartava ticks atrasados).
- O PR deixava de fora: `ServiceTCPHealthChecker.Launch` também não esperava a goroutine do loop "unhealthy", que podia continuar viva depois do reload.
- O teste TCP `TestServiceTCPHealthChecker_Launch` dependia da primeira checagem só acontecer depois de um intervalo. Com a checagem imediata, ele discava antes de o mock abrir o listener e quebrava de forma intermitente (o caso TLS falhava sempre).

Ajustes:

- Integração com o #12485: o `timer.Reset` fica depois de `checkTargets`, que só retorna quando todas as checagens do ciclo terminam.
- `ServiceTCPHealthChecker.Launch` agora espera os dois loops (`sync.WaitGroup`), como já tinha sido feito no HTTP no #12485.
- Mock `sequencedTCPServer`: `Next()` agora é síncrono e só retorna quando a sequência está pronta (listener aberto ou fechado). O teste chama `Launch` depois do primeiro `Next()`.

Testes:

- `go test -race -count=2 ./pkg/healthcheck/`: ok. O teste novo `TestServiceHealthChecker_ImmediateFirstCheck` passa.
- `go test -race -count=20 -run TestServiceTCPHealthChecker_Launch`: 0 falhas. A referência sem o PR também teve 0 falhas em 10 execuções.
- `go build ./pkg/...`: ok.
- `golangci-lint run ./pkg/healthcheck/...`: 0 issues.
