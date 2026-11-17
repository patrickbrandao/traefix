# Add CRL-based client certificate revocation support for mTLS

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14014](https://github.com/traefik/traefik/pull/14014) |
| Tipo | Recurso |
| Área | tls/mtls, provider/kubernetes-crd |
| Prioridade | **P2** |
| Valor | alta |
| Esforço de port | alto |
| Autor | vantrop |
| Aberto em | 2026-10-06 (última atualização: 2026-10-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +5186/-31 em 42 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/tls, area/provider/k8s/crd |
| Status | pendente |

## O que faz

Adiciona o tipo de autenticacao RequireAndVerifyClientCertWithExpiry, que checa revogacao via CRL de cada certificado da cadeia do cliente. As CRLs vem de um store global em arquivo (tls.crlFiles) ou da CDP do certificado, com configuracao por TLSOption.

## Riscos e situação no upstream

Muito grande (+5186/-31, 42 arquivos, inclui CRD gerado e fsnotify). Label status/2-needs-review; seis reviews COMMENTED e nenhum aprovado. Pontos de seguranca: CRLs do store global sao confiadas implicitamente, sem verificar assinatura do emissor nem thisUpdate/nextUpdate. Exige revisao cuidadosa antes de portar.

## Principais arquivos

- `cmd/traefik/traefik.go`
- `pkg/config/static/static_config.go`
- `pkg/api/handler_certificate_test.go`
- `pkg/provider/kubernetes/crd/generated/applyconfiguration/traefikio/v1alpha1/clientauth.go`
- `docs/content/reference/install-configuration/tls/crl.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14014/head:upstream-pr-14014`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
