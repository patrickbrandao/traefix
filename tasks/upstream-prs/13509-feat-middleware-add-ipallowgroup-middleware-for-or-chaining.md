# feat(middleware): add IPAllowGroup middleware for OR-chaining

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13509](https://github.com/traefik/traefik/pull/13509) |
| Tipo | Recurso |
| Área | middleware/ipallowlist |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | Aisu2635 |
| Aberto em | 2026-07-11 (última atualização: 2026-10-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +359/-0 em 5 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/middleware |
| Status | pendente |

## O que faz

Adiciona o middleware IPAllowGroup, que combina vários IPAllowList com lógica OR: a requisição é liberada se o IP casar com qualquer uma das regras configuradas. Atende o caso de múltiplos parceiros, cada um com seu conjunto de IPs, citado na issue #12249.

## Riscos e situação no upstream

Adiciona campo novo em config dinâmica (exige regenerar zz_generated.deepcopy.go). Status needs-design-review: maintainers pediram análise de impacto e o autor cobrou retorno. Sem review ainda.

## Principais arquivos

- `pkg/middlewares/ipallowlist/ip_allowgroup.go`
- `pkg/middlewares/ipallowlist/ip_allowgroup_test.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/config/dynamic/zz_generated.deepcopy.go`
- `pkg/server/middleware/middlewares.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13509/head:upstream-pr-13509`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
