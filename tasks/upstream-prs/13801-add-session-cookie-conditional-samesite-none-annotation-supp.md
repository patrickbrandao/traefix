# Add session-cookie-conditional-samesite-none annotation support

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13801](https://github.com/traefik/traefik/pull/13801) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | ArchitDewan |
| Aberto em | 2026-08-28 (última atualização: 2026-09-24) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +388/-132 em 15 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/L, area/provider/k8s/ingress-nginx |
| Status | importado com ajustes |

## O que faz

Implementa a annotation nginx.ingress.kubernetes.io/session-cookie-conditional-samesite-none. Com session-cookie-samesite: none, o SameSite=None é omitido para User-Agents incompatíveis (Chromium antigo, iOS 12, Safari do macOS 10.14, UC Browser antigo). Útil para quem migra do ingress-nginx.

## Riscos e situação no upstream

388 linhas em 15 arquivos, com detecção de User-Agent (heurística que envelhece). Relacionado ao #12631. Base master, status/2-needs-review. Inclui documentação.

## Principais arquivos

- `pkg/config/dynamic/http_config.go`
- `pkg/provider/kubernetes/ingress-nginx/annotations.go`
- `pkg/provider/kubernetes/ingress-nginx/annotations_test.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `pkg/provider/kubernetes/ingress-nginx/translator.go`
- `pkg/server/service/loadbalancer/leasttime/leasttime.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13801/head:upstream-pr-13801`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- Implementa `nginx.ingress.kubernetes.io/session-cookie-conditional-samesite-none`. Com `SameSite=None`, o atributo é omitido para user agents que sabidamente tratam `None` errado: iOS 12, Safari/WebView no macOS 10.14, Chrome/Chromium 51–66 e UC Browser abaixo de 12.13.2. É a mesma lista do Chromium usada pelo ingress-nginx. Sem isso, esses clientes descartam o cookie de afinidade ou o tratam como `Strict`.
- O campo `dynamic.Cookie.ConditionalSameSiteNone` é interno (`json:"-"`, `label:"-"`, `file:"-"`, `kv:"-"`), seguindo o precedente de `Cookie.Expires`: não aparece na configuração pública e só o provider ingress-nginx o liga.
- Custo e DoS: as regex são do pacote `regexp` do Go (RE2, tempo linear), rodam só quando a opção está ligada **e** `SameSite=None`, e só sobre o `User-Agent`, que já é limitado pelo tamanho máximo de headers. Não há risco de ReDoS.
- `WriteStickyCookie` ganhou o parâmetro `req`. Todos os chamadores do código atual (wrr, p2c, leasttime) foram atualizados; o hrw não escreve cookie.
- `http.SameSiteDefaultMode` faz o Go omitir o atributo (desde o Go 1.16), que é o comportamento esperado.

**Ajustes**

- As expectativas de `TestLoadIngresses` foram resolvidas manualmente (os hunks do PR conflitavam com o formato atual): `ConditionalSameSiteNone: true` foi adicionado aos quatro cookies das fixtures `ingress-with-sticky` e `ingresses-with-canary-and-sticky`.

**Testes**

- `go build ./...` ok.
- `go test ./pkg/server/service/loadbalancer/... ./pkg/provider/kubernetes/ingress-nginx/ ./pkg/config/dynamic/` ok.
- `golangci-lint` nos pacotes tocados: 0 issues.
