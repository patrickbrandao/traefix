# Make ACME email field optional

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13943](https://github.com/traefik/traefik/pull/13943) |
| Tipo | Recurso |
| Área | acme/config |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | jnoordsij |
| Aberto em | 2026-09-22 (última atualização: 2026-11-13) |
| Draft | não |
| Branch-alvo | `v3.8` |
| Tamanho | +60/-34 em 13 arquivos |
| Labels | kind/enhancement, area/documentation, status/2-needs-review, area/acme, size/S |
| Status | pendente |

## O que faz

Torna o campo email do resolver ACME opcional, já que o Let's Encrypt deixou de usá-lo e alguns provedores rejeitam registro que contenha e-mail. Ajusta provider e documentação para não exigir o campo.

## Riscos e situação no upstream

Mexe em config e docs de vários setups (Docker, K8s, Swarm). Base v3.8. Um reviewer aprovou; outro comentou pedindo cuidado com revogação. Status needs-review.

## Principais arquivos

- `pkg/provider/acme/provider.go`
- `pkg/provider/acme/account.go`
- `pkg/provider/acme/provider_test.go`
- `docs/content/reference/install-configuration/tls/certificate-resolvers/acme.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13943/head:upstream-pr-13943`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
