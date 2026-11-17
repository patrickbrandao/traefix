# Do not compress partial content responses

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13856](https://github.com/traefik/traefik/pull/13856) |
| Tipo | Correção |
| Área | middleware/compress |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | datrixlab |
| Aberto em | 2026-09-07 (última atualização: 2026-09-10) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +65/-0 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/middleware |
| Status | adiado |

## O que faz

Impede que os handlers de compressão Brotli e Zstandard comprimam respostas com Content-Range (206 Partial Content). Sem isso, o corpo comprimido é enviado com o Content-Range do conteúdo original, e clientes que respeitam range leem bytes corrompidos.

## Riscos e situação no upstream

Correção localizada em compress_handler.go com testes. Status needs-review, sem review de maintainer ainda.

## Principais arquivos

- `pkg/middlewares/compress/compression_handler.go`
- `pkg/middlewares/compress/compression_handler_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13856/head:upstream-pr-13856`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: adiado.** Superado pelo traefik/traefik#14235, já importado.

Análise:
- O bug procede. Brotli e zstd comprimiam respostas `206` com `Content-Range`, o que deixava `Content-Range`/`Content-Length` descrevendo bytes que o cliente não recebe. O gzip (`gzhttp`) já pulava esses casos.
- O fix é correto e mínimo. A review automática do upstream não fez apontamentos.
- Nenhum risco de segurança.

Motivo: o #14235 move as três codificações para o mesmo `CompressionHandler`, e o `compressionAllowed()` dele já recusa respostas com `Content-Range`, com o mesmo efeito deste PR. O teste `TestResponseHeaders/partial content is not compressed` cobre o caso para gzip, br e zstd. Aplicar este PR por cima seria redundante e geraria conflito, porque altera a função `Write` antiga, que o #14235 reescreveu.

Testes: `go test ./pkg/middlewares/compress/` (já com o #14235) cobre o comportamento e passa.
