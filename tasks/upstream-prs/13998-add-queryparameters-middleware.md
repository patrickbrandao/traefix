# Add queryParameters middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13998](https://github.com/traefik/traefik/pull/13998) |
| Tipo | Recurso |
| Área | middleware/queryParameters |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | IslamElsayed |
| Aberto em | 2026-10-02 (última atualização: 2026-10-03) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +561/-0 em 15 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/middleware |
| Status | pendente |

## O que faz

Novo middleware queryParameters que edita a query string antes do encaminhamento: set, add e delete, aplicados nessa ordem. Trabalha sobre a query crua para preservar ordem e encoding de parametros nao tocados (importante para URLs assinadas).

## Riscos e situação no upstream

Toca CRD (tipos, deepcopy gerado, applyconfiguration) e docs. Nome da opcao (delete vs remove) e inclusao nos providers file/docs ainda em aberto pelo autor. Tres reviews COMMENTED; label status/2-needs-review. Nao e path rewrite (fica para depois).

## Principais arquivos

- `pkg/middlewares/queryparameters/query_parameters.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/server/middleware/middlewares.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/middleware.go`
- `docs/content/reference/routing-configuration/http/middlewares/queryparameters.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13998/head:upstream-pr-13998`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
