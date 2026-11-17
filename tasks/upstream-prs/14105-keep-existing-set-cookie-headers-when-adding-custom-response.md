# Keep existing Set-Cookie headers when adding custom response headers

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14105](https://github.com/traefik/traefik/pull/14105) |
| Tipo | Correção |
| Área | middleware/headers |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | lazerg |
| Aberto em | 2026-10-20 (última atualização: 2026-10-20) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +50/-3 em 3 arquivos |
| Labels | status/0-needs-triage, kind/bug/fix, size/S, area/middleware |
| Status | pendente |

## O que faz

O middleware headers, ao aplicar customResponseHeaders, usa Header.Set e apaga os Set-Cookie já existentes. Com sticky sessions e um headers middleware que define cookie, o cookie de afinidade se perde. A correção acrescenta Set-Cookie em vez de substituir; os demais headers continuam sendo sobrescritos.

## Riscos e situação no upstream

Mudança pequena e localizada, com testes e docs atualizados. Escopo restrito a Set-Cookie. Status: aguardando triagem dos maintainers (status/0-needs-triage). Base v3.7, cherry-pick deve ser simples.

## Principais arquivos

- `pkg/middlewares/headers/header.go`
- `pkg/middlewares/headers/header_test.go`
- `docs/content/reference/routing-configuration/http/middlewares/headers.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14105/head:upstream-pr-14105`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
