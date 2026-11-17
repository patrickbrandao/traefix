# Use EC256 as default key type for certificates requested via ACME providers

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13849](https://github.com/traefik/traefik/pull/13849) |
| Tipo | Recurso |
| Área | acme, tls |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | jnoordsij |
| Aberto em | 2026-09-07 (última atualização: 2026-10-30) |
| Draft | não |
| Branch-alvo | `v3.8` |
| Tamanho | +36/-36 em 9 arquivos |
| Labels | kind/enhancement, status/2-needs-review, area/acme, size/S, breaking |
| Status | pendente |

## O que faz

Troca o tipo de chave padrão do ACME de RSA4096 para EC256, reduzindo tamanho e custo do handshake. Os clientes atuais suportam bem ECDSA.

## Riscos e situação no upstream

Mudança de default marcada como breaking. Afeta só certificados novos: os antigos mantêm a chave existente, e o #13004 é pré-requisito para rotacionar (comentário de maintainer). Base v3.8. Status/2-needs-review, com dois reviews COMMENTED.

## Principais arquivos

- `integration/acme_test.go`
- `internal/gendoc.go`
- `pkg/provider/acme/account.go`
- `pkg/provider/acme/provider.go`
- `docs/content/https/ref-acme.toml`
- `docs/content/https/ref-acme.txt`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13849/head:upstream-pr-13849`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
