# Fix file provider middleware errors at startup

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14064](https://github.com/traefik/traefik/pull/14064) |
| Tipo | Correção |
| Área | provider/aggregator |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | DeviousCardi |
| Aberto em | 2026-10-13 (última atualização: 2026-11-13) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +189/-1 em 2 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/L, area/provider |
| Status | pendente |

## O que faz

Garante que a primeira configuracao do provider file chegue ao watcher antes da do provider interno, com throttling ativo. Evita o erro de middleware de entrypoint inexistente no log de startup (issue #9779). Passa a primeira mensagem do provider por fora do ring channel.

## Riscos e situação no upstream

Mexe no aggregator (nucleo de configuracao). Label status/1-needs-design-review; maintainer pediu design review antes de mudar mais. Sem reviews formais. Nao resolve a ordem entre providers Kubernetes assincronos.

## Principais arquivos

- `pkg/provider/aggregator/aggregator.go`
- `pkg/provider/aggregator/aggregator_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14064/head:upstream-pr-14064`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
