# Fix compress excludedContentTypes option

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12066](https://github.com/traefik/traefik/pull/12066) |
| Tipo | Correção |
| Área | middleware/compress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | DorianBlues |
| Aberto em | 2025-09-15 (última atualização: 2026-08-26) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +8/-30 em 7 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/S, area/middleware, breaking |
| Status | pendente |

## O que faz

O excludedContentTypes do compress verifica o Content-Type da requisição em vez do da resposta, o que gera resultado ambíguo. O PR passa a usar o Content-Type da resposta. Bug confirmado localmente: compress.go linha 140 ainda lê req.Header Content-Type.

## Riscos e situação no upstream

Base é v3.7, então o port para master é manual. Maintainer pediu issue dedicada porque o comportamento antigo pode ser usado por alguns usuários e a remoção precisa de discussão. Possível breaking change.

## Principais arquivos

- `pkg/middlewares/compress/compress.go`
- `pkg/middlewares/compress/compress_test.go`
- `pkg/config/dynamic/middlewares.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12066/head:upstream-pr-12066`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
