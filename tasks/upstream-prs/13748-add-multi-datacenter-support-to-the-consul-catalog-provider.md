# Add multi-datacenter support to the Consul Catalog provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13748](https://github.com/traefik/traefik/pull/13748) |
| Tipo | Recurso |
| Área | provider/consulcatalog |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | zakiharis |
| Aberto em | 2026-08-21 (última atualização: 2026-09-02) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +926/-122 em 6 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/provider/consulcatalog |
| Status | pendente |

## O que faz

Novo providers.consulCatalog.endpoint.datacenters permite descobrir e combinar instâncias de vários datacenters Consul pelo mesmo endpoint. Cada DC tem cliente, watcher e retry próprios, mantém o último snapshot bom se um DC cai, e separa certificados de Consul Connect por DC.

## Riscos e situação no upstream

926 linhas, mexe no núcleo do provider e no Connect (certificados e transports por DC). Status/2-needs-review, sem review de maintainer. Mutuamente exclusivo com o datacenter singular.

## Principais arquivos

- `pkg/provider/consulcatalog/config.go`
- `pkg/provider/consulcatalog/config_test.go`
- `pkg/provider/consulcatalog/consul_catalog.go`
- `pkg/provider/consulcatalog/consul_catalog_test.go`
- `docs/content/reference/install-configuration/configuration-options.md`
- `docs/content/reference/install-configuration/providers/hashicorp/consul-catalog.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13748/head:upstream-pr-13748`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
