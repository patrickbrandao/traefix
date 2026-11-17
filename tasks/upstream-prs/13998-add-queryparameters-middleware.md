# Add queryParameters middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13998](https://github.com/traefik/traefik/pull/13998) |
| Tipo | Recurso |
| Área | middleware/queryParameters |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | IslamElsayed |
| Aberto em | 2026-10-02 (última atualização: 2026-10-03) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +561/-0 em 15 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/middleware |
| Status | importado com ajustes |

## O que faz

Novo middleware queryParameters que edita a query string antes do encaminhamento: set, add e delete, aplicados nessa ordem. Trabalha sobre a query crua para preservar ordem e encoding de parametros nao tocados (importante para URLs assinadas).

## Riscos e situação no upstream

Toca CRD (tipos, deepcopy gerado, applyconfiguration) e docs. Nome da opcao (delete vs remove) e inclusao nos providers file/docs ainda em aberto pelo autor. Tres reviews COMMENTED; label status/2-needs-review. Nao e path rewrite (fica para depois).

## Principais arquivos

- `pkg/middlewares/queryparameters/query_parameters.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/server/middleware/middlewares.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/middleware.go`
- `docs/content/reference/routing-configuration/http/middlewares/queryparameters.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13998/head:upstream-pr-13998`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Análise:
- Middleware novo `queryParameters` com `set`, `add` e `delete`, aplicados nessa ordem.
- Trabalha sobre a query crua, e não sobre `url.Values`, para preservar a ordem e a codificação dos parâmetros não tocados, o que é importante para URLs assinadas. Os nomes são comparados depois do decode. Também atualiza `RequestURI`.
- Implementação completa: CRD, conversão, docs e testes. Sem dependências novas, sem chamadas de rede, e todos os valores escritos passam por `url.QueryEscape`.
- Upstream: em "review". A review automática levantou só pontos menores.
- **Riscos de segurança (de uso, não do código):**
  - `delete` compara nomes com distinção de maiúsculas e só separa por `&`;
  - um backend que trata nomes sem distinção de maiúsculas, ou que também separa por `;`, recebe o parâmetro "removido" (ex.: `Admin=1` ou `x=1;admin=1`);
  - quem usar o `delete` como filtro de segurança pode ser contornado.
- Conflitou com a importação do #13179 em `crd/kubernetes.go`, porque o PR não conhecia `RedirectTrailingSlash` nem `Tap`. Resolvido mantendo os dois.

Ajustes:
- O conflito em `crd/kubernetes.go` foi resolvido sem perder os campos locais (`RedirectTrailingSlash`, `Tap`, `URNRefs`).
- Deepcopy, applyconfiguration e CRDs regenerados com `script/code-gen.sh`, com o `sed` do macOS aplicado à mão.
- Doc: novo aviso "Not an access control" sobre a distinção de maiúsculas e o `;`.
- Exemplo de referência em `docs/content/reference/dynamic-configuration/file.{toml,yaml}` (`Middleware22a`, sem renumerar).
- Testes novos: parâmetro sem valor (`admin`, `admin=`), nome com `+`, e escape inválido mantido como está.

Testes:
- `go build ./...` ok.
- `go test ./pkg/middlewares/queryparameters/ ./pkg/provider/kubernetes/crd/ ./pkg/server/middleware/ ./pkg/config/dynamic/` ok.
- `golangci-lint run` nesses pacotes: 0 issues.
