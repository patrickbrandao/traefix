# Fix CORS handling for non-allowed origins

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14133](https://github.com/traefik/traefik/pull/14133) |
| Tipo | Correção |
| Área | middleware/cors, provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | KaanSimsek |
| Aberto em | 2026-10-23 (última atualização: 2026-11-06) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +204/-10 em 6 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/M, area/middleware, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

No provider ingress-nginx, o middleware CORS passa a responder só às origens que batem com cors-allow-origin, como o ingress-nginx. Preflights de outras origens seguem para o backend, e Access-Control-Allow-Credentials e Expose-Headers só saem para origens permitidas. Um flag interno ativado só pelo provider; CORS de file provider e CRD não muda.

## Riscos e situação no upstream

Flag interno, sem impacto nos demais providers. Status: needs-design-review, então pode haver mudança de direção. Preflight continua 200 em vez de 204 do ingress-nginx. Toca middleware de headers/config dinâmica.

## Principais arquivos

- `pkg/middlewares/headers/header.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/ingress-nginx/middleware.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14133/head:upstream-pr-14133`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
