# Fix passTLSClientCert info leaking intermediate CA certificate data

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13685](https://github.com/traefik/traefik/pull/13685) |
| Tipo | Correção |
| Área | middleware/passtlsclientcert (mTLS) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | Nelwhix |
| Aberto em | 2026-08-12 (última atualização: 2026-08-13) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +64/-46 em 3 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware, area/tls |
| Status | pendente |

## O que faz

O header X-Forwarded-Tls-Client-Cert-Info passava a iterar sobre toda a cadeia do peer TLS, então um intermediário enviado pelo cliente aparecia como se fosse o próprio subject. Agora o header usa só o certificado leaf do cliente. A opção pem não é afetada. Resolve #12887.

## Riscos e situação no upstream

Muda o conteúdo do header para configs que já usavam certificados de cadeia. Status needs-review, sem review ainda. PR base v3.7.

## Principais arquivos

- `pkg/middlewares/passtlsclientcert/pass_tls_client_cert.go`
- `pkg/middlewares/passtlsclientcert/pass_tls_client_cert_test.go`
- `docs/content/reference/routing-configuration/http/middlewares/passtlsclientcert.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13685/head:upstream-pr-13685`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
