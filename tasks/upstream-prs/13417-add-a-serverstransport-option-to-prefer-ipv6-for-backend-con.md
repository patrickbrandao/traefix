# Add a ServersTransport option to prefer IPv6 for backend connections

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13417](https://github.com/traefik/traefik/pull/13417) |
| Tipo | Recurso |
| Área | service/serversTransport |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | RamazanKara |
| Aberto em | 2026-06-29 (última atualização: 2026-10-18) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +157/-4 em 17 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/service |
| Status | pendente |

## O que faz

Adiciona a opção preferIPv6 ao ServersTransport. Com ela, o hostname do backend é resolvido para IPv6 primeiro e cai para IPv4 só se nenhum IPv6 for alcançável. Literais de IP e conexões com família já fixada não mudam. Vale para o transport dinâmico (file, labels, KV), o CRD e o default@internal. Closes #10677.

## Riscos e situação no upstream

Atinge 17 arquivos, com CRD gerado e applyconfiguration. Status 2-needs-review, sem review; autor diz que roda com patch no próprio setup. Aplica no dialer, então h2c explícito não é afetado. Útil em dual-stack com custo de egress IPv4.

## Principais arquivos

- `pkg/config/dynamic/http_config.go`
- `pkg/config/static/static_config.go`
- `pkg/provider/kubernetes/crd/generated/applyconfiguration/traefikio/v1alpha1/serverstransportspec.go`
- `docs/content/reference/routing-configuration/http/load-balancing/serverstransport.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13417/head:upstream-pr-13417`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
