# Connect to Tailscale using tsnet

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11733](https://github.com/traefik/traefik/pull/11733) |
| Tipo | Recurso |
| Área | server/entrypoints/tailscale |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | ItalyPaleAle |
| Aberto em | 2025-05-05 (última atualização: 2026-01-28) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +342/-35 em 10 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/server |
| Status | pendente |

## O que faz

Permite configurar um entrypoint com tsnet, fazendo o Traefik entrar na Tailnet como dispositivo próprio, sem daemon Tailscale separado. Aceita authKey explícita ou via TS_AUTHKEY, e o autor testou HTTP/1, HTTP/2 e HTTP/3.

## Riscos e situação no upstream

Adiciona dependência tsnet (grande) e mexe no ciclo de vida de entrypoints (10 arquivos, +342/-35). Label status/2-needs-review, sem reviews dos maintainers. Precisa avaliar peso da dependência no binário.

## Principais arquivos

- `pkg/config/static/entrypoints.go`
- `pkg/server/server_entrypoint_tcp.go`
- `pkg/server/server_entrypoint_tcp_http3.go`
- `go.mod`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11733/head:upstream-pr-11733`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
