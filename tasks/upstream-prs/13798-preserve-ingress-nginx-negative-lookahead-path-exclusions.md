# Preserve ingress-nginx negative lookahead path exclusions

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13798](https://github.com/traefik/traefik/pull/13798) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | omkar619-dev |
| Aberto em | 2026-08-28 (última atualização: 2026-10-28) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +924/-21 em 9 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/L, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Paths do ingress-nginx com lookahead negativo, como /api/(?!_internal).*, não compilam no regexp do Go e o router é descartado em silêncio. O patch traduz o formato suportado para PathRegexp mais uma regra de exclusão, preservando a numeração das capturas para rewrite-target.

## Riscos e situação no upstream

924 linhas em 9 arquivos, com tradução de regex em três pontos (buildRule, buildRewriteTarget, buildRedirect). O suporte é restrito a um formato, e o resto continua sendo descartado com warning. Base v3.7, precisa de adaptação. Status/2-needs-review.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `pkg/provider/kubernetes/ingress-nginx/lookahead.go`
- `pkg/provider/kubernetes/ingress-nginx/lookahead_test.go`
- `pkg/provider/kubernetes/ingress-nginx/middleware.go`
- `pkg/provider/kubernetes/ingress-nginx/model.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13798/head:upstream-pr-13798`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
