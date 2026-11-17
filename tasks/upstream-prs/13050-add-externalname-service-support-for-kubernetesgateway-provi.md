# Add ExternalName service support for KubernetesGateway provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13050](https://github.com/traefik/traefik/pull/13050) |
| Tipo | Recurso |
| Área | provider/kubernetes/gateway-api |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | aniskhalfallah |
| Aberto em | 2026-04-26 (última atualização: 2026-08-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +347/-25 em 4 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/provider/k8s/gatewayapi |
| Status | importado com ajustes |

## O que faz

Permite usar Services do tipo ExternalName como backend no provider KubernetesGateway, com o mesmo comportamento dos providers Ingress e CRD. Reutiliza a flag allowExternalNameServices, que é opt-in. Resolve o issue #12950.

## Riscos e situação no upstream

Label status/2-needs-review; usuários aguardam há meses e cobram atualização. A flag é opt-in e já existe nos outros providers, então o risco de segurança está contido. Toca apenas gateway/kubernetes.go e testes.

## Principais arquivos

- `pkg/provider/kubernetes/gateway/kubernetes.go`
- `pkg/provider/kubernetes/gateway/kubernetes_test.go`
- `pkg/provider/kubernetes/gateway/fixtures/httproute/with_external_name_http.yml`
- `pkg/provider/kubernetes/gateway/fixtures/httproute/with_external_name_https.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13050/head:upstream-pr-13050`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- Segurança: ExternalName como backend é o vetor clássico de SSRF/bypass em controllers (CVE-2021-25740): um dono de rota pode apontar para nomes internos ou para serviços de outros namespaces. O PR protege com a opção `allowExternalNameServices`, opt-in e desligada por padrão, como nos providers CRD e Ingress. Referências cross-namespace continuam exigindo ReferenceGrant antes de `getBackendAddresses`, então o ExternalName não burla essa checagem.
- Ao contrário do que diz a descrição, a opção não existia no provider Gateway: o PR a cria em `Provider`.
- Documentação da nova opção ausente no PR.
- Os testes do PR esperavam `https://` para a porta 443; no fork o esquema do backend Gateway API vem de `appProtocol`/BackendTLSPolicy, não da porta, então o resultado correto é `http://external.domain:443` sem BackendTLSPolicy.
- Conflitos no struct `Provider` e na tabela de testes (o fork tem QPS, Burst, Gateways, RetryMaxRequestBodyBytes, etc.).

Ajustes:
- Campo `AllowExternalNameServices` integrado ao struct atual.
- Testes reescritos como `TestLoadHTTPRoutes_externalNameService` (flag desligada não gera servidor; ligada gera `http://external.domain:80` e `:443`), em vez das expectativas completas do PR, que não batiam com a geração de routers do fork.
- Docs: linha na referência do provider (`kubernetes-gateway.md`) e `configuration-options.md` regenerado com `make generate`. O `make generate` também gerou as âncoras da tabela de annotations do #12921 em `gateway-api.md`, incluídas neste commit.

Testes: `go build ./...` ok; `go test ./pkg/provider/kubernetes/gateway/` ok; `golangci-lint run ./pkg/provider/kubernetes/gateway/...` 0 issues.
