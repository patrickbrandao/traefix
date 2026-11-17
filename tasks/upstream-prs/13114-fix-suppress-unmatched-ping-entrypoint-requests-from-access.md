# fix: suppress unmatched ping entrypoint requests from access log

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13114](https://github.com/traefik/traefik/pull/13114) |
| Tipo | Correção |
| Área | accesslogs, ping |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | satulurideepak4 |
| Aberto em | 2026-05-06 (última atualização: 2026-06-11) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +82/-2 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/healthcheck, area/accesslogs |
| Status | pendente |

## O que faz

Requisições sem match no entrypoint de ping TLS, com manualRouting, apareciam no access log mesmo com addInternals: false. A correção trata o handler padrão desse entrypoint como interno. Fecha o issue #12751.

## Riscos e situação no upstream

Diff pequeno (+82/-2) em server/middleware/observability.go e router.go. Está em review há muito tempo sem revisor atribuído, e o maintainer disse que está no radar sem ETA.

## Principais arquivos

- `pkg/server/middleware/observability.go`
- `pkg/server/router/router.go`
- `pkg/server/middleware/observability_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13114/head:upstream-pr-13114`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
