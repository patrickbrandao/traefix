# Abort the response when the fast proxy fails after the response has started

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13569](https://github.com/traefik/traefik/pull/13569) |
| Tipo | Correção |
| Área | proxy/fast (fastproxy) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | FedeBev |
| Aberto em | 2026-07-26 (última atualização: 2026-09-22) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +310/-17 em 5 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/L, area/fastproxy |
| Status | importado |

## O que faz

Quando o backend morre no meio da resposta, o fast proxy escrevia 'Internal Server Error' no corpo já em andamento, e a resposta parecia completa. Agora o ServeHTTP aborta com http.ErrAbortHandler, como o httputil.ReverseProxy, e o cliente vê erro de leitura (unexpected EOF). O recovery também segue essa regra.

## Riscos e situação no upstream

Mudança grande (+310/-17) em caminho de proxy, incluindo connpool e recovery. Efeito visível para clientes: a resposta passa a ser cortada em vez de terminar limpa. Status needs-review, sem review de maintainer ainda. PR base v3.7.

## Principais arquivos

- `pkg/proxy/fast/proxy.go`
- `pkg/proxy/fast/connpool.go`
- `pkg/proxy/fast/proxy_test.go`
- `pkg/middlewares/metrics/metrics.go`
- `pkg/middlewares/metrics/metrics_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13569/head:upstream-pr-13569`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, sem review de maintainer.

### Achados no PR original

- Correto. Depois que o status já foi enviado ao cliente, o `ErrorHandler` não pode mais trocá-lo: o `WriteHeader(500)` é
  ignorado e o texto `Internal Server Error` era anexado ao corpo que o cliente já estava lendo, o que corrompia a
  resposta (um cliente tolerante não percebia que o stream tinha quebrado). Agora o FastProxy aborta com
  `http.ErrAbortHandler`, como já fazem o `httputil.ReverseProxy` e o middleware de recovery do Traefik (que deixa esse
  sentinel passar sem log).
- Efeito colateral tratado no PR: o `panic` pulava a coleta de métricas, que foi movida para um `defer`. O access log tem o
  mesmo comportamento que já existia com o proxy `httputil`.
- O marcador `responseStarted` é zerado a cada tentativa, o que importa porque as conexões são reaproveitadas pelo pool.
- Sem impacto de segurança. O comportamento fica mais correto: um erro no meio da resposta deixa de parecer resposta completa.

### Ajustes

- Conflito com o pool de conexões local (`connPool` vindo do contexto) resolvido, mantendo `connPool.ReleaseConn`.
- Conflito de imports nos testes resolvido mantendo os dois lados.

### Testes

- `go test ./pkg/proxy/fast/ -count=5` ok; `go test ./pkg/middlewares/metrics/` ok.
- `golangci-lint run --new-from-rev=HEAD ./pkg/proxy/fast/... ./pkg/middlewares/metrics/...`: 0 issues. Os dois avisos
  `errcheck` que aparecem sem o filtro são de código já existente (`upgrade.go` e um teste antigo).
- Numa das execuções sob carga, `TestNoContentLength` (teste antigo, que não mudou) travou uma vez. Rodou de novo sozinho e
  5 vezes com o pacote inteiro sem falhar: flakiness preexistente, não relacionada ao PR.
- `go vet ./cmd/traefik/` ok.
