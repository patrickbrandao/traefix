# Add per-router label-based constraints to Docker and Swarm providers

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13128](https://github.com/traefik/traefik/pull/13128) |
| Tipo | Recurso |
| Área | provider/docker, provider/swarm |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | t0mtaylor |
| Aberto em | 2026-05-07 (última atualização: 2026-11-08) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +478/-6 em 7 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/docker, size/L, area/provider/docker/swarm |
| Status | adiado |

## O que faz

Adiciona a constraint por router traefik.<router>.constraint=Label(...), avaliada contra um novo mapa estático routerLabels (providers.docker.routerLabels e swarm). Roteadores que não casam são descartados antes do registro. Serve para separar roteamento entre camadas de rede, como interna e externa.

## Riscos e situação no upstream

Maintainer disse que a feature está no roadmap, mas sem prioridade no momento. Autor vem fazendo rebase sem resposta de revisão. Toca 7 arquivos dos providers docker e swarm.

## Principais arquivos

- `pkg/provider/docker/config.go`
- `pkg/provider/docker/shared.go`
- `pkg/provider/docker/config_test.go`
- `docs/content/reference/install-configuration/providers/docker.md`
- `docs/content/reference/install-configuration/providers/swarm.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13128/head:upstream-pr-13128`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- **Vazamento de escopo (segurança):** a poda de routers (`filterRoutersByConstraint`) roda logo após o decode das labels, **antes** de `provider.BuildRouterConfiguration`. Se todos os routers HTTP do container forem podados, o mapa fica vazio e `BuildRouterConfiguration` cria o **router padrão** com a `defaultRule` (`Host(nome-do-serviço)`). Ou seja, um serviço que só tinha o router `admin` com `constraint=Label(tier,int)` acaba exposto na instância `ext` pela regra padrão, e sem os middlewares (auth etc.) que estavam no router podado. É o oposto do objetivo do recurso.
- Caso TCP/UDP: `tcpOrUDP` é calculado depois da poda. Num container só TCP, cujos routers foram todos podados e que não declara serviços TCP explícitos, o fluxo cai no caminho HTTP e ganha um router HTTP padrão para um backend TCP.
- Semântica *default-permit*: uma instância sem `routerLabels` ignora todas as constraints e carrega os routers de todas as camadas. Isso precisa estar bem destacado na documentação, porque basta esquecer a opção para expor routers internos.
- O prefixo `traefik.` está fixo em `isRouterConstraintLabel`. Hoje funciona para Docker/Swarm, mas é frágil.
- No upstream: sem review; o maintainer disse que o recurso está no roadmap, mas sem prioridade, e pediu para o autor parar de rebasear.

**Veredito:** adiado. É um recurso grande (+478 linhas), sem design aprovado no upstream, e com um bug que expõe serviços pela regra padrão. Se for trazido, é preciso pelo menos: (1) calcular `tcpOrUDP` antes da poda; (2) não criar router padrão quando algum router do container foi podado por constraint; (3) testes cobrindo os dois casos.

**Testes:** não aplicável (nada importado).
