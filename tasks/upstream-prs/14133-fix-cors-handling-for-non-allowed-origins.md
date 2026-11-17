# Fix CORS handling for non-allowed origins

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14133](https://github.com/traefik/traefik/pull/14133) |
| Tipo | Correção |
| Área | middleware/cors, provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | KaanSimsek |
| Aberto em | 2026-10-23 (última atualização: 2026-11-06) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +204/-10 em 6 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/M, area/middleware, area/provider/k8s/ingress-nginx |
| Status | importado com ajustes |

## O que faz

No provider ingress-nginx, o middleware CORS passa a responder só às origens que batem com cors-allow-origin, como o ingress-nginx. Preflights de outras origens seguem para o backend, e Access-Control-Allow-Credentials e Expose-Headers só saem para origens permitidas. Um flag interno ativado só pelo provider; CORS de file provider e CRD não muda.

## Riscos e situação no upstream

Flag interno, sem impacto nos demais providers. Status: needs-design-review, então pode haver mudança de direção. Preflight continua 200 em vez de 204 do ingress-nginx. Toca middleware de headers/config dinâmica.

## Principais arquivos

- `pkg/middlewares/headers/header.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/provider/kubernetes/ingress-nginx/middleware.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14133/head:upstream-pr-14133`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- No provider ingress-nginx, o middleware `Headers` gerado para `enable-cors` respondia a qualquer preflight (`OPTIONS` com `Origin` e `Access-Control-Request-Method`) **sem repassar ao backend**, mesmo quando a origem não estava em `cors-allow-origin`. Também mandava `Access-Control-Allow-Credentials: true` e `Access-Control-Expose-Headers` para todas as origens. O ingress-nginx só trata CORS quando a origem casa; nos outros casos, repassa a requisição ao backend sem headers CORS.
- Segurança: não é uma vulnerabilidade direta, porque sem `Access-Control-Allow-Origin` o navegador bloqueia a leitura. Mesmo assim, reduz informação vazada (credenciais e headers expostos para origens não permitidas) e devolve ao backend o tratamento de `OPTIONS` de origens desconhecidas.
- O PR implementa isso com um campo interno `CORSStrict` em `dynamic.Headers` (`json:"-"`, sem exposição na configuração pública, como o precedente `Cookie.Expires`), que só o provider ingress-nginx liga. O middleware `Headers` genérico não muda de comportamento.
- Ainda em "design-review" no upstream, sem objeção.

**Ajustes**

- Nenhum ajuste funcional. Acrescentei `CORSStrict: true` nas três expectativas de CORS de `TestLoadIngresses` que o PR não cobria (casos criados depois da base do PR).

**Testes**

- `go build ./...` ok.
- `go test ./pkg/middlewares/headers/ ./pkg/provider/kubernetes/ingress-nginx/` ok.
- `golangci-lint` nos pacotes tocados: 0 issues.

## Integração (2026-11-17)

Na integração com o #13508 (g03), a resposta passou a nunca enviar `Access-Control-Allow-Credentials` nem `Expose-Headers` para origens não permitidas, com ou sem `CORSStrict`. No preflight, `CORSStrict` decide se uma origem não permitida é repassada ao backend (strict) ou respondida sem headers CORS. Os casos de teste "sem strict mode" foram atualizados.
