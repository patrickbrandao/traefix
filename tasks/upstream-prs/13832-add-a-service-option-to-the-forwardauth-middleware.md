# Add a service option to the ForwardAuth middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13832](https://github.com/traefik/traefik/pull/13832) |
| Tipo | Recurso |
| Área | middleware/forwardauth |
| Prioridade | **P2** |
| Valor | alta |
| Esforço de port | alto |
| Autor | wjabbour |
| Aberto em | 2026-09-03 (última atualização: 2026-09-27) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +2024/-47 em 24 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/docker, size/L, area/middleware, area/authentication |
| Status | importado com ajustes |

## O que faz

ForwardAuth passa a aceitar service, um service Traefik, em vez de address fixo. A requisição de auth ganha load balancing, health checks, sticky e ServersTransport (mTLS, circuit breaker), e a opção path define o caminho. Fecha #8051.

## Riscos e situação no upstream

2024 linhas em 24 arquivos, mexe em middleware, CRD e docs. address, service e tls passam a ser mutuamente exclusivos. Base master, status/2-needs-review, review não verificado.

## Principais arquivos

- `integration/forward_auth_test.go`
- `integration/resources/compose/forward_auth.yml`
- `integration/testdata/rawdata-crd.json`
- `pkg/config/dynamic/middlewares.go`
- `pkg/middlewares/auth/forward.go`
- `pkg/middlewares/auth/forward_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13832/head:upstream-pr-13832`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

Veredito: **importado com ajustes**.

Achados no PR original:
- PR aberto, sem review de maintainer (só o agente de review automático, cujos pontos o autor tratou). O design segue o do middleware `errors` (`serviceBuilder.BuildHTTP`), o que é coerente. Sem mudanças em `go.mod` nem em CI. O compose de integração usa só `traefik/whoami:v1.12.0`, que já é usada no projeto.
- **DoS: sem timeout no modo service.** O `http.Client` do modo address tem timeout de 30s, mas o handler do service era chamado direto, sem deadline. Um auth server que nunca responde (e o `ServersTransport` padrão não tem `responseHeaderTimeout`) prenderia todas as requisições de clientes indefinidamente.
- A URL da requisição de auth era montada com o `Host` do cliente (`"http://" + req.Host + path`). Não era explorável, porque o handler não disca esse host, mas fazia o parse de um valor controlado pelo cliente: um `Host` com `@` perdia a parte do userinfo no `Host` enviado ao auth server.
- O recorder implementava `Flush()` e `Hijack()` "de fachada" e fazia asserção de `middlewares.Stateful`, um tipo que não existe mais neste código (não compilava). Isso contraria o AGENTS.md (nada de `Flush`/`Hijack` só para satisfazer interface).
- O recorder devolvia o mapa de headers vivo: headers alterados pelo handler depois do `WriteHeader` entravam na resposta de auth (por exemplo em `authResponseHeaders`), ao contrário de uma conexão real.
- O código do PR partia de uma base mais antiga de `kubernetes.go` (`createErrorPageMiddleware` com 4 retornos e sem `URNRefs`), por isso houve conflitos.
- O comportamento é fail-closed: se o service falha (502/503/504) ou excede `maxResponseBodySize` (401), o cliente é negado. Isso foi conferido.

Ajustes e melhorias:
- No modo service, a requisição de auth recebe `context.WithTimeout` com o mesmo timeout do cliente (30s, e configurável depois do #14024). Novo teste `TestForwardAuthServiceTimeout`.
- A URL usa um placeholder fixo (`http://forwardauth/<path>`) e o `Host` do cliente vai só em `forwardReq.Host`. Novo teste `TestForwardAuthServiceHost`.
- O recorder perdeu `Flush`/`Hijack`/`Stateful`. Agora implementa só `FlushError() error` (no-op, via `http.ResponseController`), e `Hijack` cai em `http.ErrNotSupported`. Os headers são congelados no `WriteHeader`. O teste de interfaces foi reescrito como `TestForwardAuthServiceResponseRecorder`, que cobre flush, hijack e header tardio.
- Os logs e o tracing mostram `service <nome>` em vez da URL placeholder.
- Integração com o CRD reescrita no padrão atual do `createErrorPageMiddleware` (`(*dynamic.ForwardAuth, *dynamic.Service, error)`, com `URNRefs`), mantendo `addAuthCookiesToResponseRegex` (#12229).
- O tratamento de cancelamento do #13655 foi preservado no novo caminho.
- CRDs, applyconfiguration, deepcopy e docs regenerados com `make generate-crd` e `go generate` (o `sed -i` do script precisou de um shim para o BSD sed do macOS; o resultado é o mesmo).

Limitação conhecida: no modo service não há suporte a NTLM/Kerberos (conexão dedicada por cliente), porque o tráfego passa pelo load balancer.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/middlewares/auth/ ./pkg/provider/kubernetes/crd/ ./pkg/server/middleware/ ./pkg/config/...`: ok.
- `golangci-lint run ./pkg/middlewares/auth/... ./pkg/provider/kubernetes/crd/... ./pkg/server/middleware/... ./pkg/config/dynamic/... ./integration/...`: 0 issues.
- Integração (`ForwardAuthSuite`, `TestCRDConfiguration`): não rodada (Docker).
