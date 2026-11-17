# feat(kubernetes): support multiple ingress endpoint IPs

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13831](https://github.com/traefik/traefik/pull/13831) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | mornhussakuyo-hub |
| Aberto em | 2026-09-03 (última atualização: 2026-10-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +144/-4 em 4 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/k8s/ingress, size/M |
| Status | importado com ajustes |

## O que faz

providers.kubernetesIngress.ingressEndpoint.ip passa a aceitar uma lista separada por vírgulas de IPv4 e IPv6, publicada no status do Ingress. Serve para dual-stack e para status diferente do Service publicado. A forma de um único IP continua compatível.

## Riscos e situação no upstream

Pequeno (144 linhas), com docs e testes. Base master, status/2-needs-review, review não verificado. O autor aponta falhas pré-existentes de fixture na suíte do pacote, não relacionadas.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/kubernetes_test.go`
- `docs/content/reference/install-configuration/configuration-options.md`
- `docs/content/reference/install-configuration/providers/kubernetes/kubernetes-ingress.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13831/head:upstream-pr-13831`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- `ingressEndpoint.ip` passa a aceitar uma lista de IPs separados por vírgula (IPv4/IPv6), com remoção de duplicados. Com um único IP, o formato legado (`{IP, Hostname}` na mesma entrada) é mantido.
- Os valores não são validados como IP: um item inválido (por exemplo, um hostname colocado em `ip`) vai parar no status do Ingress e o API server rejeita o update inteiro. Antes, um valor único inválido também falhava do mesmo jeito, então não é uma regressão, mas com listas fica mais fácil errar.
- Sem impacto de segurança: é configuração de instalação (confiável).

**Ajustes**

- Itens que não são IP válido (`net.ParseIP`) são descartados com log de erro, em vez de quebrar a atualização de status de todos os Ingresses. Se todos forem inválidos e não houver hostname, retorna erro e mantém o status atual, em vez de apagá-lo.
- `TestUpdateIngressStatusPriority` do PR não foi trazido: ele fixava a mensagem de erro antiga, que mudou com #12281. `TestIngressEndpointStatus` ganhou um caso de IP inválido.
- Docs: tabela do provider e `configuration-options.md` (regenerado).

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress/` ok.
- `golangci-lint run ./pkg/provider/kubernetes/ingress/...`: 0 issues.
