# Allow service healthcheck body

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12468](https://github.com/traefik/traefik/pull/12468) |
| Tipo | Recurso |
| Área | healthcheck |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | aek |
| Aberto em | 2025-12-24 (última atualização: 2026-03-12) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +44/-1 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, contributor/waiting-for-corrections, size/S, area/healthcheck |
| Status | pendente |

## O que faz

Permite definir um body na healthcheck de services HTTP, para usar requisições POST, por exemplo JSON-RPC (caso Odoo). Inclui exemplos na documentação.

## Riscos e situação no upstream

Pequeno (+44/-1). Um reviewer pediu mudanças (CHANGES_REQUESTED) e o autor ainda não fez correções (contributor/waiting-for-corrections).

## Principais arquivos

- `pkg/config/dynamic/http_config.go`
- `pkg/healthcheck/healthcheck.go`
- `docs/content/routing/services/index.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12468/head:upstream-pr-12468`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
