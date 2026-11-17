# fix(http3): honour graceTimeOut when shutting down the HTTP/3 server

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13872](https://github.com/traefik/traefik/pull/13872) |
| Tipo | Correção |
| Área | entrypoint/http3 |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | datrixlab |
| Aberto em | 2026-09-09 (última atualização: 2026-10-14) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +115/-3 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/http3 |
| Status | importado com ajustes |

## O que faz

Faz o shutdown do servidor HTTP/3 respeitar transport.lifeCycle.graceTimeOut, usando quic-go Server.Shutdown(ctx) em vez de Close() imediato. Sem isso, respostas em andamento no HTTP/3 são cortadas no shutdown, enquanto no HTTP/1.1 do mesmo entrypoint o período de graça é respeitado.

## Riscos e situação no upstream

Depende de quic-go v0.62.0, que já está no go.mod. Status needs-review, sem review de maintainer; PR atualizado em outubro.

## Principais arquivos

- `pkg/server/server_entrypoint_tcp_http3.go`
- `pkg/server/server_entrypoint_tcp_http3_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13872/head:upstream-pr-13872`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto. O revisor jnoordsij pediu mudanças só no teste: verificar que, iniciado o shutdown, novos requests deixam
de ser aceitos. Disse que a mudança de código em si está correta.

### Achados no PR original

- Correto: o `Shutdown` do HTTP/3 ignorava o contexto com o `graceTimeOut` e chamava `http3.Server.Close()`, abortando os
  requests em andamento, enquanto HTTP/1.1 e HTTP/2 respeitavam o grace period. O quic-go v0.65 (versão atual do go.mod)
  tem `Server.Shutdown(ctx)`: envia GOAWAY, fecha o listener criado a partir do nosso `PacketConn` (então rejeita
  conexões novas), espera os requests e, quando o contexto expira, fecha tudo e devolve `ctx.Err()`, que o
  `shutdownServer` já trata.
- O teste usava a porta fixa `127.0.0.1:8092`, que pode colidir com outros testes, e não cobria o pedido do revisor.

### Ajustes

- Teste com porta dinâmica: o cliente QUIC disca o endereço real do socket UDP do entrypoint.
- Pedido do revisor: depois de iniciado o shutdown, um request novo (outro transporte, outra conexão QUIC) falha,
  enquanto o request em andamento termina.

### Testes

- `go test ./pkg/server/ -run 'TestHTTP3'` ok; `golangci-lint run ./pkg/server/...`: 0 issues.
