# Do not flush a retry attempt that is being swallowed

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13874](https://github.com/traefik/traefik/pull/13874) |
| Tipo | Correção |
| Área | middleware/retry |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | datrixlab |
| Aberto em | 2026-09-09 (última atualização: 2026-09-14) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +120/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/middleware |
| Status | importado com ajustes |

## O que faz

Corrige o Flush do responseWriter do retry, que não verifica shouldNotWrite nem written. Flushar uma tentativa descartada faz o net/http enviar um 200 implícito ao cliente enquanto o loop de retry ainda roda, o que impede o failover. Alinha com o padrão já usado em customerrors.

## Riscos e situação no upstream

Mudança de uma função de guarda com teste de regressão. Status needs-review, sem review de maintainer.

## Principais arquivos

- `pkg/middlewares/retry/retry.go`
- `pkg/middlewares/retry/retry_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13874/head:upstream-pr-13874`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.** Só os testes foram importados, porque o fix já está no código.

Análise:
- O bug procede no código do PR: o `Flush` do `responseWriter` do retry não respeitava `shouldNotWrite`. Uma tentativa descartada (ex.: `503` a ser reenviado) que fazia flush levava o `net/http` a mandar um `200` implícito ao cliente. A segunda tentativa então escrevia num response já enviado.
- O código local (vindo do upstream no commit 98720726b, "Fix broken flush, hijack, and deadlines in wrapped ResponseWriters") **já tem a correção**: o `Flush` virou `FlushError` com `http.ResponseController` e retorna sem flush quando `shouldNotWrite` está ativo. O patch do PR nem se aplica, porque mira o `Flush()` antigo, que fazia asserção de `http.Flusher` (proibido pelo AGENTS.md).
- Um maintainer pediu rebase sobre `v3.7`. A review automática não fez apontamentos.
- Nenhum risco de segurança.

Ajustes:
- Os dois testes de regressão do PR foram portados e adaptados ao AGENTS.md: flush via `http.NewResponseController(rw).Flush()` em vez de `rw.(http.Flusher)`, `maps.Copy`, `require` fora da goroutine do handler e mensagens de assert mínimas.
- O teste de streaming foi reescrito para ser determinístico: o handler só escreve o segundo pedaço depois que o cliente leu o primeiro, com timeout no cliente. Na versão original, o teste passava mesmo sem o flush chegar ao cliente.
- Verificado que `TestRetryDoesNotFlushASwallowedAttempt` falha quando a guarda `shouldNotWrite` é removida de `FlushError`, e passa com ela.

Testes:
- `go test -race -count=1 ./pkg/middlewares/retry/` ok.
- `golangci-lint run ./pkg/middlewares/retry/...`: 0 issues.
