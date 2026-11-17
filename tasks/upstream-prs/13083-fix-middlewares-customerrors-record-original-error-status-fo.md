# fix(middlewares/customerrors): record original error status for metrics when serving custom error pages

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13083](https://github.com/traefik/traefik/pull/13083) |
| Tipo | Correção |
| Área | middlewares/errors, metrics |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | wucm667 |
| Aberto em | 2026-04-30 (última atualização: 2026-08-06) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +6/-5 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/middleware |
| Status | rejeitado |

## O que faz

Quando o middleware errors serve uma página customizada para um 5xx do backend, as métricas (traefik_service_requests_total) registravam code=200. A correção chama rw.WriteHeader(originalCode) antes de servir a página, então o alerta de 5xx volta a funcionar. Fecha o issue #13054.

## Riscos e situação no upstream

Diff pequeno (+6/-5) em custom_errors.go, com testes atualizados. Afeta alertas que dependem de métricas de erro. Maintainer confirmou que está no radar, sem revisão de código ainda.

## Principais arquivos

- `pkg/middlewares/customerrors/custom_errors.go`
- `pkg/middlewares/customerrors/custom_errors_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13083/head:upstream-pr-13083`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- A correção chama `rw.WriteHeader(originalCode)` logo após capturar o código, antes de aplicar `statusRewrites` e antes de servir a página de erro. Isso comita a resposta cedo demais:
  - `statusRewrites` deixa de funcionar (o status reescrito nunca chega ao cliente, porque o header já foi enviado);
  - os headers da página de erro (Content-Type, Cache-Control, etc.) definidos pelo backend de erro são perdidos, porque os headers já foram escritos;
  - o modo nginx, que por design preserva o status escolhido pelo backend de erro, passa a ser sobrescrito. O próprio PR muda os testes para aceitar esse novo comportamento.
- O problema real (métricas registrando o status da página de erro em vez do original) precisa ser resolvido na camada de captura/métricas (ex.: registrar o código original no `capture` ou no contexto), não forçando o status na resposta.
- Sem review de maintainer.

### Ajustes

Nenhum.

### Testes

Não aplicável.

### Motivo

Rejeitado: quebra `statusRewrites`, headers da página de erro e o modo nginx para corrigir só um sintoma de métrica.
