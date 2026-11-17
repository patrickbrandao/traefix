# Fix stale TLS options name causing permanent 421 on long-lived connections

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13799](https://github.com/traefik/traefik/pull/13799) |
| Tipo | Correção |
| Área | middleware/tls, provider/kubernetes |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | sk-sanjay01 |
| Aberto em | 2026-08-28 (última atualização: 2026-09-14) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +259/-0 em 5 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/middleware, area/tls, area/provider/k8s |
| Status | importado com ajustes |

## O que faz

Conexões keep-alive resolvem o nome das TLS options uma vez, no accept. Se o router chegou depois, o request cai em 421 para sempre nessa conexão. O patch adiciona Connection: close na resposta 421 do snicheck, forçando o cliente a rehandshake e reavaliar as options.

## Riscos e situação no upstream

Base v3.7, precisa de adaptação. Trata o sintoma, não a causa (o nome das options fica fixo por conexão). Corrige #13745. Status/2-needs-review.

## Principais arquivos

- `pkg/middlewares/snicheck/snicheck.go`
- `pkg/middlewares/snicheck/snicheck_test.go`
- `pkg/server/server_entrypoint_tcp_http3.go`
- `pkg/server/server_entrypoint_tcp_test.go`
- `pkg/tcp/tls.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13799/head:upstream-pr-13799`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- Trata o sintoma (o nome das TLS options é resolvido uma vez por conexão, no handshake), não a causa, mas a correção é segura: a resposta 421 passa a fechar a conexão para o cliente refazer o handshake. HTTP/1.1 fecha o TCP, HTTP/2 envia GOAWAY (gracioso, streams em andamento terminam).
- No HTTP/3 o PR chama `CloseWithError` **antes** de escrever o 421: o `CONNECTION_CLOSE` sai antes da resposta e o cliente recebe só um erro de conexão. Além disso, o fechamento QUIC é abrupto e derruba outros streams da mesma conexão (com coalescing de conexões do navegador, requests de outros hosts que estavam em andamento).
- Toda resposta 421 do snicheck passa a fechar a conexão, inclusive o caso legítimo de request mal direcionado (coalescing HTTP/2 com TLS options diferentes, ou listeners isolados do Gateway API no fork). Comportamento aceitável: o cliente já teria de abrir outra conexão para esse host.
- Comentário longo, mais *como* do que *porquê*.

Ajustes:
- Ordem invertida: o 421 é escrito e enviado com `http.NewResponseController(rw).Flush()` antes de fechar a conexão QUIC, para o cliente receber a resposta.
- Comentário reduzido ao porquê.
- Conflito de imports no teste do entry point resolvido.

Risco residual: no HTTP/3 o fechamento continua abrupto para os outros streams da conexão; quic-go não tem shutdown gracioso por conexão.

Testes: `go build ./...` ok; `go test ./pkg/middlewares/snicheck/ ./pkg/tcp/ ./pkg/server/` ok; `golangci-lint` nos pacotes tocados: só 1 aviso `errcheck` pré-existente em `pkg/tcp/proxy.go` (arquivo não tocado).

## Testes de integração (2026-11-17)

Testes de integração (`TestHTTPSSuite/TestWithDomainFronting`) mostraram dois problemas no HTTP/3.

- O fechamento forçado da conexão QUIC logo depois do 421 concorria com a leitura da resposta pelo cliente: ele recebia `H3_NO_ERROR: stale TLS options` em vez do 421, e os outros streams da conexão também morriam.
- O header `Connection: close` é proibido no HTTP/3 e fazia o cliente rejeitar a resposta.

O fechamento forçado foi removido e o header agora só é enviado no HTTP/1.1 e no HTTP/2. No HTTP/3 a conexão continua aberta: pela RFC 9110, o cliente pode repetir a requisição que recebeu 421 numa nova conexão. Foi adicionado um teste unitário para o HTTP/3.
