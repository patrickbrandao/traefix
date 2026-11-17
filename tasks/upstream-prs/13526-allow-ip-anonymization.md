# Allow IP anonymization

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13526](https://github.com/traefik/traefik/pull/13526) |
| Tipo | Recurso |
| Área | accesslogs (privacidade/LGPD-GDPR) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | mpldr |
| Aberto em | 2026-07-15 (última atualização: 2026-08-01) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +210/-0 em 8 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/accesslogs |
| Status | pendente |

## O que faz

Adiciona opção para zerar bits dos IPs nos access logs, com o número de bits configurável, para que operadores possam reduzir IP (PII) para sub-rede, sem exigir retenção curta. A documentação traz exemplos de sub-redes /24 e /48. Atende a implementação da issue #13517.

## Riscos e situação no upstream

Adiciona opção nova de configuração de access log. Status needs-review, sem review de maintainer ainda. Afeta só a saída de log, não o roteamento. Feature em master, então exige cherry-pick ou merge com ajuste de base.

## Principais arquivos

- `pkg/ip/strategy.go`
- `pkg/ip/strategy_test.go`
- `pkg/middlewares/accesslog/logger.go`
- `pkg/middlewares/accesslog/anonymize_test.go`
- `pkg/observability/types/logs.go`
- `docs/content/reference/install-configuration/observability/logs-and-accesslogs.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13526/head:upstream-pr-13526`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
