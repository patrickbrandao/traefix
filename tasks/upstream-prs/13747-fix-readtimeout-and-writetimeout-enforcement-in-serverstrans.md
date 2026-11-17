# Fix readTimeout and writeTimeout enforcement in ServersTransport

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13747](https://github.com/traefik/traefik/pull/13747) |
| Tipo | Correção |
| Área | server/servers-transport, provider/kubernetes/crd |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | mmatur |
| Aberto em | 2026-08-21 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +411/-170 em 12 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/L, area/server, area/provider/k8s/crd, area/provider/k8s |
| Status | importado com ajustes |

## O que faz

Corrige regressões do enforcement de readTimeout/writeTimeout no FastProxy (introduzido no #12716): os deadlines passam a ser armados só durante escrita de request e leitura de response, então conexões ociosas e WebSocket não são mais derrubadas e um request não pode mais pendurar. Também documenta o comportamento com HTTP/2.

## Riscos e situação no upstream

Afeta FastProxy (experimental) e o ingress-nginx com fastProxy. Base master, status/2-needs-review, review não verificado. Depende do #12716 já estar no fork. Toca 12 arquivos, incluindo CRD.

## Principais arquivos

- `pkg/proxy/fast/builder.go`
- `pkg/proxy/fast/builder_test.go`
- `pkg/proxy/fast/connpool.go`
- `pkg/proxy/fast/connpool_test.go`
- `pkg/proxy/fast/proxy.go`
- `pkg/proxy/fast/proxy_websocket_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13747/head:upstream-pr-13747`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- O PR tem aprovação de um maintainer (LBF38) e revisões sem achados. Revisão própria do sincronismo: `readLoopErr` é escrito antes do `close(readLoopDone)` (o `defer` de close roda por último), então a leitura após `<-readLoopDone` é segura; `upgraded` passa a ser marcado antes de `r.Upgrade`, desligando os deadlines no túnel; o deadline armado na escrita é limpo nos caminhos de resposta bufferizada e de 101.
- Corrige uma regressão do FastProxy no fork: hoje `readTimeout`/`writeTimeout` derrubam conexões ociosas do pool e WebSockets, e uma conexão cujo readLoop termina antes de receber o `RWCh` deixa a request pendurada para sempre (vazamento de goroutine).
- Risco residual (não resolvido, já existe no fork): com um backend que aceita a conexão TCP mas nunca responde, sem `readTimeout`/`responseHeaderTimeout`, `roundTrip` continua bloqueado no envio de `RWCh` mesmo depois que o cliente desiste, pois o `select` não observa o contexto da request. Fechar a conexão ao cancelar o contexto não resolve sozinho (o readLoop pode estar entre o `Peek` e o `<-RWCh` e ficaria preso), então não mexi; vale uma task própria.
- O teste `TestInvalidStatusCode` travou uma vez sob `-race` com a máquina muito carregada; a corrida (backend escreve antes do request, conexão marcada como quebrada e nova conexão sem resposta) é a do item anterior, independente do PR. Rodado 30x sem `-race`, passa.
- Um teste do fork (`connpool_test.go`) usava a assinatura antiga de `newConnPool`.
- A nota de migração caiu na seção `v3.7.12` do fork.

Ajustes:
- Chamada de `newConnPool` atualizada em `connpool_test.go`.
- Nota de migração movida para a seção `v3.8.0` (as notas do Traefix sobre Forwarded/X-Real-Port foram mantidas).

Testes: `go build ./...` ok; `go test ./pkg/proxy/fast/ ./pkg/server/service/` ok; `go test -race ./pkg/proxy/fast/ -skip TestInvalidStatusCode` ok; `golangci-lint run ./pkg/proxy/fast/... ./pkg/server/service/`: só 2 avisos `errcheck` em `proxy_test.go` e `upgrade.go`, arquivos que o PR não toca (já existem no fork).
