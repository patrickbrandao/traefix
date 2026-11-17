# Fix Gateway API listener hostname ignored for TLS certificate selection

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14087](https://github.com/traefik/traefik/pull/14087) |
| Tipo | Correção |
| Área | provider/kubernetes-gateway, tls |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | aliamerj |
| Aberto em | 2026-10-17 (última atualização: 2026-10-24) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +399/-4 em 8 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/L, area/tls, area/provider/k8s/gatewayapi |
| Status | pendente |

## O que faz

Faz o provider Gateway API considerar o hostname do listener ao escolher o certificado TLS. Antes, certificados de dois listeners na mesma porta eram escolhidos so pelo SAN, podendo servir certificado errado para outro dominio. Resolve #12742.

## Riscos e situação no upstream

Altera o store de certificados (pkg/tls) e o provider Gateway (+399/-4). Mudanca de comportamento: SNI sem certificado restrito passa a cair no default. Limitacao conhecida: store indexado por conjunto de SANs. Label status/1-needs-design-review; sem reviews de maintainer.

## Principais arquivos

- `pkg/tls/certificate_store.go`
- `pkg/tls/tlsmanager.go`
- `pkg/tls/tls.go`
- `pkg/provider/kubernetes/gateway/kubernetes.go`
- `pkg/provider/kubernetes/gateway/fixtures/tls_certificates_hostnames.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14087/head:upstream-pr-14087`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
