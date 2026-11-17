# Send response headers on flush before any write in compress middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14229](https://github.com/traefik/traefik/pull/14229) |
| Tipo | Correção |
| Área | middleware/compress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | thelastbackspace |
| Aberto em | 2026-11-04 (última atualização: 2026-11-05) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +165/-51 em 2 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/M, area/middleware |
| Status | adiado |

## O que faz

No middleware compress, um Flush antes de qualquer escrita não mandava os headers ao cliente. Kubernetes watches sem eventos ficavam travados esperando headers. Agora o Flush envia headers, sem compressão quando o conteúdo não é elegível, ou inicia o stream comprimido.

## Riscos e situação no upstream

Muda decisão de compressão em FlushError; teste antigo (Test_FlushAfterWriteNil) foi alterado. Status: needs-design-review. Escopo em dois arquivos.

## Principais arquivos

- `pkg/middlewares/compress/compression_handler.go`
- `pkg/middlewares/compress/compression_handler_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14229/head:upstream-pr-14229`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: adiado.** Superado pelo traefik/traefik#14235, já importado.

Análise:
- O PR corrige o flush antes de qualquer escrita (#14225, watch do Kubernetes travado esperando headers), mas só no `CompressionHandler` de brotli/zstd. O gzip, servido pelo `klauspost/compress/gzhttp`, continua com o mesmo bug, como o próprio #14235 aponta (o problema não foi corrigido no `gzhttp`).
- O código é bom. Ele extrai `compressionAllowed()`, e a ideia foi reaproveitada pelo #14235.
- Branch-alvo `v3.7`. O bot do upstream fechou o #14235 como duplicado deste e depois o reabriu.
- Nenhum risco de segurança.

Motivo: o #14235 resolve o mesmo bug para as três codificações, e com uma máquina de estados mais simples. Os testes dele incluem `TestFlushBeforeWriteSendsHeaders`, com cliente HTTP real e timeout, equivalente ao `Test_FlushBeforeWriteSendsHeaders` deste PR. Aplicar este por cima conflitaria com a reescrita do `responseWriter`.

Testes: `go test ./pkg/middlewares/compress/` (já com o #14235) cobre o comportamento e passa.
