# Add AccessGovernance middleware for bounded access-event persistence

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14076](https://github.com/traefik/traefik/pull/14076) |
| Tipo | Recurso |
| Área | middleware/accessGovernance, provider/kubernetes-crd, udp |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | bjzhangshuai |
| Aberto em | 2026-10-15 (última atualização: 2026-10-23) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +8677/-38 em 92 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/middleware, area/provider/k8s/crd, area/udp |
| Status | pendente |

## O que faz

Middleware opt-in para persistir eventos de acesso (HTTP, TCP, UDP) em destinos MySQL nomeados, com filas limitadas, escrita em lote e pools limitados. Falhas do banco nao derrubam a requisicao, e eventos descartados sao logados. Inclui suporte a CRD e cadeia de middlewares UDP.

## Riscos e situação no upstream

Enorme (+8677/-38, 92 arquivos, go.mod/go.sum novos, dependencia de MySQL). Label status/1-needs-design-review; sete reviews COMMENTED. Tem muita superficie (credenciais, rotacao, UDP) e precisa de alinhamento com maintainers antes de qualquer port.

## Principais arquivos

- `go.mod`
- `go.sum`
- `integration/accessgovernance_test.go`
- `docs/content/reference/routing-configuration/http/middlewares/accessgovernance.md`
- `docs/content/reference/routing-configuration/udp/middlewares/accessgovernance.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14076/head:upstream-pr-14076`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
