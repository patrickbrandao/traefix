# Keep canary routers at their location priority on ingress-nginx provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13789](https://github.com/traefik/traefik/pull/13789) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | lazerg |
| Aberto em | 2026-08-27 (última atualização: 2026-10-12) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +176/-0 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/ingress-nginx |
| Status | importado com ajustes |

## O que faz

Routers de canary herdam a regra de location com a condição de canary e ganham prioridade pelo tamanho da regra. Assim, um canary em / pode vencer um path mais específico como /data/ e mandar requisições do cookie/header para o backend errado. O patch fixa a prioridade em location + 1.

## Riscos e situação no upstream

Base v3.7, precisa de adaptação. Pequeno. Corrige #13785. Status/2-needs-review. O autor aponta um caso residual: canary em / fica uma posição acima de PathPrefix('/'), empatando com Exact.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `pkg/provider/kubernetes/ingress-nginx/translator.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13789/head:upstream-pr-13789`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- **Roteamento incorreto (bug real):** os routers `-canary` e `-non-canary` usam a regra da location estendida com o critério do canary (cookie, header). Como a prioridade padrão é o tamanho da regra, eles passavam à frente dos routers de paths mais longos do mesmo host. Com um Ingress em `/data/` e `/` e um canary em `/`, o router `-non-canary` de `/` (que casa com tudo que não tem o cookie) capturava também as requisições para `/data/...`, que iam parar no backend errado, com ou sem o cookie. O ingress-nginx usa a regra do path mais longo.
- A correção (prioridade da location + 1) segue o padrão já usado no código para o router de `limit-allowlist`.
- Limitação conhecida, já apontada no PR e na revisão automática: o +1 pode empatar com outro router cuja regra tenha exatamente um caractere a mais (por exemplo, um `Exact` de 8 caracteres contra o canary de `/`). Nesse caso o desempate do muxer não é determinístico. O risco é bem menor que o do comportamento atual.

**Ajustes**

- Se o router base já tiver `Priority` explícita, ela é respeitada, como no `limit-allowlist`; o PR sempre recalculava pelo tamanho da regra.
- Teste de ponta a ponta com o muxer real (`TestCanaryRouterPriority`), adaptado ao formato atual das chaves de router.
- Interação com `limit-allowlist` conferida: o router de allowlist derivado do canary cai de 112 (tamanho da regra) para 49 (prioridade do canary + 1). A ordem dentro da location fica: base 47, base-allowlist 48, canary 48, canary-allowlist 49. O empate entre base-allowlist e canary só acontece quando os dois casam (cookie + IP permitido), e nesse caso o canary-allowlist (49) tem prioridade maior e também casa, então o resultado é determinístico.
- As expectativas de `Priority` dos routers canary/non-canary em `TestLoadIngresses` foram atualizadas por script (tamanho da regra base + 1).

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress-nginx/` ok.
- `golangci-lint run ./pkg/provider/kubernetes/ingress-nginx/...`: 0 issues.
