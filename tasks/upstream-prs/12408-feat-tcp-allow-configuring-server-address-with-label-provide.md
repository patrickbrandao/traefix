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
| Status | rejeitado |

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

## Revisão (2026-11-16)

**Achados no PR original**

- O PR troca a tag `label:"-"` de `dynamic.TCPServer.Address` por `label:"address"`. Isso vale para **todos** os providers baseados em labels/tags (Docker, Swarm, ECS, Consul Catalog, Nomad), não só para o Docker, ao contrário do que o título sugere.
- Escopo: hoje um container só consegue apontar o tráfego para si mesmo (o IP vem da rede do container; a label só escolhe a porta). Com o PR, quem controla as labels de qualquer container pode mandar o tráfego TCP de um router para um endereço arbitrário (outros hosts internos, serviços de metadados de nuvem, IPs fora do cluster). O HTTP mantém deliberadamente `Server.URL` com `label:"-"`, e o PR quebra essa simetria sem discussão de design.
- O modelo de ameaça do upstream confia nas labels, mas isso amplia o que um container pode fazer fora do próprio escopo, exatamente o tipo de mudança que o Traefix quer evitar.
- O atalho `if loadBalancer.Servers[0].Address != "" { return nil }` também pula a validação de porta e de rede. Fica só um teste, e só para o Docker.
- Sem review de maintainer, sem documentação.

**Veredito:** rejeitado. Se o caso de uso (backend TCP externo) for necessário, o caminho seguro é declarar o serviço no file provider e referenciá-lo com `@file` a partir das labels.

**Testes:** não aplicável (nada importado).
