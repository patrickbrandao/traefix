# Apply server-alias to every Ingress of the host

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14124](https://github.com/traefik/traefik/pull/14124) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | amazon7737 |
| Aberto em | 2026-10-22 (última atualização: 2026-10-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +682/-15 em 6 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

A anotação server-alias passa a valer para todo o host, e não só para o Ingress que a declara, como faz o ingress-nginx. A propriedade segue a regra do ingress-nginx: o primeiro Ingress, por ordem de criação, que declara aliases para o host é o dono. Complementa o PR 14110.

## Riscos e situação no upstream

Depende do 14110 para o caso de Prefix envolvendo Exact. Muda o comportamento de aliases entre Ingresses do mesmo host. Status: aguardando review. Docs atualizadas.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/provider/kubernetes/ingress-nginx/model.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14124/head:upstream-pr-14124`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
