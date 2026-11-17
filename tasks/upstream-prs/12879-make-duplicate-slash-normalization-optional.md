# Make duplicate slash normalization optional

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12879](https://github.com/traefik/traefik/pull/12879) |
| Tipo | Recurso |
| Área | server/entrypoints/sanitizePath |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | boinger |
| Aberto em | 2026-03-26 (última atualização: 2026-08-10) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +447/-6 em 10 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/server |
| Status | importado com ajustes |

## O que faz

Adiciona a opção de entrypoint mergeSlashes (padrão true), que controla a junção de barras consecutivas em sanitizePath. Com false, // é preservado e segmentos . e .. continuam sendo resolvidos. Corrige a issue #12666.

## Riscos e situação no upstream

Mexe na sanitização de path, que é superfície de segurança (path traversal, autorização de gateways a montante). Precisa de review de segurança. Status review sem maintainer comentando.

## Principais arquivos

- `pkg/config/static/entrypoints.go`
- `pkg/server/server_entrypoint_tcp.go`
- `docs/content/reference/install-configuration/entrypoints.md`
- `docs/content/security/request-path.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12879/head:upstream-pr-12879`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, sem review de maintainer (apenas "vamos revisar").

### Achados no PR original

- **Regressão de segurança (path parameters):** o novo `sanitizeDotSegments` (usado com `mergeSlashes=false`) não removia os
  parâmetros dos segmentos `..;`, `.;` e `;`, que o `sanitizePath` atual trata. Com `mergeSlashes=false`, `/public/..;/admin`
  chegaria intacto a backends Servlet/Spring, que o resolvem como `/admin` (bypass de roteamento/autorização).
- **Divergência entre `Path` e `RawPath`:** o PR aplicava `removeDotSegments` separadamente a `Path` e `RawPath`. Com barra
  codificada (`/a%2Fb/../c`), `Path` virava `/a/c` e `RawPath` `/c`: o roteador e o backend passavam a ver caminhos diferentes.
- Âncora `#mergeslashes` citada na doc sem a seção correspondente.
- Risco inerente (opt-in): com `mergeSlashes=false`, `//admin` não casa com regras de `/admin`; se o backend fundir barras,
  há bypass de router/middlewares. O padrão continua `true`, sem mudança de comportamento.

### Ajustes

- `sanitizeDotSegments` passou a trabalhar só sobre o caminho escapado (como o `JoinPath` do caminho padrão), remove os
  parâmetros de dot-segments e deriva `Path` de `RawPath` (erro de decodificação gera 400).
- Lógica de remoção de parâmetros extraída para `removeDotSegmentParameters`, compartilhada pelos dois caminhos.
- `MergeSlashes` nulo é tratado como `true` (padrão seguro).
- Conflitos resolvidos preservando os campos locais (`MaxHeaderValueCount`, `AliasHeadersStrategy`); `configuration-options.md`
  regenerado com `go generate`.
- Doc: seção `mergeSlashes` com aviso de segurança e exemplos.
- Testes de `sanitizeDotSegments` reescritos: barra codificada, `..;`, `.;`, `;`, `RequestURI` com query.

### Testes

- `go build ./...` ok.
- `go test ./pkg/server/ ./pkg/config/static/` ok.
- `golangci-lint run ./pkg/server/... ./pkg/config/static/...`: 0 issues.
