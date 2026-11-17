# Support gRPC timeout directives in ingress-nginx snippets

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14059](https://github.com/traefik/traefik/pull/14059) |
| Tipo | Recurso |
| Área | provider/ingress-nginx |
| Prioridade | **P4** |
| Valor | baixa |
| Esforço de port | alto |
| Autor | daniel-garcia |
| Aberto em | 2026-10-12 (última atualização: 2026-10-12) |
| Draft | sim |
| Branch-alvo | `master` |
| Tamanho | +703/-10 em 13 arquivos |
| Labels | — |
| Status | pendente |

## O que faz

Suporta grpc_connect_timeout, grpc_read_timeout e grpc_send_timeout em snippets do ingress-nginx, aplicando-os ao ServersTransport (dialTimeout, readTimeout, writeTimeout) para backends GRPC/GRPCS. Annotations proxy-*-timeout tem precedencia.

## Riscos e situação no upstream

Em draft, sem reviews. Grande (+703/-10, 13 arquivos) e com varias regras de precedencia e parsing de tempo do NGINX. Depende do fluxo de snippets tambem tocado pelo #14058.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/grpc_timeouts.go`
- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/middlewares/ingressnginx/snippet/collector.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14059/head:upstream-pr-14059`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
