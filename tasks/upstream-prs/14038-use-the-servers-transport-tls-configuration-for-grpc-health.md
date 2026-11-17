# Use the servers transport TLS configuration for gRPC health checks

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14038](https://github.com/traefik/traefik/pull/14038) |
| Tipo | Correção |
| Área | healthcheck/grpc |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | jayhemnani9910 |
| Aberto em | 2026-10-11 (última atualização: 2026-10-15) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +113/-12 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/healthcheck |
| Status | pendente |

## O que faz

O health check gRPC (mode: grpc) so usava texto puro e falhava em backends TLS, marcando o servidor como DOWN (503). Agora usa o scheme da URL do servidor e, para https, a configuracao TLS do serversTransport do servico.

## Riscos e situação no upstream

Pequeno (4 arquivos, +113/-12). Afeta apenas quem usa healthCheck gRPC com TLS. Tres reviews COMMENTED; label status/2-needs-review.

## Principais arquivos

- `pkg/healthcheck/healthcheck.go`
- `pkg/healthcheck/healthcheck_test.go`
- `pkg/server/service/service.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14038/head:upstream-pr-14038`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
