# feat: add global TCP access logging

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11998](https://github.com/traefik/traefik/pull/11998) |
| Tipo | Recurso |
| Área | middleware/accesslogs, entrypoints |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | Kronusus |
| Aberto em | 2025-08-13 (última atualização: 2026-04-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +294/-19 em 9 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/middleware, area/accesslogs |
| Status | pendente |

## O que faz

Adiciona access log de conexões TCP, registrando início e fim com endereços, bytes, duração, certificado de cliente e erro, habilitado pelo accessLog global. Suporta formato JSON e texto.

## Riscos e situação no upstream

Mexe em logs e entrypoints (9 arquivos, +294/-19). Autor marcou testes não feitos e descreve como 'early draft'. Label needs-design-review, sem reviews.

## Principais arquivos

- `cmd/traefik/tcp_accesslog.go`
- `pkg/middlewares/accesslogtcp/logger.go`
- `pkg/tcp/accesslog.go`
- `pkg/server/server_entrypoint_tcp.go`
- `pkg/server/routerfactory.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11998/head:upstream-pr-11998`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
