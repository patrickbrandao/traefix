# Fix ingress-nginx snippet location matching

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13770](https://github.com/traefik/traefik/pull/13770) |
| Tipo | Correção |
| Área | middleware/snippet, provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | parrahex |
| Aberto em | 2026-08-25 (última atualização: 2026-08-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +115/-7 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/middleware, area/provider/k8s/ingress-nginx |
| Status | importado com ajustes |

## O que faz

Dois ajustes para igualar ao ingress-nginx em snippets de location: operandos entre aspas deixam de incluir as aspas no padrão, e o match usa o path original da requisição e não o path já reescrito por rewrite-target. Corrige o #13690.

## Riscos e situação no upstream

Base v3.7, precisa de adaptação. Mudança de comportamento no middleware de rewrite e de snippet, o que pode alterar configs existentes que dependiam do path reescrito. Status/2-needs-review. Autor validou os testes contra o base sem o patch.

## Principais arquivos

- `pkg/middlewares/ingressnginx/request.go`
- `pkg/middlewares/ingressnginx/rewritetarget/rewrite_target.go`
- `pkg/middlewares/ingressnginx/snippet/collector.go`
- `pkg/middlewares/ingressnginx/snippet/snippet_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13770/head:upstream-pr-13770`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- **Segurança, bypass de regras de acesso:** os blocos `location` de `server-snippet` eram comparados com o caminho **já reescrito** pelo `rewrite-target`. Um bloqueio como `location ~ ^/admin { deny all; }`, numa rota com `rewrite-target: /$1` que remove o prefixo, deixava de casar e o acesso passava. No ingress-nginx a escolha da location acontece antes do rewrite da location da Ingress, então o bloqueio vale.
- **Segurança, regra ignorada em silêncio:** padrões entre aspas (`location ~ "^/x"`, `location = "/x"`), que são NGINX válido e comuns em regex, mantinham as aspas e nunca casavam. O bloco inteiro, incluindo `deny`/`return 403`, era ignorado sem aviso.
- O PR corrige os dois problemas guardando o caminho original no contexto e removendo as aspas do operando.
- Sobreposição com #13754: os dois PRs criam helpers concorrentes no pacote `ingressnginx` (`original_uri.go` e `request.go`) e mexem no mesmo ponto de `rewrite_target.go`.

**Ajustes**

- Juntei os helpers no `original_uri.go` de #13754. `WithOriginalPath` e `OriginalPath` seguem a mesma API de `*http.Request` de `WithOriginalURI`, e o primeiro valor guardado prevalece.
- O caminho original é guardado só pelo `rewrite-target`, não pelo snippet: no NGINX, um `rewrite` do próprio server-snippet acontece antes da escolha da location e deve afetá-la.
- Teste novo de regressão de segurança: `location = "/admin/secret" { deny all; }` com `rewrite-target` que remove `/admin` continua respondendo 403. Os testes de padrões entre aspas e o de location antes do rewrite, do PR, foram mantidos.
- Aspas simples dependem do `trimQuote` de #14044, importado em seguida.

**Testes**

- `go build ./...` ok.
- `go test ./pkg/middlewares/ingressnginx/...` ok.
- `golangci-lint run ./pkg/middlewares/ingressnginx/...`: 0 issues.
