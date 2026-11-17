# Flush the OTLP access logs on shutdown

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13993](https://github.com/traefik/traefik/pull/13993) |
| Tipo | Correção |
| Área | accesslogs/otel |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | mloiseleur |
| Aberto em | 2026-09-30 (última atualização: 2026-10-01) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +95/-13 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/accesslogs, area/otel |
| Status | pendente |

## O que faz

Faz o shutdown do logger provider OTLP dos access logs no Handler.Close, para exportar os registros ainda na fila do batch processor. Sem isso, cada parada perde os logs do ultimo intervalo de exportacao (1s).

## Riscos e situação no upstream

Pequeno (2 arquivos, +95/-13). Relacionado ao #13994 (mesmo bug no provider de logs do Traefik). Um review COMMENTED de maintainer. Autor observa que o teste so passa com o fix se aguardar o Close.

## Principais arquivos

- `pkg/middlewares/accesslog/logger.go`
- `pkg/middlewares/accesslog/logger_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13993/head:upstream-pr-13993`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
