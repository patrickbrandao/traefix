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
| Status | adiado |

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

## Revisão (2026-11-16)

**Veredito: adiado.**

Achados (PR original):
- **Bypass de autorização.** `ServeHTTP` percorre todos os `req.TLS.PeerCertificates` e libera a requisição se
  *qualquer* um casar com o filtro. Os certificados intermediários enviados pelo cliente não são verificados
  pelo Go: só a cadeia que leva do leaf a uma CA confiável é validada, e o resto do que o cliente manda é
  aceito sem checagem. Quem tem um certificado de cliente válido (com qualquer identidade) pode anexar um
  certificado autoassinado com `Subject: CN=admin` e passar pelo filtro. Pior: com `clientAuthType`
  `RequestClientCert` ou `RequireAnyClientCert`, nem o leaf é verificado, e o middleware vira um controle
  de acesso ilusório. Uma implementação correta tem de usar apenas `req.TLS.VerifiedChains` (leaf
  `VerifiedChains[i][0]`) e falhar fechado quando não houver cadeia verificada.
- **DN montado sem escape.** `buildDN` concatena os valores sem o escape da RFC 4514. Um valor com `,` ou `=`
  (por exemplo `O=x,CN=admin` dentro de um campo) produz uma string ambígua, que uma regex sem âncora pode
  casar indevidamente. Além disso, a ordem é C→CN (formato "oneline" antigo do OpenSSL), e não a RFC 2253 que o
  nginx usa desde a 1.11.6, apesar do que diz o comentário. Atributos como `serialNumber`, `emailAddress` e
  `DC` também ficam de fora.
- As regex não são ancoradas: `CN=foo` também casa com `CN=foobar`. Isso precisa estar bem documentado ou ser
  ancorado automaticamente.
- Não houve wiring no CRD `Middleware` do Kubernetes, então o recurso não funciona no provider CRD. A
  documentação está fora da estrutura atual (`reference/routing-configuration/...`).
- No upstream, o PR está em `status/1-needs-design-review`, sem nenhum review de maintainer. O formato do DN e
  o nome da opção são decisões de API ainda em aberto.

Motivo do adiamento: corrigir o bypass é simples, mas o formato do DN e a semântica do filtro são decisões de
API públicas. Se o fork escolher agora um formato diferente do que o upstream vier a adotar, a configuração dos
usuários vai quebrar no próximo sync. Reavaliar quando houver design review. Se for importado antes disso, os
requisitos mínimos são: usar só `VerifiedChains`, gerar o DN com escape RFC 4514 (`pkix.Name.String()`), fazer
o wiring no CRD e escrever testes de cadeia com intermediário forjado.

Testes: nenhum (não importado).
