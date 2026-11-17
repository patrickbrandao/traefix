# Fix file provider middleware errors at startup

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14064](https://github.com/traefik/traefik/pull/14064) |
| Tipo | Correção |
| Área | provider/aggregator |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | DeviousCardi |
| Aberto em | 2026-10-13 (última atualização: 2026-11-13) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +189/-1 em 2 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/L, area/provider |
| Status | importado com ajustes |

## O que faz

Garante que a primeira configuracao do provider file chegue ao watcher antes da do provider interno, com throttling ativo. Evita o erro de middleware de entrypoint inexistente no log de startup (issue #9779). Passa a primeira mensagem do provider por fora do ring channel.

## Riscos e situação no upstream

Mexe no aggregator (nucleo de configuracao). Label status/1-needs-design-review; maintainer pediu design review antes de mudar mais. Sem reviews formais. Nao resolve a ordem entre providers Kubernetes assincronos.

## Principais arquivos

- `pkg/provider/aggregator/aggregator.go`
- `pkg/provider/aggregator/aggregator_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14064/head:upstream-pr-14064`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- A corrida existe. Com throttling ligado, o `Provide` do file provider retornava assim que a mensagem entrava no ring channel, e a goroutine de encaminhamento podia ser escalonada depois de o provider interno já ter enviado a sua configuração. Como o `ConfigurationWatcher` aplica a configuração quando chega a do provider interno, routers internos que usam middlewares de entrypoint vindos do file provider logavam "middleware does not exist" no boot.
- A solução: a primeira mensagem não passa pelo ring channel e, se for enviada durante o `Provide`, é entregue antes do retorno, garantido pelo handshake `synced`/`stopped`. As mensagens seguintes continuam com throttling.
- Não há risco de deadlock: o canal do `ConfigurationWatcher` tem buffer de 100 e o receptor já está rodando antes do aggregator. Providers que mandam a primeira mensagem de forma assíncrona não ficam bloqueados (o handshake é aceito logo). Se o provider retorna erro sem enviar nada, o `Provide` retorna logo. Os demais providers são lançados em `safe.Go`, então a espera de um não atrasa os outros.
- O comportamento do throttling depois da primeira mensagem não muda (espera `providerThrottleDuration` antes de consumir o ring).
- Lint: o teste ignorava o erro recebido de `errCh` (errcheck).
- Os testes usam tempos curtos (100 ms), o que pode gerar flakiness em CI muito carregado, mas passaram com `-race -count=3`.

**Ajustes:** só a correção do errcheck no teste (o erro recebido vai para a mensagem de falha).

**Testes:** `go build ./pkg/...` ok; `go test -race -count=3 ./pkg/provider/aggregator/` ok; `go test ./pkg/server/` ok; `golangci-lint run ./pkg/provider/aggregator/...` com 0 issues.
