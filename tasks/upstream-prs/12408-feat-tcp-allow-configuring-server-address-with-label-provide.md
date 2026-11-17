# feat(tcp): allow configuring server address with label providers

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12408](https://github.com/traefik/traefik/pull/12408) |
| Tipo | Recurso |
| Área | provider/docker |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | divyanshu-vashu |
| Aberto em | 2025-12-11 (última atualização: 2025-12-18) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +71/-12 em 3 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/docker, size/S |
| Status | pendente |

## O que faz

No provider Docker, permite definir o endereço de servers TCP via labels, em vez de sempre autodetectar IP e porta do container. Iguala o comportamento ao que já existe para server.url em HTTP. Corrige a issue #12386.

## Riscos e situação no upstream

Pequeno (3 arquivos, +71/-12) e com testes. Só label de triagem, sem review de maintainer ainda.

## Principais arquivos

- `pkg/provider/docker/config.go`
- `pkg/provider/docker/config_test.go`
- `pkg/config/dynamic/tcp_config.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12408/head:upstream-pr-12408`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
