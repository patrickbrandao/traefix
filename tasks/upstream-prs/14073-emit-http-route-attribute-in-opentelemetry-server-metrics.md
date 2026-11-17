# Emit http.route attribute in OpenTelemetry server metrics

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14073](https://github.com/traefik/traefik/pull/14073) |
| Tipo | Recurso |
| Área | middleware/observability, otel |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | PragalvaXFREZ |
| Aberto em | 2026-10-14 (última atualização: 2026-10-15) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +143/-3 em 6 arquivos |
| Labels | kind/bug/fix, size/M, area/middleware/metrics, area/otel |
| Status | pendente |

## O que faz

Adiciona o atributo http.route (regra do router que casou) a metrica http.server.request.duration do OpenTelemetry, permitindo quebrar a latencia por rota sem aumentar cardinalidade alem do numero de routers. Parcialmente resolve #12247.

## Riscos e situação no upstream

Medio (+143/-3, 6 arquivos). Usa um holder no contexto preenchido pelo router. Sem reviews de maintainer ainda. Decisao em aberto: usar a regra crua ou o path extraido.

## Principais arquivos

- `pkg/middlewares/observability/observability.go`
- `pkg/middlewares/observability/router.go`
- `pkg/middlewares/observability/semconv.go`
- `docs/content/reference/install-configuration/observability/metrics.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14073/head:upstream-pr-14073`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
