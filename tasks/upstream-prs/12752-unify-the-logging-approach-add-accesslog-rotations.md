# Unify the logging approach, add accessLog rotations

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12752](https://github.com/traefik/traefik/pull/12752) |
| Tipo | Recurso |
| Área | logs/accesslogs |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | TheFedaikin |
| Aberto em | 2026-03-03 (última atualização: 2026-09-12) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +113/-29 em 12 arquivos |
| Labels | status/1-needs-design-review, area/logs, size/M, area/middleware |
| Status | importado com ajustes |

## O que faz

Usa lumberjack para rotacionar todos os arquivos de log (acesso e Traefik), com rotação por SIGUSR1 em todos eles. Também corrige a opção de compressão da seção log, que não era aplicada. Fecha a issue #11342.

## Riscos e situação no upstream

Toca o bootstrap de logs (cmd/traefik/traefik.go, server_signals.go) e a documentação de configuração. Maintainers disseram que está no radar, sem ETA. Um comentário externo deu LGTM.

## Principais arquivos

- `cmd/traefik/logger.go`
- `cmd/traefik/traefik.go`
- `pkg/middlewares/accesslog/logger.go`
- `pkg/server/server_signals.go`
- `pkg/observability/types/logs.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12752/head:upstream-pr-12752`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

### Achados

- Em `status/1-needs-design-review` sem decisão; mudança autocontida, sem dependência nova (lumberjack já é usado para o log do Traefik).
- Regressão de comportamento: o log do Traefik sempre comprimia os backups (`Compress: true` fixo); o PR passa a respeitar `log.compress`, cujo default é `false`, então quem já usa rotação deixaria de ter backups comprimidos.
- Com lumberjack, o arquivo do access log só é aberto na primeira escrita: um caminho sem permissão deixaria de falhar na inicialização e as linhas seriam descartadas em silêncio.
- Pontos positivos: erro de abertura do log do Traefik deixa de ser ignorado, o arquivo de teste deixa de ficar aberto (descritor vazado) e o modo passa de 0666 para 0644.
- Mudança de semântica do USR1: para arquivos rotacionados pelo Traefik, o sinal faz uma rotação imediata (renomeia com timestamp) em vez de fechar e reabrir. Documentado.

### Ajustes

- `TraefikLog.SetDefaults` agora define `Compress = true`, preservando o comportamento anterior e permitindo desligar.
- `accesslog.NewHandler` valida o caminho (cria diretório e abre/fecha o arquivo) também no modo lumberjack.
- Conflito de doc resolvido com o texto atual do Traefix, descrevendo os dois modos do USR1; doc de `log.compress` corrigida; `go generate` para os docs gerados.
- Testes novos: `TestLogRotation_builtin` (rotação via lumberjack no USR1) e `TestNewHandler_builtinRotationUnwritablePath`.

### Testes

- `go build ./...`: ok.
- `go test ./pkg/middlewares/accesslog/ ./cmd/traefik/ ./pkg/server/ ./pkg/server/middleware/ ./pkg/observability/types/`: ok.
- `golangci-lint run` nos pacotes tocados: 0 issues.
