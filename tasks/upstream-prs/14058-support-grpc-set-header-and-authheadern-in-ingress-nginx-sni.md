# Support grpc_set_header and $authHeaderN in Ingress NGINX snippets

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14058](https://github.com/traefik/traefik/pull/14058) |
| Tipo | Recurso |
| Área | provider/ingress-nginx, middleware/ingressnginx/snippet |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | daniel-garcia |
| Aberto em | 2026-10-12 (última atualização: 2026-10-12) |
| Draft | sim |
| Branch-alvo | `master` |
| Tamanho | +64/-4 em 5 arquivos |
| Labels | — |
| Status | pendente |

## O que faz

Adiciona a diretiva grpc_set_header (server e location), equivalente a proxy_set_header, e as variaveis $authHeaderN com os valores de auth-response-headers apos auth-url. Usado com frequencia em backends gRPC do ingress-nginx. Resolve #14042.

## Riscos e situação no upstream

Em draft, sem reviews. Escopo pequeno (+64/-4, 5 arquivos). Depende da correcao de aspas do #14044 para o caso mais comum de grpc_set_header 'Nome' valor.

## Principais arquivos

- `pkg/middlewares/ingressnginx/snippet/snippet.go`
- `pkg/middlewares/ingressnginx/snippet/collector.go`
- `pkg/middlewares/ingressnginx/snippet/directive_context.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14058/head:upstream-pr-14058`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
