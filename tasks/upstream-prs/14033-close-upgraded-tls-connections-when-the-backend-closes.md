# Close upgraded TLS connections when the backend closes

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14033](https://github.com/traefik/traefik/pull/14033) |
| Tipo | Correção |
| Área | proxy/httputil, server |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | bhaskargurram-ai |
| Aberto em | 2026-10-09 (última atualização: 2026-10-12) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +185/-1 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/server |
| Status | importado |

## O que faz

Em conexoes upgraded (WebSocket) sobre TLS, o Go 1.25+ chama CloseWrite no cliente quando o backend fecha e isso so envia close_notify, deixando o TCP aberto. O PR oculta CloseWrite nessas conexoes para o cliente receber FIN. Conexoes sem TLS nao mudam. Corrige #13999 (agentes Jenkins que nao reconectavam).

## Riscos e situação no upstream

Mexe no caminho de upgrade do ReverseProxy (proxy/httputil). Sem reviews ainda; label status/2-needs-review. Testes de regressao incluidos. Vale checar contra os demais upgrades (h2c, TCP).

## Principais arquivos

- `pkg/proxy/httputil/proxy.go`
- `pkg/proxy/httputil/upgrade.go`
- `pkg/proxy/httputil/upgrade_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14033/head:upstream-pr-14033`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, status "review", sem review técnico.

### Achados no PR original

- Correto: desde o Go 1.25, ao receber EOF do backend numa conexão com upgrade, o `httputil.ReverseProxy` chama
  `CloseWrite` na conexão do cliente. Num `*tls.Conn` isso só envia um `close_notify` e mantém o TCP aberto, então a maioria
  dos clientes WebSocket fica com uma conexão meio aberta e nunca reconecta. O PR esconde o `CloseWrite` só para requests
  TLS com `Connection: Upgrade`. Conexões sem TLS mantêm o half-close.
- Segue as convenções do projeto: o wrapper implementa `Hijack` chamando `http.ResponseController` e expõe `Unwrap`.
- Sem impacto de segurança; também reduz conexões penduradas (consumo de fds).

### Ajustes

Conflito em `buildSingleHostProxy` resolvido: o código local não tem o `newConnectHandler` do upstream, então o wrapper foi inserido na cadeia local (`newH2CUpgradeHandler(newTLSUpgradeHandler(proxy))`).

### Testes

- `go test ./pkg/proxy/httputil/` ok; `golangci-lint run ./pkg/proxy/httputil/...`: 0 issues.
