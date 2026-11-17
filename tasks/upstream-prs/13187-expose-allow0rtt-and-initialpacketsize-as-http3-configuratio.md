# Expose Allow0RTT and InitialPacketSize as HTTP3 configuration options.

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13187](https://github.com/traefik/traefik/pull/13187) |
| Tipo | Recurso |
| Área | entrypoints/http3 |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | davefinster |
| Aberto em | 2026-05-17 (última atualização: 2026-05-18) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +91/-9 em 7 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/server, area/http3 |
| Status | importado com ajustes |

## O que faz

Expõe como opções de entrypoint HTTP/3 o Allow0RTT e o InitialPacketSize do QUIC. O InitialPacketSize ajuda em redes com problemas de MTU, como o caso de Tailscale. Hoje o Allow0RTT está fixo em false no código do fork.

## Riscos e situação no upstream

Allow0RTT tem implicação de segurança (replay de 0-RTT), e um maintainer já pediu análise nesse ponto. Autor adicionou documentação na segunda commit. Diff pequeno (+91/-9) em server_entrypoint_tcp_http3.go e entrypoints.go.

## Principais arquivos

- `pkg/config/static/entrypoints.go`
- `pkg/server/server_entrypoint_tcp_http3.go`
- `pkg/server/server_entrypoint_tcp_http3_test.go`
- `docs/content/reference/install-configuration/entrypoints.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13187/head:upstream-pr-13187`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, sem review técnico; o comentário do maintainer lembra que o 0-RTT foi desativado por segurança (#10867).

### Achados no PR original

- **`allow0RTT` reabre uma vulnerabilidade conhecida:** expor a opção permite reativar o 0-RTT, que foi desligado por causa
  do GHSA-gxrv-wf35-62w9 (bypass de IP allow-list com 0-RTT; os dados 0-RTT também podem ser reenviados por replay).
  Uma opção que só serve para reabrir uma falha corrigida não foi importada.
- **Overflow de `uint16`:** `InitialPacketSize` era convertido de `int` para `uint16` sem limite superior. Por exemplo,
  `66836` virava `1300` sem aviso, e valores acima de 1452 eram reduzidos em silêncio pelo quic-go.
- Validação feita depois de abrir o listener UDP.
- Teste com `time.Sleep` e request HTTP/1 a uma porta fixa (`127.0.0.1:8090`) que não testava nada do HTTP/3.

### Ajustes

- Importado só o `initialPacketSize`; `Allow0RTT` continua fixo em `false`.
- Validação do intervalo `[1200, 1452]` (mínimo do RFC 9000 §14.1, máximo suportado pelo quic-go), feita antes de abrir o socket.
- Conflitos resolvidos sobre o código local (handler com read timeout, `MaxHeaderBytes`, `IdleTimeout`, `ConnContext`).
- Doc: linha `http3.initialPacketSize` em `entrypoints.md`; `configuration-options.md` regenerado com `go generate`. O
  gerador também renumerou as âncoras da tabela de exemplos `mergeSlashes` (#12879).
- Teste de tabela determinístico: padrão, mínimo, máximo, abaixo, acima, overflow e `Allow0RTT` sempre falso.

### Testes

- `go test ./pkg/server/ -run 'TestNewHTTP3Server|TestHTTP3'` ok; `go test ./pkg/config/static/` ok.
- `golangci-lint run ./pkg/server/... ./pkg/config/static/...`: 0 issues; `go vet ./cmd/traefik/` ok.
