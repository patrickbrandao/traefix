# Add load balancer strategy annotation for Ingress

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13002](https://github.com/traefik/traefik/pull/13002) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | PantelisK |
| Aberto em | 2026-04-17 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +181/-7 em 6 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/provider/k8s/ingress, size/M |
| Status | importado com ajustes |

## O que faz

Adiciona a annotation traefik.ingress.kubernetes.io/service.strategy em Services, para escolher wrr, p2c, hrw ou leasttime. Dá paridade com IngressRoute/TraefikService para quem usa Ingress padrão, como na migração do Nginx.

## Riscos e situação no upstream

Label status/2-needs-review; maintainer só confirmou o recebimento, sem revisão de código. Toca annotations.go e kubernetes.go do provider Ingress, com baixo risco de conflito.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/annotations.go`
- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/annotations_test.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13002/head:upstream-pr-13002`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- Mudança pequena e correta: a annotation `traefik.ingress.kubernetes.io/service.strategy` (no Service) é validada contra as quatro estratégias suportadas e um valor inválido descarta o backend com erro, como as outras annotations inválidas.
- Sem risco de segurança: a annotation fica no Service, só escolhe o algoritmo de balanceamento.
- Os testes do PR não compilavam no código atual (`pointer(...)` em vez de `new(...)`, `dynamic.KubernetesIngressMetadata` inexistente, faltava a metadata de observabilidade do Service).

**Ajustes**

- Testes adaptados ao formato atual de metadata e ao helper `new(...)`.
- Novo `TestLoadServiceUnsupportedStrategy` cobrindo o caminho de erro, que o PR não testava.

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress/` ok.
- `golangci-lint run ./pkg/provider/kubernetes/ingress/...`: 0 issues.
