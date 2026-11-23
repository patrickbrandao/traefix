# Tasks de sincronização com o upstream (traefik/traefik)

Geradas pelo agente `upstream-tracker` (`.claude/agents/upstream-tracker.md`).
Ponto de partida: tag `fork-point` (upstream `master` em 36b2bd1, base v3.8.0-rc.3).

| Nº | Título | Tipo | Prioridade | Status | PR upstream |
|----|--------|------|------------|--------|-------------|
| [0001](0001-keep-ocsp-staple-cache-janitor-alive.md) | Keep the OCSP staple cache janitor alive | bugfix | média | aplicada `35c665896` | [#14257](https://github.com/traefik/traefik/pull/14257) |
| [0002](0002-apply-upstream-vhost-after-auth-middleware.md) | Apply the upstream vhost after the auth middleware | bugfix | baixa | aplicada `6fbc69724` | [#13787](https://github.com/traefik/traefik/pull/13787) |
| [0003](0003-split-yaml-test-fixtures-on-crlf.md) | Split YAML test fixtures on CRLF document separators | bugfix | baixa | aplicada `2b558060d` | [#13866](https://github.com/traefik/traefik/pull/13866) |
| [0004](0004-detect-tls-options-conflicts-trailing-dot.md) | Detect TLS options conflicts on hosts with a trailing dot | bugfix | média | aplicada `e3f72b006` | [#14192](https://github.com/traefik/traefik/pull/14192) |
| [0005](0005-document-accesslog-entrypointname-traceid-spanid.md) | Document entryPointName, trace_id and span_id JSON access log fields | bugfix | baixa | aplicada `ceb036373` | [#14302](https://github.com/traefik/traefik/pull/14302) |
| [0006](0006-remove-duplicate-consul-catalog-address-check.md) | Remove duplicate Consul Catalog address check | refactor | baixa | aplicada `1770b4b70` | [#14298](https://github.com/traefik/traefik/pull/14298) |
| [0007](0007-redact-static-plugin-provider-and-provider-fields.md) | Redact the static plugin provider configuration + Redact static provider fields | segurança | média | aplicada `7cb758e39` | [#14304](https://github.com/traefik/traefik/pull/14304), [#14305](https://github.com/traefik/traefik/pull/14305) |
| [0008](0008-bump-otel-sdk-log-v1-47-0.md) | Bump go.opentelemetry.io/otel/sdk/log to v1.47.0 | deps | baixa | aplicada `956481d4f` | [#14315](https://github.com/traefik/traefik/pull/14315) |
| [0009](0009-log-gateway-api-informer-list-and-watch-errors.md) | Log Gateway API informer list and watch errors | bugfix | média | aplicada `06fd84adf` | [#14313](https://github.com/traefik/traefik/pull/14313) |
| [0010](0010-expose-configuration-building-kubernetes-providers.md) | Expose the configuration building of the Kubernetes providers | refactor | média | aplicada `8618f77ba` | [#14286](https://github.com/traefik/traefik/pull/14286) |
| [0011](0011-support-location-modifier-ingress-nginx-server-snippet.md) | Support the ^~ location modifier in ingress-nginx server-snippet | bugfix | baixa | aplicada `f738ad60e` | [#13904](https://github.com/traefik/traefik/pull/13904) |
| [0012](0012-support-gateway-api-listener-filters.md) | Support Gateway API listener filters | feature | baixa | aplicada `48bb2caec` | [#14320](https://github.com/traefik/traefik/pull/14320) |
| [0013](0013-update-yaegi-v2-filemapping-mounts.md) | Update Yaegi v2 and add fileMapping and mounts settings | feature | média | aplicada `97e317578` | [#14303](https://github.com/traefik/traefik/pull/14303) |

Ordem de aplicação: a numeração segue a ordem do upstream. A 0010 depende da 0009, e a 0012 depende da 0009 e da 0010.
Na 0007, não faça cherry-pick dos commits do branch v2.11: eles trazem imports `traefik/v2` (veja a task).

Aplicadas em 2026-11-23, uma por commit, na ordem acima. O commit de documentação `4ded58e2f` (#14317, multi-tenancy e Traefik Operator) entrou também, sem task. A 0007 não é cherry-pick (adota `redactor.go` e `redactor_config_test.go` do oficial e acrescenta o teste dos segredos do `cluster`); a 0009 leva o ajuste de `WatchAll` nos testes importados; 0002, 0010 e 0011 carregam no próprio commit a resolução de conflito descrita na task.
