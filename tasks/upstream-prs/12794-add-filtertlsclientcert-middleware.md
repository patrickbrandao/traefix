# Add FilterTLSClientCert middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12794](https://github.com/traefik/traefik/pull/12794) |
| Tipo | Recurso |
| Área | tls/middleware |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | nerzhul |
| Aberto em | 2026-03-11 (última atualização: 2026-05-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +500/-19 em 7 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/S, area/tls |
| Status | pendente |

## O que faz

Novo middleware FilterTLSClientCert que aceita ou recusa certificados de cliente mTLS pelo Subject ou Issuer. É o equivalente ao auth-tls-match-cn do nginx ingress, com uso comum em mTLS+HTTP. Hoje o autor usa um plugin WASM para isso.

## Riscos e situação no upstream

Novo middleware com CRD e código gerado (7 arquivos, +500/-19). Status design-review, e o autor está cobrando andamento sem resposta.

## Principais arquivos

- `pkg/middlewares/filtertlsclientcert/filter_tls_client_cert.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/server/middleware/middlewares.go`
- `docs/content/middlewares/http/filtertlsclientcert.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12794/head:upstream-pr-12794`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
