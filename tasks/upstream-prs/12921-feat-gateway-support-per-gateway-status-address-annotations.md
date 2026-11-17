# feat(gateway): support per-Gateway status address annotations

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12921](https://github.com/traefik/traefik/pull/12921) |
| Tipo | Recurso |
| Área | provider/kubernetes/gateway-api |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | jscaltreto |
| Aberto em | 2026-04-01 (última atualização: 2026-05-20) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +402/-12 em 5 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/provider/k8s/gatewayapi |
| Status | pendente |

## O que faz

Permite sobrescrever o statusAddress estático por Gateway através de annotations (ip, hostname, service), espelhando as opções estáticas. Assim, Gateways atrás de Services diferentes (interno/público) passam a reportar endereços distintos em .status.addresses, o que o external-dns precisa.

## Riscos e situação no upstream

Label status/1-needs-design-review; maintainer abriu design review em abril e não houve revisão de código depois; autor vem cobrando. Toca annotations.go e kubernetes.go do provider gateway, sem conflito previsto com o fork.

## Principais arquivos

- `pkg/provider/kubernetes/gateway/annotations.go`
- `pkg/provider/kubernetes/gateway/kubernetes.go`
- `pkg/provider/kubernetes/gateway/annotations_test.go`
- `docs/content/reference/routing-configuration/kubernetes/gateway-api.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12921/head:upstream-pr-12921`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
