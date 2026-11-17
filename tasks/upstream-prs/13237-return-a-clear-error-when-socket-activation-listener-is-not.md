# Return a clear error when socket activation listener is not TCP

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13237](https://github.com/traefik/traefik/pull/13237) |
| Tipo | Correção |
| Área | server/socket-activation |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | Deln0r |
| Aberto em | 2026-05-26 (última atualização: 2026-10-01) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +72/-1 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/server |
| Status | importado com ajustes |

## O que faz

Troca o panic em buildListener por um erro claro quando a socket activation entrega um listener que não é TCP (ex.: *net.UnixListener) para um entrypoint TCP. O comportamento para listeners TCP permanece igual. Fixes #10924.

## Riscos e situação no upstream

PR base v3.7 (precisa ser portado para master). Review APPROVED. Mudança pequena com testes de regressão. Afeta só quem usa socket activation.

## Principais arquivos

- `pkg/server/server_entrypoint_tcp.go`
- `pkg/server/socket_activation_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13237/head:upstream-pr-13237`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, aprovado por um revisor externo; status "review".

### Achados no PR original

- Correção simples e correta: troca um type assertion que entrava em pânico (`listener.(*net.TCPListener)`) por um erro
  quando o socket activation entrega, por exemplo, um `*net.UnixListener`.
- O teste usa `/tmp` e sockets Unix e estava num arquivo sem build tag, o que quebraria no Windows.

### Ajustes

- Teste movido para `socket_activation_unix_test.go`, com `//go:build !windows`, como os demais arquivos de socket activation.

### Testes

- `go test ./pkg/server/ -run TestBuildListenerSocketActivation` ok; `go vet ./pkg/server/` ok.
- `golangci-lint run ./pkg/server/...`: 0 issues.
