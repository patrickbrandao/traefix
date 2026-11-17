# Set old certificate's tls_certs_not_after metric to 0.

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11659](https://github.com/traefik/traefik/pull/11659) |
| Tipo | Correção |
| Área | observability/metrics, tls |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | medio |
| Autor | xjewer |
| Aberto em | 2025-03-31 (última atualização: 2025-06-10) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +322/-39 em 4 arquivos |
| Labels | status/1-needs-design-review, size/L, area/middleware/metrics, contributor/waiting-for-feedback, area/tls |
| Status | pendente |

## O que faz

A métrica tls_certs_not_after mantém séries de certificados já substituídos, porque go-kit não remove label sets. O PR zera as séries de certificados obsoletos quando o watcher TLS recebe nova configuração.

## Riscos e situação no upstream

Altera o watcher TLS (4 arquivos, +322/-39). Label contributor/waiting-for-feedback e status/1-needs-design-review; sem reviews. Comportamento de métrica precisa de validação.

## Principais arquivos

- `cmd/traefik/traefik.go`
- `pkg/tls/certificate.go`
- `pkg/tls/tlsmanager_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11659/head:upstream-pr-11659`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
