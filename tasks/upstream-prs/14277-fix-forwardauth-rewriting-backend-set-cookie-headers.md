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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

Veredito: **importado com ajustes**, combinado com o #12229.

Achados:
- PR aberto, base `v3.7`, com status `needs-design-review` e sem review humano. A correção é pequena e não muda a decisão de autenticação: só deixa de reserializar os `Set-Cookie` do backend.
- **Sobreposição com o #12229** (importado antes neste grupo): o #12229 já passou o modifier a trabalhar com as linhas `Set-Cookie` brutas, tanto do backend quanto do auth server. A parte central deste PR já estava incorporada, e de forma mais robusta: o #12229 extrai o nome cortando primeiro em `;` e depois em `=`, enquanto este PR cortava só em `=`, o que errava o nome de uma linha sem `=` como `flag; Path=/`.
- O que faltava daqui era pular o modifier quando `addAuthCookiesToResponse` está vazio. Sem isso, qualquer cookie do auth server fazia o writer do backend ser embrulhado à toa.

Ajustes:
- O modifier só é instalado quando há algo a substituir: regex configurado (que remove os cookies do backend que casam, mesmo sem cookie de auth, como decidido no #12229) ou lista não vazia com cookies na resposta de auth. Comentário de porquê adicionado.
- Os testes do PR (`TestForwardAuthSuccessBackendCookiesUnchanged`, `TestForwardAuthSuccessAddAuthCookiesKeepsRawBackendCookies`) foram importados e adaptados à assinatura nova de `NewForward`.
- A parte do PR em `buildModifier` não foi aplicada, porque a versão do #12229 a substitui.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/middlewares/auth/`: ok.
- `golangci-lint run ./pkg/middlewares/auth/...`: 0 issues.
