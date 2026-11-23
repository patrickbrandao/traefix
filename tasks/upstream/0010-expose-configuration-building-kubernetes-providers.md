# 0010 — Expose the configuration building of the Kubernetes providers

- **Status:** aplicada (`8618f77ba`, 2026-11-23)
- **Tipo:** refactor
- **Prioridade:** média (pré-requisito da 0012 e de qualquer mudança futura do oficial nos providers Kubernetes)
- **Upstream:** traefik/traefik#14286 — commits `9f8e11099` (branch v3.8)
- **Aplica limpo:** não (arquivos em conflito, aplicado depois da 0009: `pkg/provider/kubernetes/crd/kubernetes.go`, `pkg/provider/kubernetes/gateway/kubernetes.go`, `pkg/provider/kubernetes/ingress/kubernetes.go`, `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`; sem a 0009, também `gateway/client.go` e `gateway/client_test.go`)
- **Conflito com Traefix:** não (conflitos só com PRs do oficial importados; nenhuma decisão própria do Traefix é afetada)

## O que mudou no oficial
Refatoração grande (29 arquivos, cerca de 1.000 linhas nos dois sentidos) para expor a construção da configuração dos
providers CRD, Gateway, Ingress e ingress-nginx:

- CRD e Gateway: a interface `Client` e o `clientWrapper` viram um tipo concreto `*Client`, criado por
  `NewClient(provider, clientsets...)`; as funções `new*Client*` viram `new*Clientsets*`.
- Ingress: `NewClient(p, clientset)` (a interface `Client` continua); ingress-nginx: `NewClient(clientset)`.
- `loadConfigurationFromCRD`, `loadConfigurationFromGateways`, `loadConfigurationFromIngresses` e o `loadConfiguration`
  do ingress-nginx passam a se chamar `BuildConfiguration` (exportado); o campo `client` do `Provider` do Gateway vira
  `k8sClient`.
- `pkg/server/aggregator.go` ganha `Aggregate(configurations, defaultEntryPoints, strictTLSOptions)`, e
  `ConfigurationWatcher.applyConfigurations` passa a chamá-lo no lugar de `mergeConfiguration` + `applyModel` +
  `resolveHTTPTLSOptions` (mesma sequência).
- `.golangci.yml`: a exclusão de `funlen` passa de `loadConfigurationFromCRD` para `BuildConfiguration`.

## Impacto no Traefix
- **Ganchos do Traefix:** `pkg/server/configurationwatcher.go` e `aggregator.go` não foram alterados pelo Traefix. O
  transformador da malha (`watcher.AddTransformer(node.Transform)`, em `cmd/traefik/traefix_mesh.go`) roda antes da
  agregação, e `Aggregate` faz a mesma sequência de antes, então a malha não é afetada.
- **PRs importados que precisam de adaptação** (código de produção, resolvido no worktree de teste):
  - CRD: `nodePortLBNodes(client Client, ...)` do #12048 (`d8e91755f`) e `createPluginMiddleware` passam a receber
    `*Client`; `createForwardAuthMiddleware` do #13832 (`c0c3885a2`), com o retorno do serviço, mantém a assinatura
    local e passa a receber `client *Client`. O import `k8s.io/utils/ptr` do Traefix fica junto com `kclientset`.
  - Gateway: o endereço de status por Gateway do #12921 (`3c341dc6d`, com a validação de nome e namespace) é mantido,
    trocando `p.client.GetService` por `p.k8sClient.GetService`.
  - Ingress: o cache de anotações do #13267 (`d0af60c02`: `getRouterConfig`, `pruneRouterConfigCache`,
    `routerConfigCacheKey`) é mantido, e o `applyRouterTransform` e o `newK8sClient` do oficial entram depois dele.
  - ingress-nginx: no teste, manter `setDefaultMaxIdleConnsPerHost(test.expected)` (import #13229) e chamar
    `p.BuildConfiguration(t.Context())`.
- **Testes dos imports que usam os nomes antigos** (não conflitam no texto, mas não compilam):
  - `newClientImpl(` → `NewClient(&Provider{}, ` em `crd`, `gateway` e `ingress` (não no knative, que o oficial não mudou);
  - `newClient(kubeClient)` → `NewClient(kubeClient)` no ingress-nginx (`build_test.go`, `kubernetes_test.go`);
  - `p.loadConfigurationFromIngresses(` / `p.loadConfigurationFromGateways(` / `p.loadConfiguration(` → `p.BuildConfiguration(`;
  - `Provider{client: client}` e `client: client,` em literais de `Provider` do gateway → `k8sClient:`.

## Plano de implementação
1. Aplicar antes a 0009 (o refactor parte do `WatchAll` com contexto).
2. `git cherry-pick -x 9f8e11099` e resolver os quatro arquivos como descrito acima, mantendo toda a lógica dos
   imports e adotando os nomes e tipos do oficial.
3. Atualizar os testes dos imports (lista acima); `gofmt -w` em `crd/kubernetes.go` (ordem dos imports).
4. Testes: `go build ./... && go vet ./pkg/provider/kubernetes/... && go test ./pkg/provider/kubernetes/... ./pkg/server/`
   (tudo passou no worktree de teste com essa resolução) e `make lint` (exclusão de `funlen` renomeada).

## Critério de pronto
- `BuildConfiguration` exportado nos quatro providers; `server.Aggregate` usado pelo `ConfigurationWatcher`.
- Todos os recursos dos PRs importados mantidos (nodePort Local, forwardAuth com serviço, endereço de status por
  Gateway, cache de anotações do Ingress, keep-alive do ingress-nginx).
- `go test ./pkg/provider/kubernetes/... ./pkg/server/` e `make lint` sem erros.
