# Fix X-Forwarded-For append handling

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13226](https://github.com/traefik/traefik/pull/13226) |
| Tipo | Correção |
| Área | middlewares/forwardedheaders, proxy |
| Prioridade | **P3** |
| Valor | media |
| Esforço de port | alto |
| Autor | choyri |
| Aberto em | 2026-05-22 (última atualização: 2026-06-06) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +80/-197 em 8 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/M, area/middleware, breaking |
| Status | pendente |

## O que faz

Centraliza o append do X-Forwarded-For no middleware forwardedheaders, controlado por forwardedHeaders.notAppendXForwardedFor, e remove a opção global notAppendXForwardedFor, que não estava ligada ao proxy. Resolve o issue #13185, em que middlewares viam só o XFF antigo.

## Riscos e situação no upstream

Marcado como breaking: remove uma opção de configuração estática e muda o XFF visto pelos middlewares. Conflita diretamente com commits recentes do fork (193af416c e fd632f41f) que mexem em forwardedheaders e nos proxies. Label needs-design-review.

## Atenção: alterações locais do Traefix

Mexe no mesmo código de X-Forwarded-For alterado nos commits locais 193af416c e fd632f41f. Revise os conflitos com a implementação local.

## Principais arquivos

- `pkg/middlewares/forwardedheaders/forwarded_header.go`
- `pkg/proxy/fast/proxy.go`
- `pkg/proxy/httputil/proxy.go`
- `pkg/config/static/static_config.go`
- `docs/content/reference/install-configuration/configuration-options.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13226/head:upstream-pr-13226`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
