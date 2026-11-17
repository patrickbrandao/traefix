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
| Status | pendente |

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
