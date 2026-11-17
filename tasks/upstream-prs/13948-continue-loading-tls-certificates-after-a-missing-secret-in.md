# Continue loading TLS certificates after a missing Secret in the ingress-nginx provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13948](https://github.com/traefik/traefik/pull/13948) |
| Tipo | Correção |
| Área | provider/ingress-nginx-tls |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | MatteoManzoni |
| Aberto em | 2026-09-23 (última atualização: 2026-09-24) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +74/-4 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

Em loadCertificates, um Secret faltando ou sem par de chaves interrompia o carregamento dos demais spec.tls[] do mesmo Ingress, e os hosts seguintes caíam no certificado default. Agora os erros são acumulados com errors.Join e o carregamento continua. Muito relevante com cert-manager adicionando hosts novos.

## Riscos e situação no upstream

Correção localizada em um arquivo com teste. Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/provider/kubernetes/ingress-nginx/build_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13948/head:upstream-pr-13948`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
