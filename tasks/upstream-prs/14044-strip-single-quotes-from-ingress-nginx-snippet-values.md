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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

**Achados no PR original**

- `trimQuote` só removia aspas duplas. Valores entre aspas simples, que são válidos no NGINX (`proxy_set_header 'X-A' 'v'`, `location = '/x'`), ficavam com as aspas: os headers saíam com o nome literal `'X-A'` e, desde #13770, os padrões de `location` entre aspas simples nunca casavam. Isso pode fazer uma regra `deny` ser ignorada em silêncio.
- A correção só remove o par quando as aspas de abertura e fechamento são iguais; pares misturados (`'foo"`) ficam como estão.
- Sem risco novo: a mudança só afeta como os valores do snippet são interpretados.

**Ajustes**

- `TestTrimQuote` usa `assert.Equal` no lugar de `t.Fatalf`, seguindo a convenção do projeto.
- Como `trimQuote` também é usado no casamento de `location` (#13770), aspas simples passam a valer ali também.

**Testes**

- `go build ./...` ok.
- `go test ./pkg/middlewares/ingressnginx/snippet/` ok.
- `golangci-lint run ./pkg/middlewares/ingressnginx/...`: 0 issues.
