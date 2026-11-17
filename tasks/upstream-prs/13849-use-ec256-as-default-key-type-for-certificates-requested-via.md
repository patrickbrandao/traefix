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
| Status | adiado |

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

## Revisão (2026-11-16)

**Veredito: adiado.**

Achados:
- O PR muda o default de `acme.keyType` de `RSA4096` para `EC256`, inclusive no fallback de valor inválido
  (`GetKeyType`). Tecnicamente está correto e não reduz a segurança: P-256 é o padrão recomendado e gera
  handshakes mais leves.
- **Mas muda o comportamento padrão de quem já está em produção.** No código atual, `renewCertificates` só
  reaproveita a chave existente quando o tipo dela é igual a `GetKeyType(p.KeyType)`. Com o novo default,
  todo resolver sem `keyType` explícito passaria a trocar silenciosamente as chaves RSA por ECDSA na próxima
  renovação. Clientes legados que só aceitam RSA, pinning de chave pública e backends ou appliances que
  inspecionam o certificado quebrariam sem nenhuma mudança de configuração.
- No upstream, só há reviews `COMMENTED` (um deles automatizado). O próprio revisor pede uma nota de migração,
  e não há decisão de maintainer sobre a versão em que essa mudança entraria.
- O PR também traz mudanças de whitespace sem relação com a feature em `acme.md` e `internal/gendoc.go`
  (drive-by), e muda a expectativa de vários testes de integração.

Motivo do adiamento: é uma mudança de default com efeito sobre certificados existentes, sem decisão de
maintainer e sem nota de migração. Quem quer ECDSA já pode configurar `keyType: EC256` hoje. Reavaliar
quando o upstream definir a versão-alvo (provavelmente uma major ou minor com nota de migração).

Testes: nenhum (não importado).
