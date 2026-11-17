# Fix OTel k8s attributes detection when using Pod hostNetwork

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12526](https://github.com/traefik/traefik/pull/12526) |
| Tipo | Correção |
| Área | observability/otel, provider/kubernetes |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | applejag |
| Aberto em | 2026-01-08 (última atualização: 2026-08-26) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +25/-4 em 1 arquivos |
| Labels | status/0-needs-triage, size/S, area/provider/k8s |
| Status | pendente |

## O que faz

Com hostNetwork: true, o K8sAttributesDetector do OpenTelemetry falha ao descobrir nome e namespace do pod. A correção lê essas informações de variáveis de ambiente. Fecha a issue #12525.

## Riscos e situação no upstream

Pequeno (1 arquivo), mas o PR é contra a branch v3.7 (maintenance), não master, então precisa adaptação. Sem testes. Só triagem. Existe workaround via OTEL_RESOURCE_ATTRIBUTES.

## Principais arquivos

- `pkg/types/k8sdetector.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12526/head:upstream-pr-12526`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
