# Fix Exact path precedence over overlapping Prefix paths

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14110](https://github.com/traefik/traefik/pull/14110) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | amazon7737 |
| Aberto em | 2026-10-21 (última atualização: 2026-11-13) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +379/-20 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/L, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

No provider ingress-nginx, um path Prefix mais longo podia vencer um path Exact sobreposto e levar a requisição ao router/middleware errado. A correção exclui de regras Prefix os caminhos Exact conflitantes, preserva prioridades e trata use-regex/rewrite-target com a mesma semântica do ingress-nginx.

## Riscos e situação no upstream

Mexe na tradução de regras e prioridades do provider ingress-nginx (+379/-20, 3 arquivos). Afeta roteamento de quem migra do ingress-nginx; precisa de testes de regressão. Status: aguardando review. Docs não atualizadas.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/translator.go`
- `pkg/provider/kubernetes/ingress-nginx/priority_test.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14110/head:upstream-pr-14110`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
