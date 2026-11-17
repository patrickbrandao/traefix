# fix: redact secrets in middlewares

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13441](https://github.com/traefik/traefik/pull/13441) |
| Tipo | Correção |
| Área | webui/api/security |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | michael-todorovic |
| Aberto em | 2026-06-30 (última atualização: 2026-07-02) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +159/-33 em 9 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, area/webui, size/M |
| Status | pendente |

## O que faz

Redige segredos de middlewares expostos pela API e pelo dashboard, para que o dashboard possa voltar a ser habilitado sem vazar credenciais. Fixes #12494.

## Riscos e situação no upstream

Status 1-needs-design-review: maintainers querem checar o impacto antes de avançar. Sem review. Exige decidir quais campos são segredos, e mexe em handlers HTTP e TCP da API e em static_config.go.

## Principais arquivos

- `pkg/api/handler_http.go`
- `pkg/api/handler_tcp.go`
- `pkg/api/handler_http_test.go`
- `pkg/api/handler_tcp_test.go`
- `pkg/config/static/static_config.go`
- `docs/content/reference/install-configuration/api-dashboard.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13441/head:upstream-pr-13441`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
