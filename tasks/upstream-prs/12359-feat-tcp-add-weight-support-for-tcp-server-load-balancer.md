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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:

- O `zz_generated.deepcopy.go` foi editado à mão e ficou incompleto. `TCPServersLoadBalancer.DeepCopyInto` continuava fazendo `copy()` raso do slice `Servers`, então as cópias "profundas" dividiam o mesmo ponteiro `Weight`.
- A documentação foi aplicada em caminhos que não existem neste código (`reference/routing-configuration/other-providers/file.*`).
- Não havia testes.
- `WRRLoadBalancer` (TCP) já trata pesos 0/negativos sem loop infinito: servidores com peso `<= 0` nunca são escolhidos e "todos com peso 0" vira erro. O PR não muda esse código.

Ajustes:

- `zz_generated.deepcopy.go` regenerado com `deepcopy-gen` v0.37.1, a mesma versão de `script/code-gen.sh`. O slice `Servers` agora é copiado elemento a elemento.
- Removido o comentário redundante no campo `Weight`, para alinhar com `dynamic.Server`.
- Docs: `weight` adicionado em `reference/dynamic-configuration/file.{toml,yaml}` e na tabela de opções de `reference/routing-configuration/tcp/service.md`.
- Novo teste `TestManager_BuildTCP_ServerWeight`: três backends reais com pesos 3, 1 (padrão) e 0. Ele confere a distribuição exata em 8 conexões (6/2/0).

Testes:

- `go build ./...`: ok.
- `go test ./pkg/server/service/tcp/... ./pkg/tcp/... ./pkg/config/... ./pkg/provider/docker/... ./pkg/provider/kv/... ./pkg/provider/file/...`: ok. O teste novo também passou com `-race -count=5`.
- `golangci-lint run ./pkg/server/service/tcp/... ./pkg/config/dynamic/...`: 0 issues.
