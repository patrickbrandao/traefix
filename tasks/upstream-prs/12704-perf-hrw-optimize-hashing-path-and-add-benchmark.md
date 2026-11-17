# perf(hrw): optimize hashing path and add benchmark

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12704](https://github.com/traefik/traefik/pull/12704) |
| Tipo | Recurso |
| Área | server/loadbalancer/hrw |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | sandy2008 |
| Aberto em | 2026-02-19 (última atualização: 2026-03-17) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +89/-10 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/server |
| Status | pendente |

## O que faz

Otimiza o caminho quente do balanceador HRW (rendezvous hashing), removendo alocações por backend e concatenações de string no loop de seleção. Sem mudança de comportamento, e inclui benchmark BenchmarkNextServer.

## Riscos e situação no upstream

Pequeno (2 arquivos, +89/-10). Ganho só para quem usa HRW com muitos backends. Maintainers adiaram e pediram rebase antes da revisão.

## Principais arquivos

- `pkg/server/service/loadbalancer/hrw/hrw.go`
- `pkg/server/service/loadbalancer/hrw/hrw_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12704/head:upstream-pr-12704`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
