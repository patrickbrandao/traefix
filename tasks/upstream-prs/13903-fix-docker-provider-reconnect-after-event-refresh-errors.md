# Fix Docker provider reconnect after event refresh errors

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13903](https://github.com/traefik/traefik/pull/13903) |
| Tipo | Correção |
| Área | provider/docker |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | herefindalex |
| Aberto em | 2026-09-14 (última atualização: 2026-09-21) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +87/-4 em 2 arquivos |
| Labels | status/2-needs-review, area/provider/docker, kind/bug/fix, size/M |
| Status | importado com ajustes |

## O que faz

Propaga erros de listagem de containers durante eventos start, die e health-status. Antes o erro era só logado e o provider não entrava no caminho de retry, podendo deixar a configuração desatualizada até o próximo evento. Agora o erro reinicia a subscrição com o backoff existente.

## Riscos e situação no upstream

Afeta o loop de watch do Docker; tem teste de regressão com falha transitória e race test declarado. Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/provider/docker/pdocker.go`
- `pkg/provider/docker/pdocker_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13903/head:upstream-pr-13903`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- Bug real: quando o `ContainerList` falhava durante o refresh disparado por um evento, o erro só era logado e o loop seguia. A configuração ficava velha até outro evento, sem passar pelo caminho de reconexão com backoff. O PR propaga o erro, então a operação termina e o `RetryNotify` recria o cliente e a assinatura de eventos, como já acontecia com o erro da listagem inicial.
- Sem risco de segurança. Um erro transitório causa uma reconexão com backoff, não um loop apertado.
- O teste do PR fazia type assertion em `http.Flusher` no `ResponseWriter`, contra a convenção do AGENTS.md.

**Ajustes**

- Conflito resolvido com o #13412 (importado com ajustes), que também mexe no loop de eventos: agora é `shouldHandleEvent` (dedup de `health_status`) seguido da propagação do erro. O mapa de status de saúde é recriado a cada reconexão.
- O teste foi incorporado ao `pdocker_test.go` criado pelo #13412, usando `http.NewResponseController(rw).Flush()`.
- Confirmei que o teste falha sem a correção.

**Relação com outros PRs:** o #13578 (timeout do stream) e o #12706 (polling) ficaram adiados. Este PR cobre o caminho concreto de configuração velha.

**Testes:** `go build ./pkg/...` ok; `go test ./pkg/provider/docker/` ok (também com `-race -count=3` nos testes novos); `golangci-lint run ./pkg/provider/docker/...` com 0 issues.
