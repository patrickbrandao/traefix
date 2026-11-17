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
| Status | pendente |

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
