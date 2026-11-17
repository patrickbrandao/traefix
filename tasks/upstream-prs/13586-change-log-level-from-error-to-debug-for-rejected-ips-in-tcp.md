# Change log level from ERROR to DEBUG for rejected IPs in TCP IPAllowList

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13586](https://github.com/traefik/traefik/pull/13586) |
| Tipo | Correção |
| Área | middleware/tcp/ipallowlist (logs) |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | craftydomops |
| Aberto em | 2026-07-29 (última atualização: 2026-10-22) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +11/-3 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware, breaking |
| Status | pendente |

## O que faz

Rejeição de IP no IPAllowList TCP era logada como ERROR, o que gera ruído quando health checks batem no entrypoint TCP (2 logs por segundo). Passa para DEBUG, alinhando com o middleware HTTP equivalente. Renomeia o teste para TestIPAllowLister_ServeTCP.

## Riscos e situação no upstream

Marcado como breaking. Maintainers aprovaram a ideia, mas pediram para aguardar a próxima major, porque quem monitora o ERROR perde o sinal. Também toca docs/migrate/v3.md.

## Principais arquivos

- `pkg/middlewares/tcp/ipallowlist/ip_allowlist.go`
- `pkg/middlewares/tcp/ipallowlist/ip_allowlist_test.go`
- `pkg/middlewares/tcp/ipwhitelist/ip_whitelist.go`
- `docs/content/migrate/v3.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13586/head:upstream-pr-13586`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
