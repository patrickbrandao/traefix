# Fix ACME Challenge for IP addresses

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13388](https://github.com/traefik/traefik/pull/13388) |
| Tipo | Correção |
| Área | acme/tls |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | medio |
| Autor | AkinoKaede |
| Aberto em | 2026-06-20 (última atualização: 2026-10-21) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +407/-2 em 6 arquivos |
| Labels | status/1-needs-design-review, area/acme, kind/bug/fix, size/M, area/tls |
| Status | pendente |

## O que faz

Corrige a validação ACME de IPs. HTTP-01 passa a remover os colchetes do Host IPv6 sem porta antes de procurar o token. TLS-ALPN-01 passa a casar o SNI de reverse-address (in-addr.arpa / ip6.arpa) com certificados que têm IP SAN, sem mudar matchDomain do TLS comum.

## Riscos e situação no upstream

PR base v3.7 (portar para master). Reviews: CHANGES_REQUESTED seguido de APPROVED. Autor testou com Pebble nas combinações IPv4/IPv6. Mexe no certificate store, com política fixa por store. Útil só para quem emite certificado para IP.

## Principais arquivos

- `pkg/provider/acme/challenge_http.go`
- `pkg/provider/acme/challenge_http_test.go`
- `pkg/tls/certificate_store.go`
- `pkg/tls/certificate_store_test.go`
- `pkg/tls/tlsmanager.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13388/head:upstream-pr-13388`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
