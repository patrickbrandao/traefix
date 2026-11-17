# Warn when routing configuration is found in install configuration

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12230](https://github.com/traefik/traefik/pull/12230) |
| Tipo | Recurso |
| Área | server/config-validation |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | medio |
| Autor | rtribotte |
| Aberto em | 2025-11-05 (última atualização: 2026-09-18) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +665/-4 em 4 arquivos |
| Labels | status/2-needs-review, contributor/waiting-for-corrections, kind/bug/fix, size/L, bot/no-merge, area/server |
| Status | pendente |

## O que faz

Adiciona um aviso no startup quando há configuração de roteamento no arquivo de instalação, para evitar confusão do operador. Resolve a issue 12098.

## Riscos e situação no upstream

Base v3.7, mexe no startup do cmd/traefik (4 arquivos, +665/-4). Tem label bot/no-merge e review com CHANGES_REQUESTED. Uso de valor limitado.

## Principais arquivos

- `cmd/traefik/traefik.go`
- `pkg/cli/routing_warning.go`
- `pkg/cli/deprecation.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12230/head:upstream-pr-12230`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
