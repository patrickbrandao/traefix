# Fix ForwardAuth rewriting backend Set-Cookie headers

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14277](https://github.com/traefik/traefik/pull/14277) |
| Tipo | Correção |
| Área | middleware/forwardauth |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | ump45nose |
| Aberto em | 2026-11-12 (última atualização: 2026-11-12) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +82/-5 em 2 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/M, area/authentication |
| Status | pendente |

## O que faz

O ForwardAuth parseava e reserializava todos os Set-Cookie do backend quando o auth server devolvia cookies, perdendo valores e atributos. Agora os cookies do backend passam intactos e só os listados em addAuthCookiesToResponse são substituídos.

## Riscos e situação no upstream

Base é v3.7, precisa de port. Mudança pequena em forward.go com testes. Status status/1-needs-design-review, então um maintainer pediu revisão de design antes de avançar. Afeta qualquer ForwardAuth que devolve cookies.

## Principais arquivos

- `pkg/middlewares/auth/forward.go`
- `pkg/middlewares/auth/forward_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14277/head:upstream-pr-14277`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
