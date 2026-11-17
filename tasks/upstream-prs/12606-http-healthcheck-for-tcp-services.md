# HTTP Healthcheck for TCP services

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12606](https://github.com/traefik/traefik/pull/12606) |
| Tipo | Recurso |
| Área | healthcheck/tcp |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | supergoudvis116 |
| Aberto em | 2026-01-27 (última atualização: 2026-04-28) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +421/-25 em 6 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/healthcheck, area/tcp |
| Status | pendente |

## O que faz

Permite healthcheck HTTP em services TCP, para decidir o roteamento pelo resultado de um endpoint HTTP e não só por uma checagem TCP. Caso de uso: clusters Patroni, etcd ou PostgreSQL que roteiam só para o nó primário. Maintainers confirmaram que a funcionalidade está no roadmap.

## Riscos e situação no upstream

Novo caminho de healthcheck no TCP (6 arquivos, +421/-25). Maintainers confirmaram que está no roadmap, mas sem ETA e sem review ainda.

## Principais arquivos

- `pkg/healthcheck/healthcheck_tcp.go`
- `pkg/healthcheck/tcphealthcheckmode/tcphealthcheckmode.go`
- `pkg/config/dynamic/tcp_config.go`
- `pkg/server/service/tcp/service.go`
- `docs/content/reference/routing-configuration/tcp/service.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12606/head:upstream-pr-12606`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
