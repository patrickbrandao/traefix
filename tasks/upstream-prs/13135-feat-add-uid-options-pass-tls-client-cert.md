# feat: add uid options pass tls client cert

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13135](https://github.com/traefik/traefik/pull/13135) |
| Tipo | Recurso |
| Área | middlewares/passtlsclientcert |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | n0ta26 |
| Aberto em | 2026-05-09 (última atualização: 2026-07-27) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +68/-3 em 13 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/middleware |
| Status | pendente |

## O que faz

Adiciona a opção info.subject.uid ao passTLSClientCert, que inclui o atributo UID (OID 0.9.2342.19200300.100.1.1) do Subject DN no header X-Forwarded-Tls-Client-Cert-Info. Evita precisar de pem: true e parsear o certificado completo no backend.

## Riscos e situação no upstream

Sem revisão de código; maintainer respondeu que está no radar, sem ETA. Muda CRD gerado e docs, então precisa rodar a geração de CRD.

## Principais arquivos

- `pkg/middlewares/passtlsclientcert/pass_tls_client_cert.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/config/label/label_test.go`
- `docs/content/reference/routing-configuration/http/middlewares/passtlsclientcert.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13135/head:upstream-pr-13135`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
