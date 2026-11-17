# feat: add global TCP access logging

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11998](https://github.com/traefik/traefik/pull/11998) |
| Tipo | Recurso |
| Área | middleware/accesslogs, entrypoints |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | Kronusus |
| Aberto em | 2025-08-13 (última atualização: 2026-04-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +294/-19 em 9 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/middleware, area/accesslogs |
| Status | rejeitado |

## O que faz

Adiciona access log de conexões TCP, registrando início e fim com endereços, bytes, duração, certificado de cliente e erro, habilitado pelo accessLog global. Suporta formato JSON e texto.

## Riscos e situação no upstream

Mexe em logs e entrypoints (9 arquivos, +294/-19). Autor marcou testes não feitos e descreve como 'early draft'. Label needs-design-review, sem reviews.

## Principais arquivos

- `cmd/traefik/tcp_accesslog.go`
- `pkg/middlewares/accesslogtcp/logger.go`
- `pkg/tcp/accesslog.go`
- `pkg/server/server_entrypoint_tcp.go`
- `pkg/server/routerfactory.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11998/head:upstream-pr-11998`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- Usa `github.com/sirupsen/logrus` para escrever os logs, o que o AGENTS.md proíbe (o projeto usa só zerolog) e reintroduz uma dependência direta que o access log HTTP não usa.
- Abre o mesmo `FilePath` do access log HTTP uma segunda vez, com outro descritor: não participa da rotação via USR1, não é fechado em `Close()` (o bloco do `Close` é vazio) e as duas escritas no mesmo arquivo não são coordenadas.
- `setupTCPAccessLog` engole o erro de abertura do arquivo e liga o log TCP sempre que `accessLog` existe, sem opção para desligar só o TCP (mudança de comportamento para quem já usa access log HTTP).
- `CountingConn` incrementa `bytesRead`/`bytesWritten` sem sincronização; o proxy TCP lê e escreve em goroutines diferentes, então há data race.
- `ServeTCP` usa `context.Background()`, força `Handshake()` em `*tls.Conn` (o conn que chega ali normalmente não é `*tls.Conn`, então o caminho nunca roda) e grava o certificado de cliente inteiro em PEM em cada evento (linhas de log enormes, crescimento rápido do arquivo).
- Não respeita filtros, `fields.headers`, buffering nem o formato `common`/OTLP do access log; o erro da conexão nunca é passado (sempre `success`).
- Sem testes; código mal formatado (indentação com espaços em `traefik.go`, comentário de `NewTCPEntryPoints` removido, linha em branco extra); o autor diz que é um rascunho inicial e o PR está em `status/1-needs-design-review` sem decisão de design.

### Ajustes

Nenhum: não importado.

### Testes

Não aplicável.

### Motivo

Rejeitado: viola as convenções do projeto (logrus), tem data race e problemas de ciclo de vida do arquivo, e a abordagem ainda não passou pela revisão de design do upstream. Vale reimplementar do zero quando o upstream definir o design (reaproveitando o `accesslog.Handler` e zerolog).
