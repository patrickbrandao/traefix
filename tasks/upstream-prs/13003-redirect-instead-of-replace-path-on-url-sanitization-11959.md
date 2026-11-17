# Redirect instead of Replace Path on URL Sanitization (11959)

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13003](https://github.com/traefik/traefik/pull/13003) |
| Tipo | Recurso |
| Área | entrypoints/sanitizePath |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | unihost-com |
| Aberto em | 2026-04-18 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +188/-3 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, contributor/waiting-for-corrections, size/M, area/server |
| Status | pendente |

## O que faz

Adiciona uma opção de entrypoint para responder com redirect 301/308 quando o path é normalizado (ex.: /../), em vez de reescrever a URL silenciosamente e responder 200. O caso de uso é SEO e respostas corretas a requisições malformadas.

## Riscos e situação no upstream

Foi redirecionado de v2.11 para master com conflitos; autor relata falha local em TestRestSuite/TestSimpleConfiguration (não confirmada) e label waiting-for-corrections. Precisa de correções antes de revisão.

## Principais arquivos

- `pkg/config/static/entrypoints.go`
- `pkg/server/server_entrypoint_tcp.go`
- `pkg/server/server_entrypoint_tcp_test.go`
- `docs/content/reference/install-configuration/entrypoints.md`
- `docs/content/security/request-path.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13003/head:upstream-pr-13003`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
