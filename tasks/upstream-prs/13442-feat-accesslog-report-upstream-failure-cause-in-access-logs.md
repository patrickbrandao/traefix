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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

### Achados

- Sem review de maintainer; mudança pequena no ponto único de erro do proxy (`ErrorHandlerWithContext`), reaproveitando o `LogData.Core`.
- Vazamento de dados: o PR gravava `err.Error()` cru. Um `*url.Error` inclui a URL completa da requisição, com query string, o que contornaria `accesslog.fields.queryParameters` (tokens em query iriam para o log).
- Log injection: algumas mensagens embutem valores do cliente (ex.: `client tried to switch to invalid protocol %q` com o header `Upgrade`). O campo vai para o JSON (escapado), mas também para o OTLP e outras saídas; sem limite de tamanho.
- Com retries, o campo guarda o erro da última tentativa que falhou mesmo se uma tentativa posterior der certo; isso não estava documentado.

### Ajustes

- `originErrorMessage`: desembrulha `*url.Error` (sem URL), troca caracteres de controle por espaço e limita a 512 bytes (UTF-8 válido).
- Conflitos de doc e teste resolvidos sobre a versão do Traefix; linha do `OriginError` na tabela de campos JSON, incluindo o comportamento com retries.
- Testes reescritos: URL não vaza, caracteres de controle, 499 sem campo, limite de tamanho, ausência de `LogData`.

### Testes

- `go vet ./...` (compilação de todos os pacotes): ok. O link do binário em `go build ./...` falhou por falta de espaço em disco na máquina (ENOSPC), não por erro de código.
- `go test ./pkg/proxy/... ./pkg/middlewares/accesslog/`: ok.
- `golangci-lint run ./pkg/proxy/httputil/... ./pkg/middlewares/accesslog/...`: 0 issues.
