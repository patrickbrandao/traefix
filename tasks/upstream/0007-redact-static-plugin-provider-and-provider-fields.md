# 0007 — Redact the static plugin provider configuration + Redact static provider fields

- **Status:** aplicada (`7cb758e39`, 2026-11-23)
- **Tipo:** segurança
- **Prioridade:** média (a correção funcional já está no Traefix desde os imports de 2026-11-16; falta alinhar com a versão final do oficial)
- **Upstream:** traefik/traefik#14304 — commits `26878fab5` (branch v2.11); traefik/traefik#14305 — commits `1a2803a35` (branch v3.7); resolução dos dois no merge `ece978d30` (v2.11 → v3.7)
- **Aplica limpo:** não (arquivos em conflito: `pkg/redactor/redactor.go`, `pkg/redactor/redactor_config_test.go`, `pkg/redactor/testdata/secured-static-config.json`)
- **Conflito com Traefix:** não

Corresponde às tasks `tasks/upstream-prs/14304-redact-the-static-plugin-provider-configuration.md` e
`tasks/upstream-prs/14305-redact-static-provider-fields.md`, as duas com o status "importado com ajustes"
(commits locais `9a8f70a40` e `e8c6ace2a`). Os dois PRs foram mergeados no oficial.

## O que mudou no oficial
- #14304: `doOnStruct` em `pkg/redactor/redactor.go` passa a tratar `static.PluginConf` como `dynamic.PluginConf`:
  as chaves dos plugins de provider (`providers.plugin.<nome>`) ficam e os valores são esvaziados no log
  `Static configuration loaded`. O PR também tira os auxiliares internos `anonymize`/`removeCredentials`/`marshal`
  com o parâmetro `indent`, e os testes passam a chamar `Anonymize`/`RemoveCredentials` e a indentar o JSON só ao
  gravar com `-update_expected`. A configuração dinâmica de teste vira `fullDynamicConfiguration()` (antes era
  `fullDynConf` em `init`), e alguns testes mudam de nome (`TestSecure_dynamicConfiguration` →
  `TestRemoveCredentials_dynamicConfiguration`, `TestDo_staticConfiguration` → `TestAnonymize_staticConfiguration`).
- #14305: `loggable:"false"` em `password` dos providers Docker e Swarm (`pkg/provider/docker/shared.go`), em
  `token` do Knative e em `headers` do provider HTTP (que perde o `export:"true"`).

## Impacto no Traefix
- **A correção funcional já está aplicada e é idêntica à do oficial.** `pkg/redactor/redactor.go` do Traefix já
  tem a checagem de `static.PluginConf`; as tags de Docker, Swarm, Knative e HTTP são iguais; os arquivos de
  `pkg/redactor/testdata/` são idênticos aos de `upstream/master`. A diferença está só no que o import deixou de fora
  para não aumentar o diff: a remoção dos auxiliares com `indent` e a reorganização dos testes.
- **Os campos secretos do Traefix já são redigidos pelo mecanismo existente, e a mudança oficial não pede tag ou
  lista nova.** `RemoveCredentials` troca por `xxxx` todo campo com `loggable:"false"`, e `Anonymize` troca todo
  campo sem `export:"true"`. Teste temporário no worktree de teste, já com o `redactor.go` final do oficial, com
  `static.Configuration{Cluster: ...}`: `token`, `additionalTokens`, `redis.username`, `redis.password` e
  `redis.sentinelPassword` de `pkg/cluster/config.go` saíram como `xxxx` nas duas funções. `RequestHeaders` do
  forwardAuth (`pkg/config/dynamic/middlewares.go`) e `Headers` da exportação do fingerprint
  (`pkg/config/static/entrypoints_fingerprint.go`) já são cobertos por `pkg/redactor/redactor_fingerprint_test.go`,
  que fica como está (arquivo exclusivo do Traefix).
- Não há teste fixo para os segredos do `cluster`. Vale acrescentar um em `redactor_fingerprint_test.go`, para que
  uma mudança futura de tag não os exponha sem ninguém perceber.
- `pkg/provider/docker/shared.go` difere do oficial por causa do import do #13404 (`5e7d3690f`, `ExposedPorts`),
  não por esta task. Prevalece o código local.

## Plano de implementação
1. Não fazer cherry-pick de `26878fab5`: é do branch v2.11 e acrescenta o import
   `github.com/traefik/traefik/v2/pkg/config/static`. Não fazer cherry-pick de `1a2803a35`: as tags já estão
   aplicadas e o resto conflita com os testes do import.
2. Adotar a versão final do oficial nos dois arquivos que ainda diferem:
   `git checkout upstream/master -- pkg/redactor/redactor.go pkg/redactor/redactor_config_test.go`
   (os de `testdata/` já são iguais). Não mexer em `pkg/redactor/redactor_fingerprint_test.go`.
3. Acrescentar em `pkg/redactor/redactor_fingerprint_test.go` um teste `TestRemoveCredentials_clusterSecrets` que
   preencha `Cluster.Token`, `AdditionalTokens`, `Redis.Username`, `Redis.Password` e `Redis.SentinelPassword` e
   verifique que nenhum valor aparece na saída de `RemoveCredentials` nem na de `Anonymize`.
4. Testes: `go test ./pkg/redactor/ ./pkg/provider/docker/ ./pkg/provider/http/ ./pkg/provider/kubernetes/knative/`
   (os de `pkg/redactor` passaram no worktree de teste com os arquivos do oficial).
5. Depois de aplicar, marcar as tasks 14304 e 14305 de `tasks/upstream-prs/` como absorvidas.

## Critério de pronto
- `pkg/redactor/redactor.go` e `redactor_config_test.go` iguais aos de `upstream/master`; `redactor_fingerprint_test.go` mantido.
- Teste novo garantindo a redação dos segredos do `cluster`.
- `go test ./pkg/redactor/...` e `make lint` sem erros.
