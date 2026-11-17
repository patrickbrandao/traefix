# Skip renewal of ACME certificates not used by any router

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14259](https://github.com/traefik/traefik/pull/14259) |
| Tipo | Recurso |
| Área | tls/acme |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | alekc |
| Aberto em | 2026-11-09 (última atualização: 2026-11-09) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +1123/-27 em 12 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/acme, size/L, area/rules |
| Status | pendente |

## O que faz

O provider ACME deixa de renovar certificados que nenhum router TLS usa e os remove do store depois que expiram. A checagem de uso cobre hosts das regras, tls.domains e wildcards, e cai para 'tudo em uso' quando há regras que não listam hosts. Reduz pedidos ao Let's Encrypt e limites de rate limit.

## Riscos e situação no upstream

Grande (+1123/-27, 12 arquivos) e com lógica complexa de correspondência de hosts e locks; toca em muxer/http, muxer/tcp, rules/parser e cmd/traefik. Risco de remover certificado em uso se a checagem falhar. Review: COMMENTED, sem aprovação de maintainer.

## Principais arquivos

- `pkg/provider/acme/provider.go`
- `pkg/provider/acme/provider_test.go`
- `pkg/rules/parser.go`
- `pkg/muxer/http/mux.go`
- `pkg/muxer/tcp/mux.go`
- `cmd/traefik/traefik.go`
- `docs/content/reference/install-configuration/tls/certificate-resolvers/acme.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14259/head:upstream-pr-14259`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
