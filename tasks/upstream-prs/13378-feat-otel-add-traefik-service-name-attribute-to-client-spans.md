# feat(otel): add traefik.service.name attribute to client spans

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13378](https://github.com/traefik/traefik/pull/13378) |
| Tipo | Recurso |
| Área | middleware/tracing |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | sinashahoveisi |
| Aberto em | 2026-06-18 (última atualização: 2026-10-29) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +21/-0 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/middleware, area/middleware/tracing, area/otel |
| Status | pendente |

## O que faz

Adiciona o atributo traefik.service.name ao span client do ReverseProxy. O nome do serviço passa pelo contexto desde serviceTracing até o RoundTripper. Sem isso, o APM desenha um nó numérico por IP de pod, e com o atributo ele agrupa por serviço lógico. Fixes #13365.

## Riscos e situação no upstream

Escopo pequeno (2 arquivos, +21 linhas). Sem review. Testes e docs marcados como pendentes. Reusa o nome de atributo já usado no span Service.

## Principais arquivos

- `pkg/middlewares/observability/service.go`
- `pkg/proxy/httputil/observability.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13378/head:upstream-pr-13378`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
