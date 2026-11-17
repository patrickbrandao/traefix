# Add TLSKeyExchange access log field

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12714](https://github.com/traefik/traefik/pull/12714) |
| Tipo | Recurso |
| Área | accesslogs/tls |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | jacobsee |
| Aberto em | 2026-02-23 (última atualização: 2026-03-14) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +26/-2 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/logs, size/S, area/middleware, area/accesslogs |
| Status | pendente |

## O que faz

Adiciona o campo TLSKeyExchange aos access logs, com o grupo de troca de chaves negociado (por exemplo X25519MLKEM768). Permite auditar uso de troca de chaves pós-quântica, já que o TLSCipher não mostra isso no TLS 1.3. Também corrige um cipher de teste inválido.

## Riscos e situação no upstream

Pequeno (5 arquivos, +26/-2). Usa tls.ConnectionState.CurveID, que exige Go 1.25+. Status de review, sem maintainer comentando.

## Principais arquivos

- `pkg/middlewares/accesslog/logdata.go`
- `pkg/middlewares/accesslog/logger.go`
- `pkg/tls/key_exchange.go`
- `docs/content/observability/access-logs.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12714/head:upstream-pr-12714`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
