# fix(kubernetes-ingress-nginx): preserve original auth URI

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13754](https://github.com/traefik/traefik/pull/13754) |
| Tipo | Correção |
| Área | middleware/ingressnginx, provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | fredrikblau |
| Aberto em | 2026-08-24 (última atualização: 2026-11-16) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +75/-5 em 6 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/middleware, area/provider/k8s/ingress-nginx |
| Status | importado com ajustes |

## O que faz

Quando rewrite-target reescreve o path, o X-Forwarded-Uri original se perde no forward-auth do ingress-nginx. O PR guarda a URI original antes da reescrita e a usa na requisição de auth. Corrige o #13682.

## Riscos e situação no upstream

Base master, pequeno (75 linhas). Atualizado em 2026-11-16, ainda em status/2-needs-review. Mexe no middleware de rewrite e no forward-auth do ingress-nginx.

## Principais arquivos

- `pkg/middlewares/ingressnginx/original_uri.go`
- `pkg/middlewares/ingressnginx/rewritetarget/rewrite_target.go`
- `pkg/middlewares/ingressnginx/rewritetarget/rewrite_target_test.go`
- `pkg/middlewares/ingressnginx/snippet/snippet.go`
- `pkg/middlewares/ingressnginx/snippet/snippet_test.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13754/head:upstream-pr-13754`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- **Segurança (autorização):** com `auth-url` combinado com `rewrite-target`, o serviço de autenticação recebia em `X-Forwarded-Uri` o caminho **reescrito**. Um serviço que decide pelo caminho (por exemplo, "só admins em `/admin`") avaliava a entrada errada. Uma regra de rewrite que remove o prefixo levava o serviço a autorizar `/secret` quando o cliente pediu `/admin/secret`. O PR guarda a URI original no contexto, dentro do `rewrite-target`, e o forward auth usa essa URI. É o comportamento do ingress-nginx.
- **Lacuna do PR:** a documentação do próprio PR cita "ou uma diretiva `rewrite` em `configuration-snippet`/`auth-snippet`", mas só o `rewrite-target` guardava a URI original. Um `rewrite` dentro do snippet altera `req.URL` antes do forward auth (ambos estão no mesmo middleware) e continuava mandando o caminho reescrito ao serviço de autenticação.
- `$request_uri` (e `$escaped_request_uri`), que no NGINX é sempre a URI do cliente, também refletia o caminho reescrito, inclusive dentro de `auth-url` interpolado.

**Ajustes**

- `snippet.ServeHTTP` guarda a URI original antes de executar as diretivas, se ainda não houver uma.
- Novo helper `ingressnginx.OriginalRequestURI` (com fallback para `req.URL.RequestURI()`), usado no `X-Forwarded-Uri` e na interpolação de `$request_uri`/`$escaped_request_uri`. `$uri` continua refletindo o caminho atual, como no NGINX.
- Testes novos: rewrite em `configuration-snippet` seguido de forward auth (`X-Forwarded-Uri` original, upstream reescrito) e `$request_uri` depois de um rewrite.
- O valor guardado fica só no contexto da requisição; o cliente não tem como forjá-lo.

**Testes**

- `go build ./...` ok.
- `go test ./pkg/middlewares/ingressnginx/...` ok.
- `golangci-lint run ./pkg/middlewares/ingressnginx/...`: 0 issues.
