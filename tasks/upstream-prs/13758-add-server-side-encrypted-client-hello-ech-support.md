# Add server-side Encrypted Client Hello (ECH) support

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13758](https://github.com/traefik/traefik/pull/13758) |
| Tipo | Recurso |
| Área | tls, provider/file, provider/kv, provider/kubernetes/crd |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | savely-krasovsky |
| Aberto em | 2026-08-24 (última atualização: 2026-11-10) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +2554/-42 em 40 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/tls |
| Status | pendente |

## O que faz

Adiciona ECH server-side (RFC 9849) em TCP e HTTP/3, com chaves ECH nas TLS options vindas de arquivo, KV ou Secret (chave tls.ech), e chaves decrypt-only para rotação. Usa crypto/hpke do Go 1.26, sem a dependência circl. Oculta SNI/ALPN de observadores on-path.

## Riscos e situação no upstream

2554 linhas em 40 arquivos, com mexida em TLS e providers. Substitui o #11866. Requer Go 1.26 (confirmar go.mod do fork). Status/2-needs-review, sem review de maintainer. Autor declara uso de Codex em testes e docs. Precisa de revisão de segurança antes de uso.

## Principais arquivos

- `cmd/ech/ech.go`
- `cmd/traefik/traefik.go`
- `integration/https_test.go`
- `pkg/provider/file/file.go`
- `pkg/provider/file/file_test.go`
- `pkg/provider/kubernetes/crd/generated/applyconfiguration/traefikio/v1alpha1/tlsoptionspec.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13758/head:upstream-pr-13758`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
