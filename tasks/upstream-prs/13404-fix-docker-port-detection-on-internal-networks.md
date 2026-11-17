# Fix Docker port detection on internal networks

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13404](https://github.com/traefik/traefik/pull/13404) |
| Tipo | Correção |
| Área | provider/docker |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | amazon7737 |
| Aberto em | 2026-06-26 (última atualização: 2026-08-26) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +37/-3 em 4 arquivos |
| Labels | status/2-needs-review, area/provider/docker, kind/bug/fix, size/S |
| Status | importado |

## O que faz

A detecção automática de porta no provider Docker passa a cair em Config.ExposedPorts quando NetworkSettings.Ports não tem bindings, como em containers de rede interna sem publish. A precedência fica: label de porta, bindings de runtime, exposed ports. Fixes #11878.

## Riscos e situação no upstream

Escopo pequeno (4 arquivos, +37/-3). Maintainer pediu rebase para v3.6 (branch de bugfix) e marcou need-triage. Sem review ainda. Afeta a detecção de porta, então vale testar com containers reais.

## Principais arquivos

- `pkg/provider/docker/data.go`
- `pkg/provider/docker/shared.go`
- `pkg/provider/docker/builder_test.go`
- `pkg/provider/docker/shared_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13404/head:upstream-pr-13404`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- Containers ligados só a redes `internal: true` não têm `NetworkSettings.Ports`, então a detecção automática de porta falhava ("port is missing"). O PR passa a usar `Config.ExposedPorts` como fallback e mantém a precedência: label explícita, depois portas em runtime, depois portas expostas (a menor delas).
- A mudança é pequena e determinística (as portas são ordenadas). Não afeta o Swarm, que não preenche `ExposedPorts`.
- Segurança: a porta vem da imagem/config do próprio container e o IP continua sendo o da rede dele, então nada sai do escopo do container. O único efeito é que containers que antes ficavam sem servidor (erro de porta ausente) agora recebem um servidor, que é o objetivo da correção.

**Ajustes:** nenhum.

**Testes:** `go build ./pkg/... ./internal/...` + `go vet ./cmd/...` ok (o link dos binários em `go build ./...` falhou por falta de espaço em disco na máquina, sem erro de compilação); `go test ./pkg/provider/docker/` ok; `golangci-lint run ./pkg/provider/docker/...` com 0 issues.
