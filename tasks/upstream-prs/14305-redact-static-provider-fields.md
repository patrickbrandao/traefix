# Redact static provider fields

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14305](https://github.com/traefik/traefik/pull/14305) |
| Tipo | Correção |
| Área | logs/redactor (providers docker, swarm, http, knative) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | jspdown |
| Aberto em | 2026-11-16 (última atualização: 2026-11-16) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +585/-4 em 6 arquivos |
| Labels | status/0-needs-triage, area/provider/docker, kind/bug/fix, size/M, area/provider/docker/swarm, area/provider/http, area/provider/k8s/knative |
| Status | importado com ajustes |

## O que faz

Campos sensíveis de providers (password do Docker/Swarm, token do Knative e headers do HTTP) não tinham a tag loggable:"false" e apareciam no log DEBUG da configuração estática. O PR adiciona a tag e remove export:"true" dos headers do provider HTTP.

## Riscos e situação no upstream

Base é v3.7, precisa de port. Remover export:"true" dos headers do HTTP muda o resultado de Anonymize, então a saída de config muda. Os arquivos de testdata podem conflitar com #14304. Sem review de maintainer ainda.

## Principais arquivos

- `pkg/provider/docker/shared.go`
- `pkg/provider/http/http.go`
- `pkg/provider/kubernetes/knative/kubernetes.go`
- `pkg/redactor/redactor_config_test.go`
- `pkg/redactor/testdata/anonymized-static-config.json`
- `pkg/redactor/testdata/secured-static-config.json`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14305/head:upstream-pr-14305`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- **Vazamento de segredos em log (confirmado):** `RemoveCredentials` só mascara campos com `loggable:"false"`. Ficavam em claro no log DEBUG da configuração estática: `password` dos providers Docker e Swarm (Basic Auth do endpoint), `token` do provider Knative e `headers` do provider HTTP (que costumam levar `Authorization`/API keys). Além disso, `headers` tinha `export:"true"`, então também saía no `Anonymize`.
- O PR é de um maintainer (jspdown) e só mexe em tags de struct; não há mudança de lógica. `username` continua logável, como no upstream (não é segredo).
- Auditei os demais campos parecidos dos providers (`token` do Nomad, Consul e Consul Catalog, `httpAuth` do Consul Catalog, `password` do etcd/redis): todos já têm `loggable:"false"`.
- Sobreposição com o #14304 (mesmo autor): os dois criam `fullStaticConfiguration()`, `TestRemoveCredentials_staticConfiguration` e `secured-static-config.json`.

**Ajustes**

- Combinado com o #14304, importado antes: reaproveitei `fullStaticConfiguration()`, o teste e o arquivo de testdata dele, e só acrescentei `username`/`password` no Docker e no Swarm, o provider Knative e os `headers` do HTTP. As tags foram aplicadas como no PR.
- `anonymized-static-config.json` e `secured-static-config.json` regenerados com `-update_expected` e revisados: `password`/`token` aparecem como `xxxx` e `headers` some nos dois.
- `go run ./internal/` não altera a documentação (as tags não afetam a referência).

**Testes:** `go build ./pkg/...` ok; `go test ./pkg/redactor/ ./pkg/provider/http/ ./pkg/provider/kubernetes/knative/ ./pkg/provider/docker/` ok; `golangci-lint` nesses pacotes com 0 issues.
