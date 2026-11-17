# feat: enhance headers middleware to support regex replacement

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12563](https://github.com/traefik/traefik/pull/12563) |
| Tipo | Recurso |
| Área | middleware/headers |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | jayl1e |
| Aberto em | 2026-01-17 (última atualização: 2026-04-13) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +308/-2 em 9 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/middleware, area/provider/k8s |
| Status | pendente |

## O que faz

Adiciona ao middleware headers a substituição de headers por regex, sem dependências novas. Um header só é substituído se existir, sem criar um novo, o que resolve o caso da issue #5809 (por exemplo, reescrever Location em redirects).

## Riscos e situação no upstream

Médio (9 arquivos, +308/-2, inclui CRD e regeneração). Comentários de usuários e maintainer confirmaram a semântica de substituir sem adicionar. Sem review de maintainer ainda.

## Principais arquivos

- `pkg/middlewares/headers/header.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/config/dynamic/zz_generated.deepcopy.go`
- `docs/content/reference/routing-configuration/other-providers/file.yaml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12563/head:upstream-pr-12563`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
