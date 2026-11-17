# Add support for h2c to forwardauth

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13909](https://github.com/traefik/traefik/pull/13909) |
| Tipo | Recurso |
| Área | middleware/forwardauth |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | Links2004 |
| Aberto em | 2026-09-15 (última atualização: 2026-11-06) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +107/-1 em 4 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/middleware, area/authentication |
| Status | pendente |

## O que faz

Permite que o forwardAuth fale h2c (HTTP/2 em texto puro) com o servidor de autenticação, para evitar esgotamento de portas efêmeras e somaxconn quando há muitas requisições HTTP/1.1 de auth. Autor é de uma empresa usuária que relata 100% de CPU e erros de cannot assign requested address.

## Riscos e situação no upstream

Maintainers marcaram review e já houve três reviews COMMENTED; o autor pediu andamento. Pede compatibilidade do scheme h2c no config dinâmico.

## Principais arquivos

- `pkg/middlewares/auth/forward.go`
- `pkg/middlewares/auth/forward_test.go`
- `pkg/config/dynamic/middlewares.go`
- `docs/content/reference/routing-configuration/http/middlewares/forwardauth.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13909/head:upstream-pr-13909`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
