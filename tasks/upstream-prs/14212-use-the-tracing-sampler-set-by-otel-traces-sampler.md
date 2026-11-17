# Use the tracing sampler set by OTEL_TRACES_SAMPLER

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14212](https://github.com/traefik/traefik/pull/14212) |
| Tipo | Correção |
| Área | middleware/tracing, observability/otel |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | ShikhJohari |
| Aberto em | 2026-11-03 (última atualização: 2026-11-10) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +216/-4 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/middleware/tracing, area/otel |
| Status | pendente |

## O que faz

As variáveis OTEL_TRACES_SAMPLER e OTEL_TRACES_SAMPLER_ARG eram ignoradas, porque o Traefik sempre passava o próprio sampler. Agora, se o env define um sampler suportado pelo SDK, ele vence. Se o valor é inválido, o Traefik loga aviso e mantém sampleRate, sem ligar amostragem total.

## Riscos e situação no upstream

Muda a precedência entre env e sampleRate; quem já define OTEL_TRACES_SAMPLER (ex.: Operator) passa a usar o env. Status: needs-review, base v3.7. Tem docs e testes.

## Principais arquivos

- `pkg/observability/types/tracing.go`
- `pkg/observability/types/tracing_test.go`
- `docs/content/reference/install-configuration/observability/tracing.md`
- `docs/content/migrate/v3.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14212/head:upstream-pr-14212`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
