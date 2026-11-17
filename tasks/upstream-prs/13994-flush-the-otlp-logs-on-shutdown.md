# Flush the OTLP logs on shutdown

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13994](https://github.com/traefik/traefik/pull/13994) |
| Tipo | Correção |
| Área | logs/otel |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | mloiseleur |
| Aberto em | 2026-09-30 (última atualização: 2026-10-06) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +97/-12 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, area/logs, size/M, area/otel |
| Status | pendente |

## O que faz

SetupOTelLogger passa a retornar um io.Closer que encerra o logger provider OTLP, e runCmd o chama antes de svr.Close(). Assim os logs emitidos durante o shutdown e os enfileirados sao exportados.

## Riscos e situação no upstream

Muda a assinatura de SetupOTelLogger (funcao exportada, mas com um unico caller no repo). Autor pergunta se o maintainer prefere manter a assinatura e expor funcao de shutdown separada, como metrics.StopOpenTelemetry. Tres reviews COMMENTED.

## Principais arquivos

- `cmd/traefik/logger.go`
- `cmd/traefik/traefik.go`
- `pkg/observability/logs/otel.go`
- `pkg/observability/logs/otel_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13994/head:upstream-pr-13994`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
