# Add support multiple tls certificate stores

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11380](https://github.com/traefik/traefik/pull/11380) |
| Tipo | Recurso |
| Área | tls |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | jidckii |
| Aberto em | 2024-12-31 (última atualização: 2026-07-02) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +638/-22 em 9 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/M, area/tls |
| Status | pendente |

## O que faz

Permite separar certificados TLS em múltiplos stores, para que um cliente não sobrescreva ou invalide certificados de outro com o mesmo CN. O corpo do PR motiva pelo cenário de nós onde clientes sobem seus próprios certificados, mas não detalha a API final.

## Riscos e situação no upstream

Toca TLS manager, config dinâmica e router TCP (9 arquivos, +638/-22). Label status/1-needs-design-review; um comentário de maintainer diz que a funcionalidade não deve entrar no upstream e a resposta seguinte é só 'sem ETA'. Sem reviews formais.

## Principais arquivos

- `pkg/config/dynamic/http_config.go`
- `pkg/config/dynamic/tcp_config.go`
- `pkg/provider/acme/provider.go`
- `pkg/server/router/router.go`
- `pkg/server/router/tcp/manager.go`
- `pkg/server/router/tcp/router.go`
- `docs/content/https/tls.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11380/head:upstream-pr-11380`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
