# fix: instantiate a fresh WASI host per request for wasm middlewares

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13617](https://github.com/traefik/traefik/pull/13617) |
| Tipo | Correção |
| Área | plugins/wasm (WASI) |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | gterminator |
| Aberto em | 2026-07-31 (última atualização: 2026-08-03) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +261/-24 em 10 arquivos |
| Labels | status/1-needs-design-review, size/L, area/plugins |
| Status | adiado |

## O que faz

Um único wasi.System (dono dos file descriptors reais) era compartilhado por todas as requisições de um middleware wasm. Com concorrência, uma requisição podia ler bytes de outra socket ou colidir em fd (Bad file number), o que explica o sintoma NOT_LEADER_FOR_PARTITION da issue #11629. O fix cria um WASI por requisição.

## Riscos e situação no upstream

Vazamento de dados entre requisições, o que dá peso alto. Mexe em go.mod e num fixture com plugin.wasm binário. Status needs-design-review, com maintainers pedindo análise de impacto. Sem review ainda.

## Principais arquivos

- `pkg/plugins/wasip.go`
- `pkg/plugins/wasip_mock.go`
- `pkg/plugins/middlewarewasm.go`
- `pkg/plugins/middlewarewasm_test.go`
- `pkg/plugins/middlewarewasm_socket_test.go`
- `go.mod`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13617/head:upstream-pr-13617`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, em "design-review" com maintainer (juliens), sem conclusão.

### Achados no PR original

- **Muda a semântica dos plugins WASM com sockets:** as instâncias guest são reaproveitadas num pool pelo
  http-wasm-host-go e guardam na memória os números de fd que abriram. Com um `wasi.System` novo por request,
  fechado ao fim dela, um plugin que mantém conexões entre requests (o caso da issue #11629: cliente Kafka) ou que
  abre sockets na inicialização (o contexto de setup é fechado logo depois do `NewMiddleware`) passa a usar fds
  inválidos ou de outro System. O bug de concorrência sai, mas troca-se por quebra funcional nesses plugins.
- **Custo e DoS por request:** `Builder.Instantiate` recompila e instancia o host module wazergo
  (`wazergo.MustInstantiate`, que entra em pânico se falhar), duplica os fds de stdio e reabre os diretórios montados a
  cada request. Sob carga, isso multiplica syscalls e fds abertos.
- **Binário de terceiros:** o teste depende de um `plugin.wasm` pré-compilado incluído no PR (com `go.mod` próprio
  apontando para `github.com/stealthrocket/net`), que não dá para auditar nem reproduzir aqui. Pelas regras desta
  importação ele não pode ser executado.
- `go.mod`: `wazergo` passa a indireto (efeito colateral, não é problema).

### Motivo do adiamento

A correção certa é vincular um `wasi.System` a cada instância guest do pool, e não a cada request. Isso precisa de um
gancho no http-wasm-host-go para o ciclo de vida da instância. A abordagem ainda está em design-review no upstream.
Vale reavaliar quando houver decisão do maintainer ou um fixture que possa ser reconstruído a partir do código-fonte.

### Testes

Nenhum. O PR não foi aplicado.
