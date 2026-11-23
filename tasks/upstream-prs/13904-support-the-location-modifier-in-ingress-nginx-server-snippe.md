# Support the ^~ location modifier in ingress-nginx server-snippet

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13904](https://github.com/traefik/traefik/pull/13904) |
| Tipo | Correção |
| Área | middleware/ingress-nginx-snippet |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | ArchitDewan |
| Aberto em | 2026-09-14 (última atualização: 2026-11-15) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +48/-2 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware, area/provider/k8s/ingress-nginx |
| Status | absorvido pelo oficial |

## O que faz

Trata o modificador ^~ do NGINX em server-snippet como prefixo: remove o modificador antes do match. Antes, location ^~ /internal virava o prefixo literal ^~/internal e nunca casava. Documenta o comportamento suportado; a precedência de location do NGINX não é implementada.

## Riscos e situação no upstream

Escopo restrito a snippets do ingress-nginx, útil sobretudo para migração. Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/middlewares/ingressnginx/snippet/collector.go`
- `pkg/middlewares/ingressnginx/snippet/snippet_test.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13904/head:upstream-pr-13904`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Merge no upstream (2026-11-23)

Mergeado no oficial (`4ee187fb4`, branch v3.7). Esta task não foi importada; a sincronização está em
`tasks/upstream/0011-support-location-modifier-ingress-nginx-server-snippet.md`, que a substitui. Marque como
absorvida quando a 0011 for concluída.

Concluído em 2026-11-23 pela task correspondente de `tasks/upstream/` (aplicada no master).
