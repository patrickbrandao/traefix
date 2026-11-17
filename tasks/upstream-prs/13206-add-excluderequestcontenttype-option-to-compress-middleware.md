# Add ExcludeRequestContentType option to compress middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13206](https://github.com/traefik/traefik/pull/13206) |
| Tipo | Recurso |
| Área | middlewares/compress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | KaanSimsek |
| Aberto em | 2026-05-19 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +106/-42 em 8 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/middleware |
| Status | pendente |

## O que faz

Adiciona a opção excludeRequestContentType ao middleware compress. Com ela desligada, o Content-Type da requisição deixa de decidir se a resposta será comprimida, e só o Content-Type da resposta é avaliado. Fecha o issue #12070, por exemplo para text/event-stream.

## Riscos e situação no upstream

Comentário positivo de maintainer, sem aprovação formal. Toca o redactor (testdata) e o label, então precisa checar o valor padrão para não mudar o comportamento atual.

## Principais arquivos

- `pkg/middlewares/compress/compress.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/middlewares/compress/compress_test.go`
- `docs/content/reference/routing-configuration/http/middlewares/compress.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13206/head:upstream-pr-13206`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
