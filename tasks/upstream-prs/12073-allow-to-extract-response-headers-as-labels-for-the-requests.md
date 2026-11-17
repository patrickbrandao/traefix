# Allow to extract response headers as labels for the requests_total metrics

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12073](https://github.com/traefik/traefik/pull/12073) |
| Tipo | Recurso |
| Área | middleware/metrics |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | jybp |
| Aberto em | 2025-09-16 (última atualização: 2026-03-12) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +306/-53 em 18 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/middleware/metrics |
| Status | importado com ajustes |

## O que faz

Adiciona responseHeaderLabels às métricas requests_total do Prometheus, espelhando headerLabels existente para headers de requisição. Útil para labels como X-Cache-Status ou headers próprios das aplicações.

## Riscos e situação no upstream

Toca 18 arquivos, boa parte testes e config. Label status/2-needs-review, sem reviews. Cuidado com cardinalidade alta se o operador escolher headers variáveis.

## Principais arquivos

- `pkg/middlewares/metrics/metrics.go`
- `pkg/observability/metrics/headers.go`
- `pkg/observability/metrics/prometheus.go`
- `pkg/observability/types/metrics.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12073/head:upstream-pr-12073`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- Sem review de maintainer; feature opt-in (`metrics.prometheus.responseHeaderLabels`), só no Prometheus.
- Bug: um mesmo nome de label em `headerLabels` e `responseHeaderLabels` gera nomes duplicados no `CounterVec`; o `With` do Prometheus entra em panic a cada requisição.
- Risco de cardinalidade: cada valor distinto do header vira série em memória. Os valores vêm do backend (não do cliente, como em `headerLabels`), mas um header com IDs ou datas cresce sem limite.
- O PR adicionava `integration/fixtures/grpc/config_1598140383.toml`, um arquivo temporário gerado por teste (com chave privada de teste), lixo sem uso.
- Nenhum teste novo exercitava de fato os headers de resposta (só ajustes de assinatura).

### Ajustes

- Labels de resposta que repetem um label de request são ignorados com warning (o request vence), evitando o panic.
- Removido o fixture temporário; `cli-ref.md`/`env-ref.md` mantidos como estão no Traefix; `docs/content/observability/metrics/prometheus.md` não existe mais e ficou de fora.
- Doc: aviso de cardinalidade e da regra de colisão.
- Teste novo `TestCounterWithHeaders_responseHeaderLabels` (valor do header de resposta e colisão sem panic).
- `go generate` rodado: docs gerados consistentes.

### Testes

- `go build ./...`: ok.
- `go test ./pkg/observability/metrics/ ./pkg/middlewares/metrics/`: ok (`TestInfluxDB2` é instável por timing e falhou uma vez; passa isolado e na reexecução, sem relação com a mudança).
- `golangci-lint run ./pkg/observability/metrics/... ./pkg/middlewares/metrics/...`: 0 issues.
