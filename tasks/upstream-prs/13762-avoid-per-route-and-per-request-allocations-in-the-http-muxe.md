# Avoid per-route and per-request allocations in the HTTP muxer

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13762](https://github.com/traefik/traefik/pull/13762) |
| Tipo | Recurso |
| Área | server/muxer, router |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | dgprivate |
| Aberto em | 2026-08-25 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +118/-15 em 4 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M |
| Status | pendente |

## O que faz

Elimina alocações por rota e por request no muxer HTTP: getRoutingPath passa a retornar (string, bool), e withRoutingPath evita cópias quando não há percent-encoding. Benchmarks do autor: com 1000 rotas sem match, de 1008 para 6 allocs/op e de 19 para 11 µs/op.

## Riscos e situação no upstream

Otimização de performance, não recurso novo. Toca o caminho quente de roteamento, então exige teste de regressão. Base master, status/2-needs-review, review não verificado. Atualizado em 2026-10-29.

## Principais arquivos

- `pkg/muxer/http/matcher.go`
- `pkg/muxer/http/mux.go`
- `pkg/muxer/http/mux_bench_test.go`
- `pkg/muxer/http/mux_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13762/head:upstream-pr-13762`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
