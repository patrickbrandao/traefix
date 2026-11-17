# Keep the brackets of an IPv6 host with a port in RedirectScheme

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14039](https://github.com/traefik/traefik/pull/14039) |
| Tipo | Correção |
| Área | middleware/redirectScheme |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | jayhemnani9910 |
| Aberto em | 2026-10-11 (última atualização: 2026-10-17) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +51/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | pendente |

## O que faz

RedirectScheme (e entrypoint http.redirections) respondia 502 quando o Host era um literal IPv6 com porta, ex. [2001:db8::1]:8080, porque os colchetes eram perdidos no SplitHostPort. Restaura os colchetes para IPv6.

## Riscos e situação no upstream

Pequeno (2 arquivos, +51). Caso de nicho (acesso por IP literal). Dois reviews COMMENTED.

## Principais arquivos

- `pkg/middlewares/redirect/redirect_scheme.go`
- `pkg/middlewares/redirect/redirect_scheme_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14039/head:upstream-pr-14039`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
