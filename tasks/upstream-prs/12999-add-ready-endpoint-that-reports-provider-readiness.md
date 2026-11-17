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
| Status | adiado |

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

## Revisão (2026-11-16)

### Achados

- Sem nenhum comentário ou review de maintainer; é feature nova (endpoint `/ready`) que o upstream ainda não discutiu (issue #10458 aberta).
- Bug de correção grave: a prontidão é calculada por `len(newConfigs) >= expectedProviderCount` em `applyConfigurations`, mas `receiveConfigurations` descarta mensagens vazias (`Skipping empty configuration`). Um provider sem recursos (Docker sem containers, CRD sem IngressRoute, Consul sem serviços...) nunca entra no mapa e o `/ready` fica em 503 para sempre: o pod nunca fica Ready no Kubernetes.
- O provider `rest` entra na contagem, mas o `Provide` dele não envia nada até o primeiro PUT: com `providers.rest` habilitado o `/ready` também nunca fica pronto. O mesmo vale para plugins de provider que só enviam sob demanda.
- Rota `PathPrefix(/ready)` com prioridade máxima: se `ready.entryPoint` apontar para um entrypoint público, captura qualquer caminho que comece com `/ready` (`/readyz`, `/ready-made`...). Deveria ser `Path`.
- Exposição: só devolve status (sem dados), no entrypoint interno `traefik` por padrão, como o `/ping`; aceitável.

### Ajustes

Nenhum.

### Testes

Não aplicável.

### Motivo

Adiado: a lógica de prontidão precisa ser refeita (contar providers que já reportaram, inclusive com configuração vazia, e excluir providers sob demanda como `rest`) antes de ser confiável como readiness probe; sem isso o endpoint pode travar rollouts. Reavaliar quando o upstream discutir o design.
