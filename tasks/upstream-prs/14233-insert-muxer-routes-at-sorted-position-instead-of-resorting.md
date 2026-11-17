# Insert muxer routes at sorted position instead of resorting

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14233](https://github.com/traefik/traefik/pull/14233) |
| Tipo | Correção |
| Área | muxer/http, muxer/tcp (performance) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | hood |
| Aberto em | 2026-11-05 (última atualização: 2026-11-09) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +96/-32 em 4 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/rules, area/performance |
| Status | pendente |

## O que faz

Troca o append seguido de re-sort por inserção binária na posição ordenada ao adicionar cada rota nos muxers HTTP e TCP. Benchmark reportado: 20k rotas caem de 1,19 s para 60-76 ms. Também torna determinística a ordem de rotas com mesma prioridade e provider.

## Riscos e situação no upstream

Mudança localizada em dois arquivos de mux, sem mudança de API. Só traz ganho perceptível com muitas rotas. Vinculada à issue #14228 (não encontrada pelo número no repositório). Review: status/2-needs-review, sem review de maintainer ainda.

## Principais arquivos

- `pkg/muxer/http/mux.go`
- `pkg/muxer/http/mux_test.go`
- `pkg/muxer/tcp/mux.go`
- `pkg/muxer/tcp/mux_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14233/head:upstream-pr-14233`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
