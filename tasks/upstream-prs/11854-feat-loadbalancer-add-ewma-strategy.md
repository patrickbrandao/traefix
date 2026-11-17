# feat(loadbalancer): add ewma strategy

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11854](https://github.com/traefik/traefik/pull/11854) |
| Tipo | Recurso |
| Área | loadbalancer |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | vring0 |
| Aberto em | 2025-06-24 (última atualização: 2026-04-08) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +1152/-23 em 13 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/provider/k8s/crd, area/provider/k8s |
| Status | rejeitado |

## O que faz

Adiciona balanceador Peak-EWMA com power-of-two-choices: sorteia dois backends saudáveis e escolhe o de menor RTT suavizado por EWMA. Visa latência melhor em cargas com picos e JVM em warm-up.

## Riscos e situação no upstream

Novo pacote loadbalancer/ewma e mudança no CRD (13 arquivos, +1152/-23). Label needs-design-review; maintainer comentou que será enhancement para versão menor futura. Usuários pedem andamento, sem review formal.

## Principais arquivos

- `pkg/server/service/loadbalancer/ewma/ewma.go`
- `pkg/server/service/service.go`
- `pkg/config/dynamic/http_config.go`
- `pkg/provider/kubernetes/crd/kubernetes_http.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11854/head:upstream-pr-11854`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: rejeitado.**

Achados no PR original:

- O estado EWMA fica num `map` global de pacote (`ewmaRegistry`), indexado pela URL do servidor. Serviços diferentes que apontam para a mesma URL dividem a mesma medida de latência. O estado também sobrevive a reloads sem ligação com o ciclo de vida do balanceador. A limpeza (`cleanupRegistry`) só roda dentro de `AddServer`.
- `SetStatus` nunca chama os `updaters` registrados em `RegisterStatusUpdater`. Com isso, um serviço pai (WRR/failover com healthcheck) nunca fica sabendo quando todos os filhos caem. É uma regressão em relação aos outros balanceadores.
- `AddServer` reconstrói a lista inteira de handlers a cada servidor adicionado (O(n²)). Usa `http.NotFoundHandler()` como fallback silencioso e ignora `weight`.
- A latência medida inclui o streaming do corpo inteiro: WebSocket, SSE e downloads grandes "envenenam" a média. Também não há contagem de requisições em andamento, então não é Peak-EWMA de verdade.
- A estratégia `leasttime`, já presente no código, cobre o mesmo caso de uso (TTFB + inflight).
- No upstream o PR está em `status/1-needs-design-review`, sem review técnico, e mexe nos CRDs.

Nenhum código importado e nenhum teste rodado.
