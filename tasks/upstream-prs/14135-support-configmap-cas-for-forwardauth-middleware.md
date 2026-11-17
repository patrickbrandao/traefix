# Support ConfigMap CAs for ForwardAuth middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14135](https://github.com/traefik/traefik/pull/14135) |
| Tipo | Recurso |
| Área | middleware/forwardauth, provider/kubernetes/crd |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | manenim |
| Aberto em | 2026-10-24 (última atualização: 2026-11-15) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +209/-0 em 9 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/authentication, area/provider/k8s/crd |
| Status | importado com ajustes |

## O que faz

Adiciona forwardAuth.tls.caConfigMap ao Middleware CRD, para carregar CAs de um ConfigMap no namespace do Middleware, com a mesma lógica de tls.ca/ca.crt. caSecret continua suportado; configurar as duas fontes gera erro. Inclui CRDs regeneradas e apply configuration.

## Riscos e situação no upstream

Muda CRD e código gerado (make generate-crd), então o diff é grande mas mecânico. Pode conflitar com o código gerado do fork. Status: needs-review, base master. Testes e docs incluídos.

## Principais arquivos

- `pkg/provider/kubernetes/crd/kubernetes.go`
- `pkg/provider/kubernetes/crd/kubernetes_test.go`
- `pkg/provider/kubernetes/crd/fixtures/with_auth.yml`
- `docs/content/reference/routing-configuration/http/middlewares/forwardauth.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14135/head:upstream-pr-14135`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

Veredito: **importado com ajustes**.

Achados no PR original:
- PR aberto, sem review de maintainer (só o agente automático, sem achados). A lógica reaproveita `loadCAConfigMap`/`getCABlocksFromConfigMap`, que já existem e já são usados pelo provider, então não precisa de RBAC novo. A exclusão mútua com `caSecret` é verificada antes de carregar qualquer fonte. A busca se restringe ao namespace do Middleware, sem acesso cross-namespace (há teste).
- Risco a conhecer: a CA decide em quais auth servers o Traefik confia. Quem puder editar o ConfigMap pode redirecionar a confiança (MITM do auth server). Pelo threat model do projeto isso exige permissão de escrita de configuração no namespace, então não é vulnerabilidade, mas ConfigMaps costumam ter RBAC mais permissivo que Secrets.
- O PR partia da assinatura antiga de `createForwardAuthMiddleware` (função livre, 2 retornos, variável `k8sClient`), incompatível depois do #13832. A doc conflitava com as seções de `service`/`h2c`/`timeout`.

Ajustes:
- Código adaptado à assinatura atual (método do `Provider`, 3 retornos, `client`). O teste `TestCreateForwardAuthMiddleware_CAConfigMap` chama o método e verifica que nenhum service é criado.
- Expectativa do fixture `forwardauth-configmap` atualizada com o timeout padrão (#14024).
- Doc reinserida à mão, com um aviso sobre quem pode alterar o ConfigMap.
- CRDs e applyconfiguration regenerados com `make generate-crd`, e `go generate` rodado.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/provider/kubernetes/crd/`: ok.
- `golangci-lint run ./pkg/provider/kubernetes/crd/...`: 0 issues.
