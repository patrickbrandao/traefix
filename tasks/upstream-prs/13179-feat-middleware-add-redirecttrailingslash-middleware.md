# feat(middleware): add RedirectTrailingSlash middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13179](https://github.com/traefik/traefik/pull/13179) |
| Tipo | Recurso |
| Área | middlewares/redirect |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | KaanSimsek |
| Aberto em | 2026-05-16 (última atualização: 2026-05-22) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +556/-53 em 15 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/middleware, area/provider/k8s/crd, area/provider/k8s |
| Status | importado com ajustes |

## O que faz

Adiciona o middleware RedirectTrailingSlash, com mode add ou remove, que redireciona para incluir ou remover a barra final. Preserva query string, ignora a raiz e usa 307/308 em métodos não GET. Resolve o issue #11841, que hoje exige RedirectRegex com padrões complexos.

## Riscos e situação no upstream

Label needs-review sem revisão de maintainer. Acrescenta um tipo novo ao CRD Middleware e à applyconfiguration gerada, então exige regeneração. Reaproveita a base newRedirect, sem dependências novas.

## Principais arquivos

- `pkg/middlewares/redirect/redirect_trailing_slash.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/crd/generated/applyconfiguration/traefikio/v1alpha1/middlewarespec.go`
- `docs/content/reference/routing-configuration/http/middlewares/redirecttrailingslash.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13179/head:upstream-pr-13179`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:
- **Bug: o middleware não funcionava no provider Kubernetes CRD.** O PR adiciona `RedirectTrailingSlash` ao `MiddlewareSpec`, mas não o copia para `dynamic.Middleware` em `pkg/provider/kubernetes/crd/kubernetes.go`. A configuração via CRD era descartada sem aviso.
- Sem validação de `mode` no CRD: um valor inválido só falhava na construção do middleware em runtime.
- `v1alpha1/middleware.go` estava com alinhamento quebrado (não passou por `gofumpt`).
- A entrada nova no mapa de redirects do `mkdocs.yml` aponta para uma página legada (`middlewares/http/redirecttrailingslash.md`) que nunca existiu.
- O PR alterava `docs/content/reference/routing-configuration/other-providers/file.{toml,yaml}`, arquivos que não existem neste branch, e renumerava todos os middlewares do exemplo.
- Funções exportadas sem comentário, e testes com `context.Background()`.
- Segurança:
  - a URL de destino vem do `rawURL` já usado pelo RedirectRegex/RedirectScheme (mesmo host e esquema da requisição);
  - um caminho com barra dupla (`//evil.com`) continua no mesmo host, como os testes novos verificam;
  - as regex são fixas e RE2;
  - nenhuma dependência nova.
- Upstream: em "review", sem comentários técnicos ainda.

Ajustes:
- Mapeamento `RedirectTrailingSlash` em `crd/kubernetes.go`.
- `+kubebuilder:validation:Enum=add;remove` em `mode`.
- Deepcopy, applyconfiguration e CRDs regenerados com `script/code-gen.sh`, com o `sed` do macOS aplicado à mão.
- `gofumpt` nos structs, comentário na função exportada e `t.Context()` nos testes.
- A entrada legada do `mkdocs.yml` foi removida (a entrada no `nav` foi mantida).
- Exemplo adicionado em `docs/content/reference/dynamic-configuration/file.{toml,yaml}` como `Middleware20a`, sem renumerar os outros.
- Testes novos: barra dupla no início (add e remove) e várias barras no fim.

Observação: é uma opção de configuração nova. Se o upstream adotar outro nome ou formato, será preciso migrar.

Testes:
- `go build ./...` ok.
- `go test ./pkg/middlewares/redirect/ ./pkg/provider/kubernetes/crd/ ./pkg/config/... ./pkg/server/middleware/` ok.
- `golangci-lint run` nesses pacotes: 0 issues.
