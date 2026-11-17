# Add noTimestamp option for logs and access logs

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14043](https://github.com/traefik/traefik/pull/14043) |
| Tipo | Recurso |
| Área | logs, accesslogs |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | AdamMagued |
| Aberto em | 2026-10-11 (última atualização: 2026-11-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +306/-26 em 9 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, area/logs, size/L, area/accesslogs |
| Status | pendente |

## O que faz

Novas opcoes log.noTimestamp e accesslog.noTimestamp para suprimir o timestamp interno dos logs (texto e JSON) e dos access logs, quando o coletor (journald, Docker, Kubernetes) ja adiciona o seu. Resolve #14000.

## Riscos e situação no upstream

9 arquivos (+306/-26) tocando formatters e tipos de log. Label status/1-needs-design-review. Maintainer sugeriu inverter a opcao (ex.: timestamp: true) para evitar opcao negativa; API ainda pode mudar.

## Principais arquivos

- `cmd/traefik/logger.go`
- `pkg/observability/logs/log.go`
- `pkg/middlewares/accesslog/logger_formatters.go`
- `pkg/observability/types/logs.go`
- `docs/content/reference/install-configuration/observability/logs-and-accesslogs.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14043/head:upstream-pr-14043`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
