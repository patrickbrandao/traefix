# Make ipAllowList lookup fast for large lists

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13077](https://github.com/traefik/traefik/pull/13077) |
| Tipo | Recurso |
| Área | middlewares/ipallowlist, pkg/ip |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | medio |
| Autor | giner |
| Aberto em | 2026-04-29 (última atualização: 2026-06-06) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +850/-37 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/middleware |
| Status | pendente |

## O que faz

Troca a busca linear do ipAllowList por hash map para IPs únicos e busca binária para CIDRs, com os ranges ordenados e mesclados na inicialização. Ganha desempenho em listas com 10k ou mais entradas, inclusive no caminho sem match.

## Riscos e situação no upstream

Mexe em controle de acesso, então precisa de testes de equivalência com o algoritmo atual. Autor corrigiu os benchmarks porque CIDRs adjacentes estavam sendo mesclados. Três reviews COMMENTED, sem aprovação explícita; o maintainer autorizou rebase.

## Principais arquivos

- `pkg/ip/checker.go`
- `pkg/ip/checker_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13077/head:upstream-pr-13077`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
