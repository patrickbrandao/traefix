# Fix OTLP gRPC exporters with an IPv6 collector endpoint

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14234](https://github.com/traefik/traefik/pull/14234) |
| Tipo | Correção |
| Área | observability/otel (tracing, metrics, logs) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | b84330808 |
| Aberto em | 2026-11-05 (última atualização: 2026-11-09) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +137/-3 em 5 arquivos |
| Labels | kind/bug/fix, size/M, area/otel |
| Status | importado com ajustes |

## O que faz

Os três exportadores OTLP gRPC remontavam o endpoint com fmt.Sprintf, perdendo os colchetes de endereços IPv6, e por isso nunca conectavam a um coletor como [::1]:4317. O PR usa net.JoinHostPort e adiciona testes por exportador.

## Riscos e situação no upstream

Base é o branch v3.7, então o port para o master precisa de ajuste. O bug existe no master (Sprintf confirmado em pkg/observability/types/tracing.go, metrics/otel.go e types/logs.go). Afeta só endpoints IPv6 com gRPC. Sem review de maintainer ainda.

## Principais arquivos

- `pkg/observability/types/tracing.go`
- `pkg/observability/metrics/otel.go`
- `pkg/observability/types/logs.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14234/head:upstream-pr-14234`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- Correção pequena e correta: os três exporters OTLP gRPC (traces, métricas, logs) remontavam o endereço com `fmt.Sprintf("%s:%s")`, perdendo os colchetes do IPv6 (`::1:4317`), e nunca conectavam a um collector IPv6. `net.JoinHostPort` resolve.
- Sem review de maintainer; sem impacto de segurança. O autor não rodou lint nem a suíte completa.
- O teste novo de métricas chama `newGRPCExporter` com a assinatura antiga; no Traefix ela recebe o seletor de temporalidade (#12789).

### Ajustes

- Teste `TestNewGRPCExporter_IPv6Endpoint` adaptado para a assinatura com `TemporalitySelector` (`nil`, default do SDK).

### Testes

- `go vet ./pkg/observability/...`: ok.
- `go test ./pkg/observability/types/ ./pkg/observability/metrics/ ./pkg/observability/tracing/`: ok; os três testes IPv6 rodaram (não foram pulados) e passaram.
- `golangci-lint run ./pkg/observability/...`: 0 issues.
