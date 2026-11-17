# Add allowCORSPreflight to BasicAuth middleware for CORS preflight

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14237](https://github.com/traefik/traefik/pull/14237) |
| Tipo | Recurso |
| Área | middleware/basicauth |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | k3a |
| Aberto em | 2026-11-06 (última atualização: 2026-11-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +121/-22 em 7 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/authentication |
| Status | importado com ajustes |

## O que faz

Adiciona a opção allowCORSPreflight ao BasicAuth, deixando passar sem credenciais os OPTIONS que têm Access-Control-Request-Method. Resolve o caso em que o preflight CORS de fetch com credenciais falha antes de chegar ao backend (issue #1019, antiga).

## Riscos e situação no upstream

Afeta a segurança: um OPTIONS sem autenticação passa pelo middleware quando o header de preflight existe, então a opção deve ser opt-in e revisada. Toca em config dinâmica e no provider CRD (kubernetes.go, middleware.go). Review: COMMENTED, sem aprovação.

## Principais arquivos

- `pkg/middlewares/auth/basic_auth.go`
- `pkg/middlewares/auth/basic_auth_test.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/middleware.go`
- `pkg/provider/kubernetes/crd/kubernetes.go`
- `docs/content/reference/routing-configuration/http/middlewares/basicauth.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14237/head:upstream-pr-14237`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

Veredito: **importado com ajustes**. A opção continua opt-in (`false` por padrão).

Achados no PR original (segurança):
- **Spoofing de identidade via `headerField`.** No caminho liberado, um header enviado pelo cliente com o nome de `headerField` (ex.: `X-WebAuth-User: admin`) chegava ao backend, que o trata como usuário autenticado. No caminho normal o middleware sobrescreve esse header, mas no bypass não. Com `allowCORSPreflight` e `headerField` juntos, qualquer cliente podia mandar um `OPTIONS` "autenticado" como qualquer usuário.
- O critério de preflight era fraco: bastava `OPTIONS` com `Access-Control-Request-Method`. Um preflight real sempre tem `Origin`.
- O bypass é por natureza não autenticado: um backend que responde `OPTIONS` com conteúdo protegido (WebDAV, frameworks que roteiam OPTIONS para o mesmo handler) fica exposto. Esse risco é inerente à opção e foi documentado.
- O PR já tinha corrigido o vazamento de `Authorization` com `removeHeader`, apontado pelo review automático.
- O PR não incluía o código gerado (CRDs, applyconfiguration) nem as referências `file.toml`/`file.yaml`/`docker-labels.yml`.

Ajustes:
- O bypass exige `Origin` e `Access-Control-Request-Method` não vazios.
- O header `headerField` é removido da requisição liberada. Novo teste `TestBasicAuthAllowCORSPreflightHeaderField`.
- Testes do PR atualizados (caso novo "missing origin").
- Docs: seção `### allowCORSPreflight` com aviso de segurança; comentários das structs atualizados.
- `make generate-crd` e `go generate` rodados; `file.toml`, `file.yaml` e `docker-labels.yml` atualizados à mão (não são gerados).

Testes:
- `go build ./...`: ok.
- `go test ./pkg/middlewares/auth/ ./pkg/provider/kubernetes/crd/ ./pkg/config/...`: ok.
- `golangci-lint run ./pkg/middlewares/auth/... ./pkg/provider/kubernetes/crd/... ./pkg/config/...`: 0 issues.
