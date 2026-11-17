# skip health_status: healthy events to prevent config churn

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13412](https://github.com/traefik/traefik/pull/13412) |
| Tipo | Correção |
| Área | provider/docker |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | KirylJazzSax |
| Aberto em | 2026-06-28 (última atualização: 2026-10-31) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +60/-3 em 2 arquivos |
| Labels | status/2-needs-review, area/provider/docker, kind/bug/fix, size/S, area/healthcheck |
| Status | importado com ajustes |

## O que faz

Ignora eventos Docker health_status: healthy para evitar que cada um dispare rebuild da configuração. A intenção é reduzir churn em containers que já estão healthy e têm health check frequente. Relacionado ao #13321, que descreve o churn.

## Riscos e situação no upstream

Risco de regressão a verificar antes de portar. Em pkg/provider/docker/config.go:229 os containers unhealthy são filtrados, então o evento healthy é o único gatilho para reincluir um container que saiu de unhealthy. A descrição não deixa claro se o PR compara o estado anterior. Maintainer pediu rebase para v3.6.

## Principais arquivos

- `pkg/provider/docker/pdocker.go`
- `pkg/provider/docker/pdocker_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13412/head:upstream-pr-13412`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- **Regressão grave:** o PR descarta **todo** evento `health_status: healthy`. Um container com healthcheck nasce como `starting` (é filtrado por não estar saudável) e depois passa a `healthy`. Esse evento de transição é exatamente o que coloca o container no roteamento, e seria ignorado. O container só entraria na configuração quando algum outro evento (`start`/`die` de qualquer container) disparasse um refresh. O mesmo acontece em `unhealthy -> healthy`: o container ficaria fora do roteamento depois de se recuperar.
- O teste do PR só valida a função isolada e cobre exatamente o caso errado ("healthy não dispara reload").
- O custo real dos eventos repetidos é o `ContainerList` + `inspect` de todos os containers a cada probe. O reload propriamente dito já é evitado pelo `configurationwatcher`, que descarta configuração idêntica (`Skipping unchanged configuration`).

**Ajustes (reimplementação)**

- `shouldHandleEvent(event, healthStatuses)` guarda o último `health_status` de cada container (por `Actor.ID`) e só ignora o evento quando o status é igual ao último visto. Toda transição continua disparando refresh.
- `start` e `die` limpam a entrada do container, então o mapa fica limitado aos containers vivos e é recriado a cada reconexão do stream (não cresce sem limite).
- Testes em tabela: transições, repetições de `healthy` e `unhealthy`, isolamento entre containers, reset em restart e limpeza do mapa.

**Testes:** `go build ./pkg/...` ok; `go test ./pkg/provider/docker/` ok; `golangci-lint run ./pkg/provider/docker/...` com 0 issues.
