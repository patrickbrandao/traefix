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
| Status | pendente |

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
