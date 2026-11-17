# Fix infinite loop in WRR load balancer when all up servers are fenced

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13511](https://github.com/traefik/traefik/pull/13511) |
| Tipo | Correção |
| Área | service/loadbalancer/wrr |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | TowyTowy |
| Aberto em | 2026-07-12 (última atualização: 2026-10-23) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +75/-6 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/service |
| Status | pendente |

## O que faz

Corrige um travamento do balanceador WRR: o nextServer fica em loop segurando o lock quando nenhum servidor está ao mesmo tempo up e não fenced (todos fenced, ou único servidor com peso zero). A guarda passa a checar a mesma condição de saída do loop, e o balanceador responde 503 como o P2C já faz.

## Riscos e situação no upstream

Correção pequena e localizada, com teste. Cenário de rolling update com pods terminando ainda up é realista. Status needs-review, sem review de maintainer ainda. PR base v3.7.

## Principais arquivos

- `pkg/server/service/loadbalancer/wrr/wrr.go`
- `pkg/server/service/loadbalancer/wrr/wrr_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13511/head:upstream-pr-13511`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
