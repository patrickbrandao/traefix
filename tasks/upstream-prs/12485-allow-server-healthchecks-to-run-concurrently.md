# Allow server healthchecks to run concurrently

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12485](https://github.com/traefik/traefik/pull/12485) |
| Tipo | Recurso |
| Área | healthcheck |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | extndr |
| Aberto em | 2025-12-30 (última atualização: 2026-01-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +131/-35 em 5 arquivos |
| Labels | status/1-needs-design-review, size/M, area/healthcheck |
| Status | importado com ajustes |

## O que faz

Executa as healthchecks dos servers de um service em paralelo, reduzindo a latência de monitoramento quando há muitos backends. Nova opção maxConcurrency, com 1 restaurando o comportamento anterior. Fecha a issue #9748.

## Riscos e situação no upstream

Muda o comportamento padrão de healthcheck (agora paralelo). Status design-review. Toca healthcheck.go e a config HTTP.

## Principais arquivos

- `pkg/healthcheck/healthcheck.go`
- `pkg/healthcheck/healthcheck_test.go`
- `pkg/config/dynamic/http_config.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12485/head:upstream-pr-12485`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:

- O padrão muda para 10 checagens simultâneas por serviço (`DefaultHealthCheckMaxConcurrency = 10`). É uma mudança de comportamento padrão sem decisão do upstream: o PR está em `status/1-needs-design-review`. Depois de um upgrade, serviços com muitos servidores passariam a receber rajadas de probes.
- Goroutines podem vazar no reload. Quando o contexto é cancelado no meio do despacho, o loop faz `return` sem `wg.Wait()` e as goroutines das checagens já iniciadas continuam vivas depois do fim do health checker. Além disso, `Launch` já não esperava a goroutine do loop "unhealthy".
- A escolha do `select` entre `ctx.Done()` e o semáforo é aleatória quando os dois estão prontos. Com isso, checagens ainda podiam ser disparadas depois do cancelamento.
- Há dois caminhos de código (sequencial e concorrente) com a mesma lógica duplicada.
- A documentação foi aplicada em caminhos antigos. O campo não existe no CRD Kubernetes (o PR também não o adiciona).

Ajustes:

- O padrão continua sequencial (`maxConcurrency` ausente ou `<= 0` vale 1). A concorrência é opt-in.
- Caminho único `checkTargets`: um semáforo limita as checagens em andamento, `defer wg.Wait()` garante que nenhuma checagem sobrevive ao ciclo e `ctx.Err()` é verificado de novo depois de pegar o semáforo.
- `Launch` agora espera também o loop de alvos unhealthy (`sync.WaitGroup`), então nenhuma goroutine de health check sobrevive ao reload.
- `testhelpers.CollectingGauge` ganhou um mutex nos escritores. Isso corrige a race que já existia em `TestDifferentIntervals` com `-race`.
- `label_test.go` ganhou a expectativa do novo campo no encode.
- Docs: `maxConcurrency` em `reference/dynamic-configuration/file.{toml,yaml}` e na tabela de `http/load-balancing/service.md`.
- Testes novos: valores padrão e inválidos de `maxConcurrency`. Também o número máximo de checagens simultâneas observado no servidor (1, 2 e limitado pelo número de alvos), a distribuição correta entre os canais healthy/unhealthy e, por fim, que nenhuma checagem é iniciada nem reportada depois do cancelamento.

Testes:

- `go build ./...`: ok. Durante parte da sessão o disco ficou cheio por builds paralelos de outros agentes. A compilação foi conferida com `go build ./pkg/... && go vet ./cmd/traefik/` e depois com o `go build ./...` completo.
- `go test -race -count=3 ./pkg/healthcheck/`: ok. A race antiga também sumiu.
- `go test ./pkg/config/... ./pkg/server/service/ ./pkg/provider/kv/ ./pkg/provider/docker/`: ok.
- `golangci-lint run ./pkg/healthcheck/... ./pkg/testhelpers/... ./pkg/config/dynamic/...`: 0 issues.
