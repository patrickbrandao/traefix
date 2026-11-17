# Strip single quotes from ingress-nginx snippet values

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14044](https://github.com/traefik/traefik/pull/14044) |
| Tipo | Correção |
| Área | provider/ingress-nginx, middleware/ingressnginx/snippet |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | SashaMIT |
| Aberto em | 2026-10-11 (última atualização: 2026-10-12) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +40/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Remove o par de aspas simples ou duplas que envolve nomes e valores em snippets do ingress-nginx (ex.: grpc_set_header 'Authorization' ...). Antes o header era enviado com as aspas no nome, e o valor com elas. Resolve #14041.

## Riscos e situação no upstream

Pequeno (+40, 2 arquivos). Um review COMMENTED; autor relata caso real com patch local. Mexe apenas no parser de snippet.

## Principais arquivos

- `pkg/middlewares/ingressnginx/snippet/action.go`
- `pkg/middlewares/ingressnginx/snippet/snippet_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14044/head:upstream-pr-14044`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
