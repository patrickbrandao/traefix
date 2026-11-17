# Keep accepting when TCP keep-alive cannot be set on a connection

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14010](https://github.com/traefik/traefik/pull/14010) |
| Tipo | Correção |
| Área | server/entrypoint-tcp |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | headw01 |
| Aberto em | 2026-10-05 (última atualização: 2026-11-06) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +108/-8 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/server |
| Status | importado com ajustes |

## O que faz

Quando o peer reseta a conexao entre accept() e a configuracao de keep-alive, o erro saia como erro do listener e o entrypoint TCP parava de aceitar conexoes de vez (macOS, illumos, FreeBSD). Agora a conexao e fechada e o listener segue aceitando.

## Riscos e situação no upstream

Pequeno (+108/-8). Duplicado funcional do #14094 (mesmo problema, mesmas issues #8841/#8071). Label status/design-review; dois reviews COMMENTED; autor cobrou andamento.

## Principais arquivos

- `pkg/server/server_entrypoint_tcp.go`
- `pkg/server/server_entrypoint_tcp_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14010/head:upstream-pr-14010`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, em "design-review". O review automatizado pediu um sinal visível (não só debug) quando as conexões são
descartadas.

### Achados no PR original

- **Bug real (DoS):** `tcpKeepAliveListener.Accept()` devolvia como erro do listener a falha de `SetKeepAlive`/
  `SetKeepAlivePeriod`, e o `TCPEntryPoint.Start` para o entrypoint em qualquer erro não temporário. Um único cliente que
  conecta e envia RST na hora derruba o entrypoint de vez em FreeBSD (`ECONNRESET`), illumos e macOS (`EINVAL`), enquanto
  o processo continua "saudável". Qualquer cliente remoto consegue fazer isso.
- O PR resolve fechando só aquela conexão e aceitando a próxima, com um aviso a cada minuto no máximo (contagem de
  conexões descartadas). Ponto fraco: usa uma variável global de pacote (`setTCPKeepAlive`) como gancho de teste e muda o
  listener para ponteiro.
- **Sobreposição com o #14094**, que corrige o mesmo bug com função injetável (`acceptTCPWithKeepAlive`), sem global, mas
  só com log de debug.

### Ajustes (combinação dos dois PRs)

- Estrutura do #14094 (`configureTCPKeepAlive` mais `acceptTCPWithKeepAlive` com dependências injetadas, sem variável
  global) e aviso com limite de taxa do #14010 (`rate.Sometimes`, um aviso por minuto com o número de conexões
  descartadas), num `keepAliveFailures` próprio de cada listener, com o logger do contexto do entrypoint.
- `tcpKeepAliveListener` continua sendo valor (compatível com o teste do #13237) e é construído por
  `newTCPKeepAliveListener`.
- `golang.org/x/time` já é dependência direta; `go.mod` não muda.
- Testes: falhas consecutivas fecham as conexões e o `Accept` devolve a próxima saudável; só o primeiro aviso é emitido
  (os seguintes ficam contados); um erro real do listener continua sendo devolvido.

### Testes

- `go test ./pkg/server/` ok; `golangci-lint run ./pkg/server/...`: 0 issues.
