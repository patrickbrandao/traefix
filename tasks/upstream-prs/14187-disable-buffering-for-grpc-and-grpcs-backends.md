# Disable buffering for gRPC and gRPCS backends

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14187](https://github.com/traefik/traefik/pull/14187) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | Akshaay1 |
| Aberto em | 2026-10-30 (última atualização: 2026-10-31) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +147/-0 em 3 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/M, area/provider/k8s/ingress-nginx |
| Status | importado com ajustes |

## O que faz

Para backends com backend-protocol GRPC ou GRPCS no provider ingress-nginx, o middleware Buffering é desligado, independente de proxy-buffering e proxy-request-buffering. Com o default proxy-request-buffering=on, o corpo era lido inteiro antes de seguir, travando streams gRPC.

## Riscos e situação no upstream

Muda a tradução de buffering para uma classe de backend. Status: needs-design-review, então a abordagem pode mudar. Escopo restrito ao ingress-nginx.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/middleware.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `pkg/provider/kubernetes/ingress-nginx/fixtures/ingresses/ingress-with-grpc-backend-protocol-and-buffering.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14187/head:upstream-pr-14187`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- Com `backend-protocol: GRPC/GRPCS` e `proxy-buffering`/`proxy-request-buffering: "on"`, o provider ingress-nginx criava um middleware `Buffering`. Para gRPC isso é incorreto: o buffering do oxy acumula request e response inteiros, o que quebra streaming (client-, server- e bidi-streaming) e pode segurar até `maxRequestBodyBytes`/`maxResponseBodyBytes` em memória por stream. O `grpc_pass` do NGINX não tem buffering equivalente. O PR deixa de criar o middleware para backends gRPC.
- Efeito colateral a registrar: para gRPC, o limite `proxy-body-size` (que só é aplicado pelo middleware de buffering) também deixa de existir. Hoje ele já não é aplicado com o buffering de requisição desligado, que é o padrão (ver a task de #13220), então quase nada muda na prática.
- Sem outros riscos: só remove um middleware para um protocolo em que ele era nocivo.

**Ajustes**

- Teste adaptado ao formato atual: o conflito com casos novos de `TestLoadIngresses` foi resolvido e as expectativas de TLS seguem o modelo atual (opção TLS por host, `hostTLSOption()`).

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress-nginx/` ok.
- `golangci-lint run ./pkg/provider/kubernetes/ingress-nginx/...`: 0 issues.
