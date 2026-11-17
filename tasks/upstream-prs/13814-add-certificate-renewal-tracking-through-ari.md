# Add certificate renewal tracking through ARI

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13814](https://github.com/traefik/traefik/pull/13814) |
| Tipo | Recurso |
| Área | acme, tls |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | amarkdotdev |
| Aberto em | 2026-08-31 (última atualização: 2026-11-06) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +246/-10 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/acme, size/L |
| Status | pendente |

## O que faz

Renova certificados ACME quando o servidor sugere via ACME Renewal Information (RFC 9773). Útil para revogações em massa ou mudanças de CA. A renovação por tempo continua valendo; o ARI só antecipa. Honra Retry-After quando for menor que o intervalo.

## Riscos e situação no upstream

Depende da lib lego com o suporte de Renew/UseARICertID (go-acme/lego#3273), então pode exigir bump de dependência. Servidores sem ARI mantêm o comportamento atual. Status/2-needs-review, review não verificado. Fecha #12461 e #13808.

## Principais arquivos

- `pkg/provider/acme/provider.go`
- `pkg/provider/acme/provider_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13814/head:upstream-pr-13814`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
