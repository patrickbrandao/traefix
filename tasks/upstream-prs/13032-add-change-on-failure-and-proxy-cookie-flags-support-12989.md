# Add change-on-failure and proxy_cookie_flags support #12989

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13032](https://github.com/traefik/traefik/pull/13032) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress-nginx, middlewares/ingressnginx |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | mmasztalerczuk |
| Aberto em | 2026-04-23 (última atualização: 2026-04-27) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +942/-119 em 25 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/middleware, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Adiciona compatibilidade com o ingress-nginx para session-cookie-change-on-failure, que invalida o cookie de sticky session após respostas 5xx, e para proxy_cookie_flags dentro de configuration-snippet, que altera flags (secure, httponly, samesite) de Set-Cookie. O objetivo é facilitar a migração do Nginx.

## Riscos e situação no upstream

Label needs-design-review; maintainer sugeriu dividir o PR em dois por tamanho. Muito grande (+942/-119, 25 arquivos) e toca o middleware de retry e o snippet, com risco de regressão.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/annotations.go`
- `pkg/middlewares/ingressnginx/snippet/snippet.go`
- `pkg/middlewares/ingressnginx/snippet/action.go`
- `pkg/middlewares/retry/retry.go`
- `pkg/config/dynamic/http_config.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13032/head:upstream-pr-13032`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
