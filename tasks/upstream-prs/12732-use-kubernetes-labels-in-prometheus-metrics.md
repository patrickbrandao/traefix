# Use Kubernetes labels in Prometheus metrics

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12732](https://github.com/traefik/traefik/pull/12732) |
| Tipo | Recurso |
| Área | observability/metrics/prometheus |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | sathieu |
| Aberto em | 2026-02-25 (última atualização: 2026-10-04) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +101/-44 em 5 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/middleware/metrics |
| Status | adiado |

## O que faz

Com a flag --metrics.prometheus.addKubernetesLabels, adiciona namespace, kind e name do recurso Kubernetes às métricas Prometheus. Permite dashboards e HPA por ingress. Fecha a issue #6864.

## Riscos e situação no upstream

Depende de PRs ainda não merged (#12721 e #12913, que o autor disse ter rebaseado). Status design-review. Cuidado com a cardinalidade de labels. Maintainers não responderam ao follow-up.

## Principais arquivos

- `pkg/observability/metrics/prometheus.go`
- `pkg/observability/metrics/metrics.go`
- `pkg/middlewares/metrics/metrics.go`
- `pkg/observability/types/metrics.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12732/head:upstream-pr-12732`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- Em `status/1-needs-design-review`: os maintainers disseram que querem definir o conjunto completo de labels (equivalente ao ingress-nginx) antes de mudar as séries, para não quebrar dashboards depois. Sem decisão até agora.
- Os labels são lidos do `accesslog.LogData` do contexto, que só existe quando o access log está habilitado: sem access log, todas as séries saem com `namespace="none"`, `name="none"`, sem aviso.
- `NeedsKubernetesLabels` no `MultiRegistry` vira `true` se qualquer registry pedir, e os labels extras passam a ir também para Datadog/StatsD/InfluxDB/OTLP, que não declararam essas dimensões.
- `kind` é fixo em `ingress` (CRD/Gateway API não cobertos); type assertions `v.(string)` sem checagem podem dar panic se o tipo do campo mudar.
- Sem testes e sem documentação (checklist do PR desmarcado).
- Cardinalidade: aumenta as séries por namespace/ingress/serviço (limitado ao número de recursos, aceitável, mas multiplica todas as séries HTTP).

### Ajustes

Nenhum.

### Testes

Não aplicável.

### Motivo

Adiado: design ainda em aberto no upstream (conjunto de labels não definido), dependência implícita do access log e vazamento dos labels para os outros backends de métricas. Reavaliar quando o upstream fechar o design.
