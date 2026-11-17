# feat(accesslog): report upstream failure cause in access logs

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13442](https://github.com/traefik/traefik/pull/13442) |
| Tipo | Recurso |
| Área | logs/accesslog |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | hbollon |
| Aberto em | 2026-06-30 (última atualização: 2026-07-02) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +218/-1 em 6 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/logs, size/L, area/middleware |
| Status | pendente |

## O que faz

Adiciona o campo OriginError ao access log, com a causa do erro no upstream (connection refused, i/o timeout, erros TLS etc.), presente só quando o forward falha (502/504 e similares). Reusa o mecanismo LogData.Core, o mesmo de RetryAttempts. Dispensa o DEBUG para diagnosticar 502/504. Closes #12011.

## Riscos e situação no upstream

Tamanho L (+218 linhas), com mudança no ErrorHandlerWithContext compartilhado pelos dois proxies HTTP. Sem review ainda. Adiciona campo novo ao formato do access log, então parsers precisam ignorá-lo.

## Principais arquivos

- `pkg/middlewares/accesslog/logdata.go`
- `pkg/middlewares/accesslog/logger.go`
- `pkg/middlewares/accesslog/logger_test.go`
- `pkg/proxy/httputil/proxy.go`
- `pkg/proxy/httputil/proxy_test.go`
- `docs/content/reference/install-configuration/observability/logs-and-accesslogs.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13442/head:upstream-pr-13442`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
