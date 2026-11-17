# feat: add UDP Proxy Protocol support

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12224](https://github.com/traefik/traefik/pull/12224) |
| Tipo | Recurso |
| Área | udp/proxyprotocol |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | sdelicata |
| Aberto em | 2025-11-03 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +1031/-54 em 12 arquivos |
| Labels | kind/enhancement, status/2-needs-review, contributor/waiting-for-corrections, size/L, area/udp |
| Status | pendente |

## O que faz

Adiciona suporte a PROXY protocol v2 para entrypoints UDP, preservando IP e porta do cliente. O header é lido do primeiro pacote, e a validação usa TrustedIPs.

## Riscos e situação no upstream

Grande (12 arquivos, +1031/-54) e com review CHANGES_REQUESTED. Reviewer questionou como os pacotes seguintes são ligados à sessão, já que o header só aparece no primeiro. Precisa de desenho sólido antes do port.

## Principais arquivos

- `pkg/udp/proxy.go`
- `pkg/udp/conn.go`
- `pkg/server/server_entrypoint_udp.go`
- `pkg/config/static/entrypoints.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12224/head:upstream-pr-12224`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
