# Add support for leaf certificate only forwarding on PassTLSClientCert middleware

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13950](https://github.com/traefik/traefik/pull/13950) |
| Tipo | Recurso |
| Área | middleware/passtlsclientcert |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | vantrop |
| Aberto em | 2026-09-23 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +132/-46 em 9 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/middleware, area/tls |
| Status | importado com ajustes |

## O que faz

Adiciona a opção onlyLeaf ao middleware PassTLSClientCert, para encaminhar só o certificado folha quando o cliente manda uma cadeia. Hoje, com pem: true, a cadeia inteira vai concatenada em X-Forwarded-Tls-Client-Cert. Não altera o comportamento atual quando a opção está desligada.

## Riscos e situação no upstream

Toca CRD e docs gerados (traefik.io_middlewares.yaml, kubernetes-crd-definition-v1.yml), então pode exigir make generate-crd. Inclui também mudança no Makefile que não é do escopo do PR. Três reviews COMMENTED.

## Principais arquivos

- `pkg/middlewares/passtlsclientcert/pass_tls_client_cert.go`
- `pkg/middlewares/passtlsclientcert/pass_tls_client_cert_test.go`
- `pkg/config/dynamic/middlewares.go`
- `docs/content/reference/routing-configuration/http/middlewares/passtlsclientcert.md`
- `Makefile`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13950/head:upstream-pr-13950`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes (combina a proposta do #13685).**

Achados:
- Adiciona `onlyLeaf` (default `false`) ao PassTLSClientCert. Com a opção ligada, `pem` e `info` passam a usar
  apenas `PeerCertificates[0]`, que é o leaf, o único garantidamente verificado no handshake. O
  comportamento padrão não muda. A opção resolve o caso do Keycloak e de outros backends que esperam um
  único PEM e, de quebra, evita que um backend leia campos de um certificado anexado e forjado pelo cliente
  (problema descrito no #13685).
- O tipo reutiliza `dynamic.PassTLSClientCert`, então o CRD só precisa ganhar o campo novo no YAML gerado. Não há
  mudança em código Go gerado.
- O PR inclui uma mudança no `Makefile` (pull multi-plataforma da imagem do gateway-operator para os testes de
  conformidade). Não tem relação com a feature e mexe em build/CI do upstream: **não importado**.
- Os comentários no `ServeHTTP` só repetiam o que o código faz.

Ajustes:
- Removi os comentários "what" e reescrevi o comentário do campo.
- Documentação da opção reescrita, com um aviso explicando que os certificados extras enviados pelo cliente não
  são necessariamente verificados e recomendando `onlyLeaf: true` quando o backend identifica o cliente pelo
  header.
- CRDs regenerados com `make generate-crd`. O último passo do script (`sed -i` em sintaxe GNU) falha no macOS,
  então apliquei manualmente a remoção do `---` inicial. O resultado é idêntico ao do PR. Também rodei
  `go generate`, que não gerou outras mudanças.
- O `Makefile` do PR foi descartado.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/middlewares/passtlsclientcert/ ./pkg/config/label/ ./pkg/config/dynamic/ ./pkg/provider/kubernetes/crd/`: ok.
- `golangci-lint run` nos pacotes tocados: 0 issues.
