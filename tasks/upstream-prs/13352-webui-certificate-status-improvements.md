# Webui certificate status improvements

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13352](https://github.com/traefik/traefik/pull/13352) |
| Tipo | Recurso |
| Área | webui/certificates |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | jnoordsij |
| Aberto em | 2026-06-12 (última atualização: 2026-07-23) |
| Draft | sim |
| Branch-alvo | `v3.7` |
| Tamanho | +47/-30 em 7 arquivos |
| Labels | status/0-needs-triage, area/webui, size/S, area/middleware, area/authentication, area/provider/k8s/ingress-nginx |
| Status | pendente |

## O que faz

A badge de expiração no dashboard trata como 'expirando' qualquer certificado com menos de 30 dias restantes, fixo. O PR propõe um limiar relativo ao tempo de vida do certificado, para não gerar alarme falso com certificados de vida curta (Let's Encrypt de 7 dias, perfil de 45 dias).

## Riscos e situação no upstream

PR em draft. Um maintainer disse que vai aguardar suporte ARI (issue #12461), que daria o tempo real de renovação. Base v3.7, status 0-needs-triage. Mudança restrita à visualização, com um pequeno trecho em api.

## Principais arquivos

- `pkg/api/certificate.go`
- `pkg/api/handler_overview.go`
- `webui/src/components/certificates/CertExpiryBadge.tsx`
- `webui/src/components/certificates/CertificateDetails.tsx`
- `webui/src/pages/certificates/Certificates.tsx`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13352/head:upstream-pr-13352`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
