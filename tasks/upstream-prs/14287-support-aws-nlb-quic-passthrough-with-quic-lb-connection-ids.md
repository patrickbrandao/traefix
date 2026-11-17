# Support AWS NLB QUIC passthrough with QUIC-LB connection IDs

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14287](https://github.com/traefik/traefik/pull/14287) |
| Tipo | Recurso |
| Área | entrypoint/http3 (QUIC) |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | medio |
| Autor | sylr |
| Aberto em | 2026-11-13 (última atualização: 2026-11-13) |
| Draft | sim |
| Branch-alvo | `v3.8` |
| Tamanho | +387/-2 em 4 arquivos |
| Labels | — |
| Status | pendente |

## O que faz

Quando AWS_LBC_QUIC_SERVER_ID está definido, os entrypoints HTTP/3 geram connection IDs de 17 bytes com o server ID, para que o NLB em modo passthrough roteie cada pacote ao Traefik correto. O transporte QUIC é criado pela entrypoint, já que o http3.Server não aceita ConnectionIDGenerator.

## Riscos e situação no upstream

Draft (draft=True), sem labels, e útil só para o caso AWS NLB com QUIC passthrough. Adiciona 387 linhas e cria um transporte QUIC próprio no entrypoint, o que é sensível. Sem review de maintainer ainda.

## Principais arquivos

- `entrypoint HTTP/3 (criação do transporte QUIC)`
- `configuração de entrypoint`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14287/head:upstream-pr-14287`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
