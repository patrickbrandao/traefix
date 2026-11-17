# fix(ingress-nginx): enforce proxy-body-size when proxy-request-buffering is off

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13220](https://github.com/traefik/traefik/pull/13220) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx, middlewares/buffering |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | filip2mac |
| Aberto em | 2026-05-21 (última atualização: 2026-09-30) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +275/-15 em 6 arquivos |
| Labels | contributor/waiting-for-corrections, kind/bug/fix, size/M, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Corrige proxy-body-size ignorado quando proxy-request-buffering está off. O limite passa a ser lido antes do early return e força o buffering de requisição, como o client_max_body_size do Nginx. Corrige o comportamento que hoje deixa passar bodies grandes.

## Riscos e situação no upstream

CHANGES_REQUESTED de um maintainer, e a última rodada de testes quebrou porque ProxyBodySize tem padrão de 1MB e o buffering passou a ser aplicado a todos. Precisa de retrabalho antes de revisão. Toca buffering.go e proxy/httputil/proxy.go.

## Principais arquivos

- `pkg/middlewares/buffering/buffering.go`
- `pkg/middlewares/buffering/buffering_test.go`
- `pkg/provider/kubernetes/ingress-nginx/middleware.go`
- `pkg/proxy/httputil/proxy.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13220/head:upstream-pr-13220`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
