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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

### Achados

- Sem review de maintainer; feature opt-in (`accessLog.anonymization`) para GDPR. `ip.MaskIP` em si está correto (`netip`, normaliza 4-in-6, valida faixas na inicialização).
- Falha de privacidade: o mascaramento era aplicado ANTES do override de `ClientHost` por `X-Forwarded-For`. Atrás de um load balancer (o caso comum), o `ClientHost` saía com o IP completo do cliente; a doc do PR até admitia que o XFF "é logado inteiro".
- Os headers que carregam o IP do cliente (`X-Forwarded-For`, `X-Real-Ip`, `Forwarded`, este último gerado pelo Traefix desde o commit 193af416c) continuavam com o IP completo quando `fields.headers` mantém esses headers.
- Armadilha de configuração: com só `ipv4Subnet` definido, o IPv6 ficava em 0 (sem truncamento) e era logado inteiro sem aviso.
- Conflito com a assinatura atual do `NewHandler` no Traefix (parâmetro `hooks`).

### Ajustes

- O mascaramento roda depois do override por XFF e trata a lista inteira (cada endereço, com ou sem porta).
- Headers de request mantidos no log com IPs (`X-Forwarded-For`, `X-Real-Ip`, `X-Original-Forwarded-For`, `True-Client-Ip`, `Cf-Connecting-Ip`, `Forwarded` com `for=`/`by=` do RFC 7239) também são truncados.
- `AccessLogAnonymization.SetDefaults` com 24/48: habilitar a seção nunca deixa uma família sem truncamento (0 continua desligando explicitamente).
- Helpers movidos para `pkg/middlewares/accesslog/anonymize.go`; conflito do `NewHandler` resolvido mantendo `hooks`.
- Doc atualizada (escopo real, defaults, e que logs do Traefik, traces e métricas não são afetados); `go generate`.
- Testes novos: `TestMaskIPList`, `TestMaskForwarded`, `TestAnonymization_xForwardedFor` (log JSON completo com XFF e headers).

### Testes

- `go vet ./pkg/... ./cmd/...`: ok (link do binário limitado por falta de espaço em disco na máquina).
- `go test ./pkg/middlewares/accesslog/ ./pkg/ip/`: ok (`TestLogger_AbortedRequest` é instável por timing; falhou uma vez e passou em 5 reexecuções, sem relação com a mudança).
- `golangci-lint run ./pkg/middlewares/accesslog/... ./pkg/ip/... ./pkg/observability/types/...`: 0 issues.
