# Support new connections to terminating endpoints for zero-downtime

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12419](https://github.com/traefik/traefik/pull/12419) |
| Tipo | Correção |
| Área | server/loadbalancer/wrr, provider/kubernetes |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | PartlyFluked |
| Aberto em | 2025-12-13 (última atualização: 2025-12-15) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +306/-15 em 4 arquivos |
| Labels | status/1-needs-design-review, size/L, area/server, area/provider/k8s |
| Status | importado com ajustes |

## O que faz

Quando todos os endpoints do Kubernetes estão terminando mas ainda servindo (serving=true), o WRR passa a mandar novas conexões para eles em vez de responder 503. Isso cobre deploys com réplica única ou estratégia Recreate, onde hoje há downtime.

## Riscos e situação no upstream

Muda o comportamento de roteamento do WRR no caminho de falha. Status design-review (maintainers vão checar impacto antes de seguir). Toca docs de Kubernetes CRD e Ingress.

## Principais arquivos

- `pkg/server/service/loadbalancer/wrr/wrr.go`
- `pkg/server/service/loadbalancer/wrr/wrr_test.go`
- `docs/content/routing/providers/kubernetes-crd.md`
- `docs/content/routing/providers/kubernetes-ingress.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12419/head:upstream-pr-12419`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:
- O novo `nextServer` limita a busca a `len(b.handlers)` pops do heap EDF. Com pesos diferentes, um handler pesado sai várias vezes antes de um leve, então um servidor saudável e não fenced de peso baixo podia nunca ser visitado: o WRR cairia no fallback fenced ou devolveria 503 mesmo com servidor saudável disponível (regressão).
- Bug pré-existente no fork, corrigido de quebra: se o único servidor saudável é fenced e os não fenced estão DOWN (ex.: health check passivo), a checagem `len(b.fenced) == len(b.handlers)` passava e o `for` girava para sempre segurando `handlersMu`, travando todas as requests do serviço (DoS).
- Só o WRR mudava; P2C, HRW e LeastTime continuariam respondendo 503 no mesmo cenário.
- Testes do PR verbosos, com mensagens em assert e casos redundantes.
- As páginas de docs que o PR altera (`docs/content/routing/providers/...`) não existem mais no fork.

Ajustes:
- WRR: antes do loop, varre o heap para saber se há handler saudável e se algum saudável não é fenced; fenced só é pulado quando existe alternativa. O loop termina sempre, e a escolha entre fenced continua respeitando os pesos.
- Mesmo fallback (saudáveis não fenced, senão saudáveis fenced) em P2C, HRW e LeastTime.
- Testes: `TestBalancerAllServersFenced`/`TestBalancerAllFencedServers` agora esperam 200; novos `TestBalancerFencedFallback` (cobre o antigo loop infinito e a volta ao servidor saudável) e `TestBalancerFencedNotPreferredOverLowWeight` (cobre a regressão do PR).
- Nota em `docs/content/reference/routing-configuration/http/load-balancing/service.md`.

Testes: `go test ./pkg/server/service/loadbalancer/...` ok (o `-race` não rodou até o fim por falta de espaço em disco na máquina, compartilhada com outros builds); `golangci-lint run ./pkg/server/service/loadbalancer/...` 0 issues.
