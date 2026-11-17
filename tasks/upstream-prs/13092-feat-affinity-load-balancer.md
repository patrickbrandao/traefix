# feat: Affinity Load Balancer

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13092](https://github.com/traefik/traefik/pull/13092) |
| Tipo | Recurso |
| Área | server/loadbalancer |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | StillNotUsefull |
| Aberto em | 2026-05-03 (última atualização: 2026-05-05) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +1186/-33 em 12 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/server |
| Status | pendente |

## O que faz

Adiciona a estratégia affinity ao load balancer, que fixa uma chave extraída da requisição (regex sobre o path ou um header) a um backend, usando jump consistent hash. Resolve o caso de protocolos como XHTTP/SplitHTTP, em que várias requisições de uma mesma sessão precisam chegar ao mesmo backend.

## Riscos e situação no upstream

Label needs-design-review e maintainer pediu rebase sobre master (base atual v3.7). +1186/-33 em 12 arquivos e mantém estado em memória com eviction, o que traz questões de memória e de concorrência. Não há revisão de código ainda.

## Principais arquivos

- `pkg/server/service/loadbalancer/affinity/affinity.go`
- `pkg/server/service/service.go`
- `pkg/config/dynamic/http_config.go`
- `docs/content/reference/routing-configuration/http/load-balancing/service.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13092/head:upstream-pr-13092`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
