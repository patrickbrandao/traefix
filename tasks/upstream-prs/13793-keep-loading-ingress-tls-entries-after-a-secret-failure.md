# Keep loading Ingress TLS entries after a secret failure

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13793](https://github.com/traefik/traefik/pull/13793) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | lazerg |
| Aberto em | 2026-08-27 (última atualização: 2026-11-15) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +48/-4 em 2 arquivos |
| Labels | status/1-needs-design-review, area/provider/k8s/ingress, kind/bug/fix, size/S |
| Status | importado com ajustes |

## O que faz

getCertificates parava no primeiro spec.tls[] cujo secret falhava, descartando os certificados de todos os domínios seguintes do mesmo Ingress. O patch junta os erros e continua o loop. Corrige #13791. O padrão já existe para TLSStore (#12522).

## Riscos e situação no upstream

Pequeno (48 linhas). Base v3.7, precisa de adaptação. Status status/1-needs-design-review; o autor cobrou atualização, e o maintainer ainda não aprovou. Mudança de comportamento no fluxo de erro do provider.

## Principais arquivos

- `pkg/provider/kubernetes/ingress/kubernetes.go`
- `pkg/provider/kubernetes/ingress/kubernetes_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13793/head:upstream-pr-13793`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- Bug real de disponibilidade no provider Kubernetes Ingress: em `getCertificates`, o primeiro `spec.tls[]` com Secret ausente ou inválida fazia a função retornar e **descartava os certificados válidos** das entradas seguintes do mesmo Ingress. Os hosts afetados caíam no certificado padrão.
- A correção junta os erros (`errors.Join`) e continua o loop. O chamador continua registrando o erro, e o comportamento das rotas não muda.
- Sem impacto de segurança: só são lidas Secrets do namespace do próprio Ingress, como antes.
- Escopo diferente de #13948, que faz a mesma correção no provider ingress-nginx; os dois não se sobrepõem.

**Ajustes**

- Teste adaptado ao lint `embedlit` (literais sem `ObjectMeta`).

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress/` ok.
- `golangci-lint run ./pkg/provider/kubernetes/ingress/...`: 0 issues.
