# Resolve signing intermediate issuer for OCSP stapling entries

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14213](https://github.com/traefik/traefik/pull/14213) |
| Tipo | Correção |
| Área | tls/ocsp |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | AdamMagued |
| Aberto em | 2026-11-03 (última atualização: 2026-11-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +386/-13 em 4 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/L, area/tls |
| Status | pendente |

## O que faz

Para o stapling de OCSP, o emissor é hoje assumido como cert.Certificate[1] ou cai no leaf. O PR busca o emissor real por DN, SubjectKeyId/AuthorityKeyId e verificação criptográfica (CheckSignatureFrom), e também cobre responders delegados. Atualiza o emissor de entradas em cache.

## Riscos e situação no upstream

Toca o gerenciador TLS e o cache OCSP. Status: needs-design-review. Tem testes unitários e de integração; é bom para cadeias fora de ordem.

## Principais arquivos

- `pkg/tls/ocsp.go`
- `pkg/tls/ocsp_test.go`
- `pkg/tls/tlsmanager.go`
- `pkg/tls/tlsmanager_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14213/head:upstream-pr-14213`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
