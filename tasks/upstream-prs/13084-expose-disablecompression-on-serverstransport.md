# Expose DisableCompression on ServersTransport

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13084](https://github.com/traefik/traefik/pull/13084) |
| Tipo | Recurso |
| Área | provider/kubernetes/crd, serverstransport |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | rupertgti |
| Aberto em | 2026-04-30 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +119/-0 em 15 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/provider/k8s/crd, area/provider/k8s |
| Status | importado com ajustes |

## O que faz

Adiciona o campo disableCompression ao ServersTransport (CRD, file provider e labels), que mapeia para http.Transport.DisableCompression. Com ele, Traefik para de pedir gzip ao backend e repassa Content-Encoding e o corpo como vieram. O padrão continua false.

## Riscos e situação no upstream

Label status/2-needs-review; autor corrigiu os checks e pediu nova revisão. Toca 15 arquivos, incluindo CRD e applyconfiguration gerados, que exigem make generate e make generate-crd.

## Principais arquivos

- `pkg/config/dynamic/http_config.go`
- `pkg/provider/kubernetes/crd/kubernetes.go`
- `pkg/provider/kubernetes/crd/generated/applyconfiguration/traefikio/v1alpha1/serverstransportspec.go`
- `integration/fixtures/k8s/01-traefik-crd.yml`
- `docs/content/reference/routing-configuration/http/load-balancing/serverstransport.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13084/head:upstream-pr-13084`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- Mudança simples e segura: `disableCompression` (padrão false) é repassado a `http.Transport.DisableCompression`. Como o smart round tripper e o transporte HTTP/2 nativo são clones do mesmo `http.Transport`, a opção vale para HTTP/1.1, HTTP/2 e h2c. O FastProxy não pede gzip ao backend por conta própria, então já se comporta como `disableCompression: true`.
- Sem risco de segurança: não muda TLS nem validação; só deixa de pedir gzip transparente e repassa `Content-Encoding`/corpo do backend como vieram.
- O PR editava à mão arquivos gerados (applyconfiguration, CRDs YAML) e páginas de docs que não existem no fork (`routing-configuration/other-providers/file.*`). Teste com mensagens nos asserts.
- Conflitos com o fork (`PeerCertSANs`, testes de Kerberos).

Ajustes:
- Código gerado e CRDs regenerados com `make generate-crd`; docs de referência com `make generate`.
- Conflitos de docs resolvidos à mão: linha na tabela do CRD ServersTransport (sem a âncora duplicada `-2` do PR) e na página de ServersTransport HTTP.
- `TestDisableCompression` integrado ao arquivo atual, sem mensagens nos asserts.

Testes: `go build ./...` ok; `go test ./pkg/server/service/ ./pkg/config/dynamic/ ./pkg/provider/kubernetes/crd/ ./pkg/redactor/ ./pkg/provider/kv/ ./pkg/provider/file/` ok; `golangci-lint run` nos pacotes tocados: 0 issues.
