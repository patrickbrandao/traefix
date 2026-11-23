# 0009 — Log Gateway API informer list and watch errors

- **Status:** aplicada (`06fd84adf`, 2026-11-23)
- **Tipo:** bugfix
- **Prioridade:** média
- **Upstream:** traefik/traefik#14313 — commits `ee7ee348c` (branch v3.8)
- **Aplica limpo:** sim no texto, mas quebra a compilação dos testes (5 chamadas de `WatchAll` vindas de PRs importados)
- **Conflito com Traefix:** não

## O que mudou no oficial
Provider Kubernetes Gateway (`pkg/provider/kubernetes/gateway/client.go`): `WatchAll` recebe um `context.Context`
(`WatchAll(ctx, namespaces, stopCh)`), cada informer ganha um `WatchErrorHandler` que registra no log os erros de
list/watch com o nome do recurso, e o provider espera todos os recursos serem listados antes de carregar a
configuração. As RBACs de referência perdem o verbo `get` e o `referencegrants/status`
(`docs/content/reference/dynamic-configuration/kubernetes-gateway-rbac.yml`), com nota de migração em
`docs/content/migrate/v3.md` e aviso em `kubernetes-gateway.md`. Testes novos em `client_test.go`.

## Impacto no Traefix
Nenhuma alteração própria do Traefix no provider Gateway. Os arquivos foram alterados por PRs importados: #14288
(`eed796aa7`), #14223 (`2496445f2`), #13050 (`1b97230ac`), #12921 (`3c341dc6d`) e #14183 (`8d6fe0d0b`). O
cherry-pick aplica sem conflito, mas os testes trazidos pelos imports #12921, #13050, #14183 e #14223 chamam a
assinatura antiga em `kubernetes_test.go`, em 5 chamadas no total:

- `client.WatchAll(nil, make(chan struct{}))`: 4 chamadas;
- `client.WatchAll(nil, stopCh)`: 1 chamada (#14223).

`docs/content/migrate/v3.md` e `kubernetes-gateway.md` foram alterados pelo Traefix (imports), sem sobreposição
com as linhas novas.

## Plano de implementação
1. `git cherry-pick -x ee7ee348c`
2. Atualizar as chamadas antigas nos testes:
   `sed -i '' -e 's/client\.WatchAll(nil, /client.WatchAll(t.Context(), nil, /' pkg/provider/kubernetes/gateway/kubernetes_test.go`
3. Testes: `go vet ./pkg/provider/kubernetes/gateway/ && go test ./pkg/provider/kubernetes/gateway/` (passaram no
   worktree de teste depois do passo 2).
4. Operação: nenhuma mudança obrigatória nas RBACs do servidor de testes (as antigas seguem valendo).

## Critério de pronto
- Nenhuma chamada a `WatchAll` sem contexto; `go vet` e `go test` do pacote gateway sem erros.
- Erros de list/watch dos informers aparecem no log com o nome do recurso.
