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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

PR aberto, em "design-review" com maintainer, sem conclusão.

### Achados no PR original

- **Redação incompleta:** só `/api/http/middlewares[/{id}]` e `/api/tcp/middlewares[/{id}]` eram redigidos. O
  `/api/rawdata`, que o dashboard também consome, continuava devolvendo todos os middlewares com segredos em claro
  (configuração de plugins, usuários do Basic Auth...).
- **Padrão inseguro quando os defaults não são aplicados:** a opção era um `bool` com default `true` vindo de `SetDefaults`.
  Uma configuração `API` montada sem defaults (zero value) expunha os segredos.
- O nome na tabela da doc estava em minúsculas (`api.hidemiddlewareparameters`), divergindo da convenção camelCase.
- Ponto positivo: usa o `redactor.RemoveCredentials` existente (tags `loggable:"false"`; valores de plugin trocados por `{}`).
  Não adiciona dependência nem endpoint novo e não altera a autenticação da API.

### Ajustes

- `/api/rawdata` passou a ser redigido também.
- `HideMiddlewareParameters` virou `*bool`: com valor ausente (ou `API` nil), a resposta é redigida. Só `false` explícito
  expõe os valores.
- Helper renomeado para `writeRedactableJSON` (agora usado também fora dos middlewares).
- Expectativas dos testes existentes (`TestHandler_GetMiddleware`, `getrawdata.json`) atualizadas para o padrão redigido.
- Doc corrigida (camelCase, menção ao `/api/rawdata`); `configuration-options.md` regenerado.

### Testes

- `go test ./pkg/api/ ./pkg/config/static/ ./pkg/redactor/` ok.
- `golangci-lint run ./pkg/api/... ./pkg/config/static/...`: 0 issues; `go vet ./cmd/traefik/` ok.
