# Fix useBindPortIP not resolving Swarm published ports

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13570](https://github.com/traefik/traefik/pull/13570) |
| Tipo | Correção |
| Área | provider/docker/swarm (useBindPortIP) |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | rmartcas |
| Aberto em | 2026-07-27 (última atualização: 2026-09-22) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +573/-13 em 8 arquivos |
| Labels | status/2-needs-review, area/provider/docker, kind/bug/fix, size/L, area/provider/docker/swarm |
| Status | pendente |

## O que faz

O provider Swarm não preenchia os bindings de porta das tasks, então useBindPortIP caía no IP interno e na porta de destino, que pode ser inalcançável fora do overlay. Agora resolve a porta publicada por Service.Endpoint.Ports (ingress) ou Task.Status.PortStatus.Ports (host), casando porta e protocolo, incluindo portas atribuídas dinamicamente. Resolve #13239.

## Riscos e situação no upstream

Grande (+573/-13) e toca lógica de descoberta do Swarm, com teste de integração. Não tem efeito quando traefik.swarm.lbswarm=true. Status needs-review, sem review de maintainer ainda. PR base v3.7.

## Principais arquivos

- `pkg/provider/docker/pswarm.go`
- `pkg/provider/docker/pswarm_test.go`
- `pkg/provider/docker/config.go`
- `pkg/provider/docker/config_test.go`
- `pkg/provider/docker/builder_test.go`
- `integration/swarm_test.go`
- `integration/fixtures/swarm/use_bind_port_ip.toml`
- `docs/content/reference/install-configuration/providers/swarm.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13570/head:upstream-pr-13570`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
