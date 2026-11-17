# feat(tcp): add weight support for TCP server load balancer

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12359](https://github.com/traefik/traefik/pull/12359) |
| Tipo | Recurso |
| Área | tcp/loadbalancer |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | nathannewyen |
| Aberto em | 2025-12-03 (última atualização: 2026-03-12) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +12/-1 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/server |
| Status | pendente |

## O que faz

Adiciona o campo weight aos servers de services TCP, trazendo para o TCP o balanceamento ponderado que o HTTP já tem. Fecha a issue #12182 e inclui exemplo de configuração em file provider.

## Riscos e situação no upstream

Pequeno (5 arquivos, +12/-1). Precisa regenerar zz_generated.deepcopy.go. Ainda sem review de maintainer (só label de triagem).

## Principais arquivos

- `pkg/config/dynamic/tcp_config.go`
- `pkg/config/dynamic/zz_generated.deepcopy.go`
- `pkg/server/service/tcp/service.go`
- `docs/content/reference/routing-configuration/other-providers/file.yaml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12359/head:upstream-pr-12359`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
