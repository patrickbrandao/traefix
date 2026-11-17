# Fix duplicate Vary header in compress middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12688](https://github.com/traefik/traefik/pull/12688) |
| Tipo | Correção |
| Área | middleware/compress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | veeceey |
| Aberto em | 2026-02-15 (última atualização: 2026-04-09) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +195/-3 em 4 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/M, area/middleware |
| Status | importado com ajustes |

## O que faz

Corrige o header Vary: Accept-Encoding duplicado quando o middleware compress está ativo e o backend também envia o header. Afeta gzip, brotli e zstd. Corrige a issue #11259.

## Riscos e situação no upstream

Pequeno a médio (4 arquivos, +195/-3) e com testes. Status design-review. Maintainers adiaram por falta de prioridade, e o autor pediu para rebasear.

## Principais arquivos

- `pkg/middlewares/compress/compress.go`
- `pkg/middlewares/compress/compression_handler.go`
- `pkg/middlewares/compress/compression_handler_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12688/head:upstream-pr-12688`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.** Reimplementado sobre o traefik/traefik#14235, que foi importado antes.

Achados no PR original:
- O bug procede. O middleware adicionava `Vary: Accept-Encoding` *antes* de chamar o handler (no `CompressionHandler` e no `gzhttp`). O reverse proxy copia os headers do backend com `Add`, então o `Vary: Accept-Encoding` enviado pelo backend aparecia duplicado.
- Para gzip, o PR embrulhava o writer num `deduplicateVaryWriter` que **viola as regras do AGENTS.md**:
  - não tem `Unwrap`, o que quebra o `http.ResponseController` (deadlines, full duplex) para tudo abaixo;
  - faz asserção de `http.Flusher`/`http.Hijacker`;
  - o `Flush` repassa sem flush quando o writer de baixo não é `Flusher`;
  - o `Hijack` só repassa.
- O `deduplicateVary` reescrevia todos os valores de `Vary`, separando listas em várias linhas e removendo duplicatas que o backend mandou de propósito. Isso vai além do escopo.
- Para brotli/zstd, o `ensureVaryHeader()` era chamado em vários pontos espalhados do `Write`.
- Upstream: em `status/1-needs-design-review`, sem objeção técnica.

Ajustes:
- Com o #14235, gzip, brotli e zstd passam pelo mesmo `CompressionHandler`, e **não é preciso nenhum wrapper novo**.
- O `Add` antecipado em `ServeHTTP` foi removido. O `Vary` passa a ser completado num único ponto, `responseWriter.writeHeader()`, usado por todos os envios de headers (`start` e `close`).
- `Accept-Encoding` só é acrescentado se ainda não estiver listado. A busca é por token, case-insensitive, aceita listas (`Cookie, accept-encoding`) e respeita `Vary: *`. Os valores do backend são mantidos como vieram.
- Teste `TestVaryHeader`, rodado para gzip, br e zstd: sem `Vary` do backend, com outros valores, resposta não comprimida, resposta já codificada, duplicata em resposta comprimida e não comprimida, lista com outros valores e `*`.

Testes:
- `go test -count=1 ./pkg/middlewares/compress/` ok.
- `golangci-lint run ./pkg/middlewares/compress/...`: 0 issues.
- `go build ./...` ok.
