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
| Status | pendente |

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
