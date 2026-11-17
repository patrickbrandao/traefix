# Add forwardAuth.addAuthCookiesToResponseRegex

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12229](https://github.com/traefik/traefik/pull/12229) |
| Tipo | Recurso |
| Área | middleware/forwardauth |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | dakriy |
| Aberto em | 2025-11-04 (última atualização: 2026-11-09) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +193/-41 em 13 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/middleware, area/authentication |
| Status | importado |

## O que faz

Adiciona addAuthCookiesToResponseRegex ao ForwardAuth, para copiar cookies da resposta de autenticação por regex. Resolve cookies com nomes dinâmicos, como os do authentik.

## Riscos e situação no upstream

13 arquivos (+193/-41), inclui CRD e código gerado (exige make generate-crd). Status needs-review, sem reviews. Base master.

## Principais arquivos

- `pkg/middlewares/auth/forward.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/crd/kubernetes.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/middleware.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12229/head:upstream-pr-12229`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

Veredito: **importado**. O PR foi aplicado como está, e o código gerado foi conferido com `make generate-crd` e `go generate`, que não produziram diferenças.

Achados:
- PR aberto e aprovado por um maintainer (lbenguigui, LGTM) depois de duas rodadas de mudanças. O ponto de design discutido foi aceito pelo maintainer por coerência com `authResponseHeadersRegex`: o regex remove do backend todos os cookies que casam, mesmo sem cookie de auth correspondente. Para a segurança é o comportamento mais seguro, porque o backend não consegue plantar um cookie que o auth server controla.
- O modifier passa a trabalhar com as linhas `Set-Cookie` brutas (sem `res.Cookies()`/`cookie.String()`), o que também corrige a perda de atributos e de valores que o `net/http` rejeita. Isso se sobrepõe ao #14277 (ver a task dele).
- Os cookies do auth server também passam a ser copiados brutos. Antes, cookies inválidos para o `net/http` eram descartados. Não há risco de injeção de header, porque o cliente HTTP do Go já rejeita CR/LF nos valores.
- Sem mudanças em `go.mod`, CI ou scripts.

Ajustes: nenhum no código. A otimização de não instalar o modifier quando não há nada a substituir ficou para o #14277.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/middlewares/auth/ ./pkg/provider/kubernetes/crd/... ./pkg/config/...`: ok.
- `golangci-lint run ./pkg/middlewares/auth/... ./pkg/provider/kubernetes/crd/... ./pkg/config/dynamic/...`: 0 issues.

Nota (#14277): o #14277 corrige o mesmo problema da reserialização dos `Set-Cookie`. A versão deste PR (linhas brutas, nome extraído cortando em `;` e depois em `=`) foi mantida. Do #14277 vieram os testes e a regra de não instalar o modifier quando não há cookie a substituir.
