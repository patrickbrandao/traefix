# Fix TCP entry point shutdown on keepalive errors

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14094](https://github.com/traefik/traefik/pull/14094) |
| Tipo | Correção |
| Área | server/entrypoint-tcp |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | deril |
| Aberto em | 2026-10-19 (última atualização: 2026-10-30) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +60/-7 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S, area/server |
| Status | adiado (combinado no #14010) |

## O que faz

Fecha a conexao aceita se a configuracao de keep-alive falhar e continua aceitando, em vez de tratar o erro como falha do listener, que parava o entrypoint. Reproduzido no FreeBSD (ECONNRESET), issue #8071.

## Riscos e situação no upstream

Duplicado funcional do #14010 (mesmo arquivo e mesma correcao, com o teste no FreeBSD como evidencia extra). Base v3.7. Dois reviews COMMENTED; label status/2-needs-review. Escolher um dos dois PRs.

## Principais arquivos

- `pkg/server/server_entrypoint_tcp.go`
- `pkg/server/server_entrypoint_tcp_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14094/head:upstream-pr-14094`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, status "review".

### Achados no PR original

- Corrige o mesmo bug do #14010: uma conexão resetada logo depois do `accept` faz o setsockopt de keep-alive falhar
  (`ECONNRESET` no FreeBSD), e o erro era devolvido como erro do listener, derrubando o entrypoint para sempre (DoS
  remoto sem autenticação).
- Implementação limpa (função `acceptTCPWithKeepAlive` com dependências injetadas, sem variável global), mas só registra
  em debug: uma falha sistemática descartaria todo o tráfego sem nenhum sinal no nível padrão de log.

### Veredito: combinado no #14010

A estrutura deste PR foi usada na importação do #14010, junto com o aviso com limite de taxa daquele PR. Ver a seção de
revisão do #14010 e o commit `Import traefik/traefik#14010`. Nada a aplicar separadamente.

### Testes

Cobertos pela importação do #14010 (`TestAcceptTCPWithKeepAlive`, baseado no teste deste PR).
