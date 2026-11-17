# Count load balancer 503 responses in service metrics

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14299](https://github.com/traefik/traefik/pull/14299) |
| Tipo | Correção |
| Área | service/metrics |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | davidgrldo |
| Aberto em | 2026-11-15 (última atualização: 2026-11-16) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +175/-13 em 2 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/M, area/middleware/metrics, area/service |
| Status | importado com ajustes |

## O que faz

Respostas geradas antes de escolher um backend, como o 503 quando nenhum servidor está disponível, não entravam nas métricas do service. O middleware de métricas passa a envolver o load balancer HTTP inteiro. Corrige #12451.

## Riscos e situação no upstream

Base é v3.7, precisa de port. Muda onde o middleware de métricas é aplicado, o que pode alterar contagens de métricas existentes. Status status/1-needs-design-review. Sem review de maintainer ainda.

## Principais arquivos

- `pkg/server/service (load balancer HTTP e middleware de métricas)`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14299/head:upstream-pr-14299`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- Fixes #12451: o middleware de métricas de serviço envolvia cada servidor, então respostas geradas pelo load balancer antes de escolher um servidor (503 sem servidor disponível) não entravam nas métricas do serviço, justamente o caso que alertas precisam ver.
- Em `status/1-needs-design-review`. A primeira versão invertia a ordem dos spans (Metrics/Service) e quebrava o `TestTracingSuite`; o autor incorporou a correção do #14301 (span `Service` também movido para o load balancer) e adicionou teste da ordem. PR declarado como assistido por IA (Codex).
- Efeitos verificados: o nome do serviço continua qualificado (`BuildHTTP` qualifica antes); o `StatusUpdater` do health check é capturado antes do novo wrapper; o gauge de in-flight do #13221 passa a contar também as requisições respondidas pelo próprio load balancer; serviços weighted/mirroring/failover não mudam (seus filhos LB são instrumentados).
- Mudança de semântica pequena: o span `Service` passa a ser um por chamada ao load balancer, não um por servidor.
- Sem impacto de segurança nem dependência nova.

### Ajustes

- Teste `service_metrics_test.go` adaptado às assinaturas do Traefix: `CounterWithHeaders.With` com headers de resposta (#12073) e `NewObservabilityMgr` com a função de rotação de log (#12752).

### Testes

- `go vet ./pkg/server/...`: ok.
- `go test ./pkg/server/service/ ./pkg/server/`: ok.
- `golangci-lint run ./pkg/server/service/...`: 0 issues.
- `TestTracingSuite` (integração, Docker) não foi rodado; vale rodar antes do deploy, pois é a suíte que pegou a regressão de ordem de spans no upstream.
