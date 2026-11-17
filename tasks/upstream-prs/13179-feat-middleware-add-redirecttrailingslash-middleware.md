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
| Status | pendente |

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
