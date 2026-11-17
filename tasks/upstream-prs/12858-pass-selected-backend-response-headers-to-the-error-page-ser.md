# Pass selected backend response headers to the error page service

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12858](https://github.com/traefik/traefik/pull/12858) |
| Tipo | Recurso |
| Área | middleware/errors |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | strausmann |
| Aberto em | 2026-03-20 (última atualização: 2026-11-13) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +305/-24 em 14 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/middleware, contributor/waiting-for-feedback |
| Status | importado com ajustes |

## O que faz

Adiciona ao middleware Errors a opção errorResponseHeaders, uma allowlist opt-in de headers da resposta do backend copiados para a requisição ao serviço de páginas de erro. Com isso, WWW-Authenticate pode chegar ao error service. Fecha a issue #11957.

## Riscos e situação no upstream

Médio a grande (14 arquivos, CRD e código gerado). Já passou por várias rodadas de review (COMMENTED) e removeu o forwardHeaders a pedido. Aguardando novo review. Autor cobrou ETA.

## Principais arquivos

- `pkg/middlewares/customerrors/custom_errors.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/middleware.go`
- `docs/content/reference/routing-configuration/http/middlewares/errorpages.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12858/head:upstream-pr-12858`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes** (ajustes pequenos).

Análise:
- O PR cria a opção `errorResponseHeaders` (opt-in, vazia por padrão) no middleware Errors. Ela copia headers da resposta original do backend para a *requisição* enviada ao serviço de página de erro, o que resolve o `WWW-Authenticate` perdido (#11957).
- A versão anterior do PR tinha `forwardHeaders`, que copiava headers do backend direto para a resposta do cliente. Era perigoso: `Content-Length`/`Content-Encoding` ficariam incoerentes com o corpo da página de erro. A opção foi removida a pedido do maintainer (amazon7737) e não existe na versão revisada.
- Os nomes são canonicalizados e deduplicados, e headers hop-by-hop são ignorados. Header listado e ausente na resposta é removido da requisição ao serviço, para que o cliente não forje o valor (ex.: um `WWW-Authenticate` falso na requisição).
- Segurança:
  - nada é enviado por padrão;
  - não há caminho de header injection (valores vêm de `http.Header` já validado);
  - nenhuma dependência nova nem chamada de rede;
  - o risco de vazar, por exemplo, `Set-Cookie` do backend para o serviço de erro só existe se o operador listar esse header, e isso está documentado.
- Upstream: em "review", aguardando um segundo maintainer. Não há objeção ao desenho atual.

Ajustes:
- O código gerado (deepcopy, applyconfiguration e YAML de CRD) não foi aplicado do patch: foi regenerado com `script/code-gen.sh` (`make generate-crd`). No macOS, o `sed -i` do fim do script foi aplicado manualmente.
- Acrescentado `errorResponseHeaders` aos exemplos de referência `docs/content/reference/dynamic-configuration/file.{toml,yaml}`, que o PR não atualizava.

Testes:
- `go build ./...` ok.
- `go test ./pkg/middlewares/customerrors/ ./pkg/provider/kubernetes/crd/ ./pkg/config/dynamic/` ok.
- `golangci-lint run` nesses pacotes: 0 issues.
