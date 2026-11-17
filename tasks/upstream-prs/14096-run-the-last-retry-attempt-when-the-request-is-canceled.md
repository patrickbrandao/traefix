# Run the last retry attempt when the request is canceled

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14096](https://github.com/traefik/traefik/pull/14096) |
| Tipo | Correção |
| Área | middleware/retry |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | lazerg |
| Aberto em | 2026-10-20 (última atualização: 2026-10-20) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +24/-0 em 2 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/S, area/middleware |
| Status | importado |

## O que faz

Quando o backoff do retry para porque o contexto foi cancelado, a ultima tentativa ainda roda e escreve a resposta (499). Sem isso, com buffering na frente, o oxy chamava WriteHeader(0) e causava panic em HTTP/2. Resolve #14095.

## Riscos e situação no upstream

Pequeno (+24, 2 arquivos). Label status/1-needs-design-review; sem reviews de maintainer. Reproduzido pelo autor com a configuracao da issue.

## Principais arquivos

- `pkg/middlewares/retry/retry.go`
- `pkg/middlewares/retry/retry_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14096/head:upstream-pr-14096`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado.**

Análise:
- O bug procede (#14095).
  - Com o contexto da requisição já cancelado, a primeira tentativa falha com erro de rede, e o writer do retry descarta o erro.
  - Em seguida, `backoff.WithContext` para antes da última tentativa, que é a única que escreve no writer real.
  - O middleware retornava sem escrever nada. Com `buffering` na frente, o oxy chamava `WriteHeader(0)`, que dá **panic em HTTP/2**. Sem `buffering`, o cliente recebia um `200` vazio implícito.
- O fix roda a última tentativa (`attempts = r.attempts; operation()`) quando `RetryNotify` termina com erro. Esse erro só ocorre quando todas as tentativas anteriores foram descartadas, então ainda não há resposta, e a última tentativa, que escreve direto no writer real, garante que alguma seja enviada. Com o contexto cancelado, o proxy responde `499` sem contatar o backend.
- Isso também cobre o fim do backoff por `MaxElapsedTime`. Nesse caso, a tentativa extra é exatamente a última que deveria ter acontecido, e o número máximo de tentativas configurado não é ultrapassado.
- Segurança: fecha um caminho de panic disparável pelo cliente (cancelar a requisição com `buffering` + `retry` em HTTP/2). Não abre nada novo.
- Upstream: em `status/1-needs-design-review`, sem objeção.

Ajustes: nenhum. O patch aplicou limpo.

Testes:
- `go test -race -count=1 ./pkg/middlewares/retry/` ok.
- Verificado que `TestRetryCanceledContext` falha sem o fix e passa com ele.
- `golangci-lint run ./pkg/middlewares/retry/...`: 0 issues.
- `go build ./...` ok.
