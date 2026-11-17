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
| Status | pendente |

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
