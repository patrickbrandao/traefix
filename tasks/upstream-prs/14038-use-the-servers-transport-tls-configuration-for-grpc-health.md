# Use the servers transport TLS configuration for gRPC health checks

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14038](https://github.com/traefik/traefik/pull/14038) |
| Tipo | Correção |
| Área | healthcheck/grpc |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | jayhemnani9910 |
| Aberto em | 2026-10-11 (última atualização: 2026-10-15) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +113/-12 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/healthcheck |
| Status | importado com ajustes |

## O que faz

O health check gRPC (mode: grpc) so usava texto puro e falhava em backends TLS, marcando o servidor como DOWN (503). Agora usa o scheme da URL do servidor e, para https, a configuracao TLS do serversTransport do servico.

## Riscos e situação no upstream

Pequeno (4 arquivos, +113/-12). Afeta apenas quem usa healthCheck gRPC com TLS. Tres reviews COMMENTED; label status/2-needs-review.

## Principais arquivos

- `pkg/healthcheck/healthcheck.go`
- `pkg/healthcheck/healthcheck_test.go`
- `pkg/server/service/service.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14038/head:upstream-pr-14038`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:

- O bug é real. Em `mode: grpc`, servidores `https://` sem `scheme` recebiam HTTP/2 em texto puro numa porta TLS. Com `scheme: https`, nenhuma credencial era configurada (`grpc: no transport security set`). Nos dois casos o servidor ficava sempre DOWN.
- A correção usa o scheme da URL do servidor quando `scheme` não está definido, como já acontece no modo HTTP. Para `https`, usa a mesma `*tls.Config` do ServersTransport usada pelo proxy (`GetTLSConfig`).
- Segurança: a verificação não é enfraquecida. `credentials.NewTLS` clona a config (a instância compartilhada com o proxy não é alterada). Com config `nil`, ele verifica contra as raízes do sistema. `insecureSkipVerify` só vale se o operador já o configurou no ServersTransport. O `ServerName`, quando não definido, vem da autoridade discada.
- Mudança de comportamento: um `scheme` desconhecido, que antes falhava por falta de credenciais, agora usa texto puro. Isso é consistente com `http`/`h2c`.

Ajustes:

- `TestServiceHealthChecker_checkHealthGRPC_TLS`: o caso "untrusted certificate" usava as raízes do sistema e falhava no macOS, cujo verificador de plataforma devolve outra mensagem (`certificate is not standards compliant`). Agora usa um pool vazio, então o resultado não depende da plataforma.
- Chamadas de `NewServiceHealthChecker` nos testes adicionados pelo #12485 e pelo #13223 foram adaptadas à nova assinatura (parâmetro `tlsConfig`).

Testes:

- `go test -race -count=2 ./pkg/healthcheck/`: ok.
- `go test ./pkg/server/service/`: ok.
- `golangci-lint run ./pkg/healthcheck/... ./pkg/server/service/`: 0 issues.
- `go build ./pkg/...`: ok.
