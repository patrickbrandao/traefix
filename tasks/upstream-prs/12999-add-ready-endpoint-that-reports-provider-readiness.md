# Add ready endpoint that reports provider readiness

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12999](https://github.com/traefik/traefik/pull/12999) |
| Tipo | Recurso |
| Área | observability/healthcheck, server |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | darkweaver87 |
| Aberto em | 2026-04-17 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +438/-5 em 12 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L |
| Status | pendente |

## O que faz

Adiciona o endpoint /ready, que responde 200 só depois que todos os providers habilitados carregaram a configuração inicial e a troca de roteador foi aplicada. Ao contrário do /ping, ele indica que as rotas já estão sendo servidas, o que ajuda readiness probes em Kubernetes.

## Riscos e situação no upstream

Label needs-design-review e sem comentários de maintainer. Implementa o primeiro passo do issue #10458 e toca cmd/traefik, configurationwatcher, aggregator e managerfactory, então precisa validar a ordem de inicialização e o shutdown gracioso.

## Principais arquivos

- `pkg/ready/ready.go`
- `pkg/server/configurationwatcher.go`
- `pkg/provider/aggregator/aggregator.go`
- `pkg/server/service/managerfactory.go`
- `cmd/traefik/traefik.go`
- `pkg/config/static/static_config.go`
- `docs/content/reference/install-configuration/observability/healthcheck.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12999/head:upstream-pr-12999`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
