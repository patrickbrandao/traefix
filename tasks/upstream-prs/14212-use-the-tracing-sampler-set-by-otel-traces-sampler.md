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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

### Achados

- Fixes #11334, abordagem sugerida por maintainer na issue (suportar `OTEL_TRACES_SAMPLER`/`OTEL_TRACES_SAMPLER_ARG` em vez de opção nova). Reviews: automatizadas (Aikido, agente do Traefik, sem achados) e do autor.
- Correção real: o SDK aplica o sampler do ambiente antes das opções, então o `WithSampler` explícito do Traefik sempre anulava as variáveis.
- Segurança/custo: se o valor do ambiente for inválido, o SDK amostra 100% das raízes. O PR valida nome e razão (inclusive NaN) antes de ceder ao ambiente e, em caso de erro, mantém `sampleRate` com warning: um typo não liga amostragem total. A lista de samplers bate com `sdk/trace/sampler_env.go` da versão do SDK no `go.mod`.
- Mudança de comportamento: quem já tem `OTEL_TRACES_SAMPLER` definido (ex.: OpenTelemetry Operator) passa a usá-lo em vez de `tracing.sampleRate`. Documentado.

### Ajustes

- Conflito em `docs/content/migrate/v3.md`: o PR documentava a mudança em `v3.7.19`; no Traefix a nota foi colocada na seção `v3.8.0` (base do fork).
- Código e testes aplicados como estão.

### Testes

- `go test ./pkg/observability/types/`: ok.
- `golangci-lint run ./pkg/observability/types/...`: 0 issues.
