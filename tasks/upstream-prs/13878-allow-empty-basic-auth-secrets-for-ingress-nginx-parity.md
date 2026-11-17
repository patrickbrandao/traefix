# Allow empty basic-auth secrets for ingress-nginx parity

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13878](https://github.com/traefik/traefik/pull/13878) |
| Tipo | Correção |
| Área | middleware/basicauth, provider/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | harriteja |
| Aberto em | 2026-09-09 (última atualização: 2026-09-16) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +199/-11 em 5 arquivos |
| Labels | status/2-needs-review, size/L, area/middleware, area/authentication, area/provider/k8s, area/provider/k8s/ingress-nginx |
| Status | importado com ajustes |

## O que faz

Alinha o provider ingress-nginx com o comportamento do ingress-nginx quando o secret de auth não tem usuários: o Ingress deixa de ser pulado (que resultava em 404) e o BasicAuth com lista vazia passa a desafiar sempre com 401. A mudança segue o tratamento de DigestAuth.

## Riscos e situação no upstream

Muda comportamento de segurança: rota com secret vazio passa a responder 401 em vez de não existir. Maintainers pediram para verificar o ingress-nginx real antes. Status needs-review, sem review ainda; PR aberto contra master sem backport.

## Principais arquivos

- `pkg/middlewares/auth/basic_auth.go`
- `pkg/middlewares/auth/basic_auth_test.go`
- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13878/head:upstream-pr-13878`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

Veredito: **importado com ajustes**. A intenção do PR foi reimplementada à mão, porque o código atual (`maxPasswordLength`, `secretUser`, `URNRefs`) divergiu da base do PR.

Achados no PR original:
- PR aberto, sem review de maintainer. O comportamento novo é fail-closed: com a lista vazia, todo pedido recebe 401 (conferido em `checkPassword`: usuário desconhecido dá `secret == ""` e devolve `false`). Antes, o middleware falhava ao ser criado e o router não era servido (404). Não há bypass.
- O PR gerava um hash bcrypt em `init` de pacote (`mustEmptyUsersNotFoundSecret`, com `panic` em caso de erro) só para "igualar o tempo". Isso é custo e risco de panic no startup sem benefício: sem usuários não há existência de usuário a vazar. O Aikido ainda acusou um falso "segredo exposto" nesse trecho.
- O PR removia o newline final de `basic_auth.go` (gofmt).
- A mudança vale para todos os providers, não só para o ingress-nginx: um `usersFile` vazio passa a negar tudo em silêncio. Isso foi mitigado com um log `warn` e documentado.
- O fixture e o teste de ponta a ponta do PR não batem com a saída atual do provider, por isso foram substituídos por testes unitários.

Ajustes:
- `NewBasic` aceita a lista vazia, loga `No users configured: all requests are denied` e não calcula hash para usuários desconhecidos quando não há usuários. Nada de `init`, de bcrypt nem de `panic`.
- `basicAuthUsers` (ingress-nginx) aceita um Secret `auth-map` vazio. O `auth-file` sem usuários já devolvia lista vazia e agora também funciona.
- Testes: `TestNewBasicEmpty` (401 com e sem usuário, sem calcular hash), dois casos novos em `Test_basicAuthUsers` e `TestProvider_resolveBasicAuth_emptyAuthMap`.
- Docs: nota em `basicauth.md` e seção `BasicAuth Without Users` em `migrate/v3.md`.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/middlewares/auth/ ./pkg/provider/kubernetes/ingress-nginx/`: ok.
- `golangci-lint run ./pkg/middlewares/auth/... ./pkg/provider/kubernetes/ingress-nginx/...`: 0 issues.
