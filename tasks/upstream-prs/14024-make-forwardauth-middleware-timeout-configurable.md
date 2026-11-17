# Make ForwardAuth middleware timeout configurable

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14024](https://github.com/traefik/traefik/pull/14024) |
| Tipo | Recurso |
| Área | middleware/forwardAuth |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | bhaskargurram-ai |
| Aberto em | 2026-10-08 (última atualização: 2026-10-09) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +192/-1 em 18 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/authentication, area/provider/k8s/crd |
| Status | importado com ajustes |

## O que faz

Adiciona a opcao timeout ao middleware ForwardAuth, que antes era fixo em 30s. Padrao 30s (sem mudanca), 0 desabilita, e em caso de estouro responde 500. Funciona em file, labels, KV e CRD.

## Riscos e situação no upstream

Logica pequena (+192/-1), mas toca varios arquivos de CRD gerado e docs. Sem reviews de maintainer; label status/2-needs-review. Resolve #9857.

## Principais arquivos

- `pkg/middlewares/auth/forward.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/crd/kubernetes.go`
- `docs/content/reference/routing-configuration/http/middlewares/forwardauth.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14024/head:upstream-pr-14024`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

Veredito: **importado com ajustes**.

Achados no PR original:
- PR aberto, sem review de maintainer. Mudança pequena, sem dependências novas (`paerser/types` já é usado).
- **Risco de DoS por configuração:** o timeout passa a vir de `config.Timeout`, e o padrão de 30s só existe via `SetDefaults()`. Conferi que todos os caminhos reais aplicam o default: file e HTTP provider (paerser `file.DecodeContent`), labels e KV (paerser), e o CRD (chama `SetDefaults()`). O ingress-nginx não usa `dynamic.ForwardAuth`. Mesmo assim, `timeout: 0` desliga o limite em silêncio, e um valor negativo também (o `http.Client` trata como "sem timeout").
- O timeout só valia para o modo `address`. O modo `service` (#13832), importado antes, tinha um limite fixo de 30s via contexto, que agora segue a opção.
- Os arquivos gerados do PR (CRDs, docker-labels) conflitavam com os PRs anteriores e foram regenerados em vez de aplicados.

Ajustes:
- Timeout negativo é erro de configuração (`TestForwardAuthNegativeTimeout`), e `timeout: 0` gera um log `warn` de DoS, como os de `maxBodySize`/`maxResponseBodySize`.
- O modo `service` usa o mesmo timeout configurado. `TestForwardAuthServiceTimeout` passa a configurar pela opção, e a doc explica que, nesse modo, a resposta é o erro do service (geralmente 504).
- Novo teste `TestForwardAuth_defaultTimeout` em `pkg/config/dynamic`, que garante que a decodificação de arquivo aplica o default de 30s e respeita `2m` e `0`.
- `createForwardAuthMiddleware` (CRD) adaptado à assinatura atual, com expectativas de teste atualizadas para os casos com `service` e `integration/testdata/rawdata-crd.json` (`"timeout": "30s"`).
- `docker-labels.yml` atualizado à mão (não é gerado pelo `go generate`); CRDs, applyconfiguration, deepcopy e `file.toml`/`file.yaml` regenerados com `make generate-crd` e `go generate`.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/middlewares/auth/ ./pkg/provider/kubernetes/crd/ ./pkg/config/... ./pkg/provider/kv/`: ok.
- `golangci-lint run ./pkg/middlewares/auth/... ./pkg/provider/kubernetes/crd/... ./pkg/config/... ./pkg/provider/kv/...`: 0 issues.
