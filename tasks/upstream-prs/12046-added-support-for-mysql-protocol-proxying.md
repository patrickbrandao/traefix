# Added support for MySQL protocol proxying

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12046](https://github.com/traefik/traefik/pull/12046) |
| Tipo | Recurso |
| Área | tcp/mysql |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | Artorios |
| Aberto em | 2025-09-08 (última atualização: 2026-11-03) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +273/-15 em 5 arquivos |
| Labels | status/0-needs-triage, size/L |
| Status | pendente |

## O que faz

Adiciona roteamento TCP por SNI para conexões MySQL com passthrough TLS, enviando um handshake MySQL falso para forçar negociação SSL e extraindo o SNI do ClientHello. Resolve a issue 10505.

## Riscos e situação no upstream

Protocolo novo com handshake simulado, interage com o router TCP (5 arquivos, +273/-15). Status needs-triage, sem review; autores e usuários pedem andamento sem ETA. Corpo do PR é longo e sem testes marcados.

## Principais arquivos

- `pkg/server/router/tcp/mysql.go`
- `pkg/server/router/tcp/router.go`
- `pkg/server/routerfactory.go`
- `pkg/server/server_entrypoint_tcp.go`
- `pkg/config/static/entrypoints.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12046/head:upstream-pr-12046`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
