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
| Status | pendente |

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
