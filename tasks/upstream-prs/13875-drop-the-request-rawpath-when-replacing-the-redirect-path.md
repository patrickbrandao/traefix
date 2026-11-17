# Drop the request RawPath when replacing the redirect path

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13875](https://github.com/traefik/traefik/pull/13875) |
| Tipo | Correção |
| Área | middleware/gatewayapi-redirect |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | datrixlab |
| Aberto em | 2026-09-09 (última atualização: 2026-09-14) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +22/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware, area/provider/k8s/gatewayapi |
| Status | pendente |

## O que faz

No filtro RequestRedirect do Gateway API, ao sobrescrever o Path com ReplaceFullPath, o RawPath da requisição original continua presente e o Location pode sair com %2F em vez de duas barras. A correção é uma linha que zera RawPath, igual ao que urlrewrite já faz.

## Riscos e situação no upstream

Alteração de uma linha, com paralelo já existente em url_rewrite.go. Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/middlewares/gatewayapi/redirect/request_redirect.go`
- `pkg/middlewares/gatewayapi/redirect/request_redirect_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13875/head:upstream-pr-13875`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
