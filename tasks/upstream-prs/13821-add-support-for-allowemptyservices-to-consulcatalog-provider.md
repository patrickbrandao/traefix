# Add support for allowEmptyServices to consulcatalog provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13821](https://github.com/traefik/traefik/pull/13821) |
| Tipo | Recurso |
| Área | provider/consulcatalog |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | oleg-z |
| Aberto em | 2026-09-01 (última atualização: 2026-09-03) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +131/-15 em 6 arquivos |
| Labels | status/1-needs-design-review, size/M, area/provider/consulcatalog |
| Status | importado com ajustes |

## O que faz

Nova opção allowEmptyServices no consulCatalog mantém routers e middlewares mesmo sem instâncias saudáveis. Evita que um router some quando um container falha no registro do Consul e o tráfego cai em outro serviço.

## Riscos e situação no upstream

Pequeno (131 linhas), com docs e testes. Status status/1-needs-design-review, e o maintainer disse que vai analisar o impacto antes de avançar.

## Principais arquivos

- `pkg/provider/consulcatalog/config.go`
- `pkg/provider/consulcatalog/config_test.go`
- `pkg/provider/consulcatalog/consul_catalog.go`
- `docs/content/reference/install-configuration/providers/hashicorp/consul-catalog.md`
- `docs/content/reference/static-configuration/file.toml`
- `docs/content/reference/static-configuration/file.yaml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13821/head:upstream-pr-13821`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- O recurso é coerente com o `allowEmptyServices` dos outros providers. O fetch do Consul usa `passingOnly=false`, então as instâncias unhealthy chegam ao builder e podem gerar routers e serviços vazios (resposta 503 em vez de o router sumir, o que pode fazer o tráfego cair em outro router de menor prioridade).
- **Bug (o mesmo do upstream 13328 no Docker):** para itens unhealthy, os builders retornam antes de `addServer`. Se a tag define `loadbalancer.server.port`, o servidor decodificado da tag (sem URL/endereço) fica no load balancer. Com só instâncias unhealthy, isso dá 500 (`unsupported protocol scheme ""`) em vez de 503. **Pior:** com instâncias healthy e unhealthy do mesmo serviço, o merge junta o servidor vazio ao servidor válido e parte das requisições recebe 500.
- O padrão continua `false`, então o comportamento atual não muda.
- O conflito no teste veio de um teste local mais recente (`Test_buildConfigurationItemKeyCollision`); os dois foram mantidos.

**Ajustes**

- Para itens unhealthy, os builders HTTP, TCP e UDP agora limpam `Servers` de todos os serviços do item, com um comentário explicando o porquê.
- Três casos novos de teste (HTTP só unhealthy com label de porta; HTTP healthy + unhealthy com label de porta; TCP unhealthy com label de porta). Confirmei que eles falham sem o ajuste.
- `configuration-options.md` regenerado com `go run ./internal/`.

**Testes:** `go build ./pkg/...` ok; `go test ./pkg/provider/consulcatalog/` ok; `golangci-lint run ./pkg/provider/consulcatalog/...` com 0 issues.
