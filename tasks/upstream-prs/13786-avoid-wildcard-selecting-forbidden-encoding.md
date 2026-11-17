# Avoid wildcard selecting forbidden encoding

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13786](https://github.com/traefik/traefik/pull/13786) |
| Tipo | Correção |
| Área | middleware/compress |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | arpansahu |
| Aberto em | 2026-08-27 (última atualização: 2026-08-27) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +49/-3 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | pendente |

## O que faz

Quando Accept-Encoding tem wildcard e proíbe explicitamente (q=0) o encoding preferido, o compress ainda o seleciona (ex.: gzip;q=0, *;q=1 escolhe gzip). O patch rastreia os encodings proibidos e não usa o padrão se ele estiver proibido.

## Riscos e situação no upstream

Pequeno (49 linhas). Viola a RFC de content negotiation, mas o impacto prático é limitado. Base master, status/2-needs-review.

## Principais arquivos

- `pkg/middlewares/compress/acceptencoding.go`
- `pkg/middlewares/compress/acceptencoding_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13786/head:upstream-pr-13786`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
