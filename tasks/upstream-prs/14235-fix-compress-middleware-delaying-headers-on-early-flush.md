# Fix compress middleware delaying headers on early flush

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14235](https://github.com/traefik/traefik/pull/14235) |
| Tipo | Correção |
| Área | middleware/compress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | kunalvirwal |
| Aberto em | 2026-11-06 (última atualização: 2026-11-09) |
| Draft | não |
| Branch-alvo | `v3.8` |
| Tamanho | +864/-262 em 5 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/L, area/middleware |
| Status | importado |

## O que faz

Corrige o middleware de compressão que segurava os headers quando o backend fazia flush antes de escrever o corpo, o que trava watches do kubectl sem eventos. Unifica gzip, brotli e zstd no mesmo CompressionHandler, trocando o gzhttp do klauspost para gzip.

## Riscos e situação no upstream

O bot do repositório fechou este PR como duplicata de #14229 (aberto); o port deve seguir #14229. Muda o comportamento do gzip: Accept-Encoding: * passa a comprimir e há outras mudanças de comportamento documentadas no PR. Mexe em 5 arquivos, incluindo a doc de compress. Review: COMMENTED.

## Principais arquivos

- `pkg/middlewares/compress/compress.go`
- `pkg/middlewares/compress/compression_handler.go`
- `pkg/middlewares/compress/compress_test.go`
- `pkg/middlewares/compress/compression_handler_test.go`
- `docs/content/reference/routing-configuration/http/middlewares/compress.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14235/head:upstream-pr-14235`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado.** Este PR foi importado fora da ordem da lista, antes do #12688, porque é a base dos outros PRs de compress do grupo.

Situação no upstream: o bot de triagem fechou o PR como duplicado do #14229 e depois o reabriu em "review". A review automática não fez apontamentos. O branch-alvo é `v3.8`, o mesmo do Traefix, e o patch aplicou sem conflito.

Análise:
- Gzip passa a usar o mesmo `CompressionHandler` de brotli e zstd, com `klauspost/compress/gzip`. O `gzhttp` sai do código. O módulo `klauspost/compress` continua (gzip e zstd), sem mudança em `go.mod`/`go.sum`.
- O writer é reaproveitado pelo `sync.Pool` que já existia. Não vi regressão de memória: o zstd mantém as opções de baixa memória.
- A máquina de estados (`pending` → `buffering` → `plain`/`compressing`) é mais simples que os booleanos anteriores. Conferi:
  - a recursão de `Write`, que termina porque o estado sai de `buffering` antes de chamar de novo;
  - a escrita curta (`io.ErrShortWrite`);
  - o `close` em cada estado;
  - a ordem dos `defer` (o `close` roda antes do `putCompressionWriter`).
- Segue o AGENTS.md: `Unwrap`, `FlushError` e `Hijack` via `http.ResponseController`, sem asserção de `http.Flusher`. O `http.CloseNotifier` deprecado deixa de existir no caminho gzip.
- Corrige o bug do flush sem corpo (#14225, watch do Kubernetes travado): os headers saem no primeiro flush.
- Corrige o caso de respostas `206`/`Content-Range` comprimidas por brotli/zstd (o mesmo bug do #13856).
- Remove `Accept-Ranges` ao comprimir, faz sniffing de `Content-Type` também para brotli/zstd e nunca comprime `204`/`304`.
- Mudanças de comportamento estão documentadas no PR. A mais visível: gzip agora respeita `defaultEncoding` e `Accept-Encoding: *` como os outros algoritmos. É uma mudança coerente.
- Segurança: nenhuma chamada de rede, nenhuma dependência nova, buffer limitado a `max(minResponseBodyBytes, 512)`. Não encontrei problemas.

Sobreposições:
- **Supera o #13856** (partial content) e o **#14229** (flush antes de escrever). O #14229 corrige só o handler de brotli/zstd e deixa o gzip (`gzhttp`) com o mesmo bug. As duas tasks foram marcadas de acordo.
- O #12688 (`Vary` duplicado) e o #13206 (`excludeRequestContentType`) foram aplicados por cima deste.

Testes:
- `go test -count=1 ./pkg/middlewares/compress/` ok.
- `golangci-lint run ./pkg/middlewares/compress/...`: 0 issues.
- `go build ./...` ok.
