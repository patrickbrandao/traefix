# fix: avoid UDP timeout before first activity

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13059](https://github.com/traefik/traefik/pull/13059) |
| Tipo | Correção |
| Área | udp |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | cyphercodes |
| Aberto em | 2026-04-28 (última atualização: 2026-09-21) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +152/-10 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/udp |
| Status | pendente |

## O que faz

Impede que uma sessão UDP nova seja fechada pelo ticker de idle timeout antes da primeira leitura ou escrita. Com udp.timeout muito pequeno, o primeiro pacote podia ser perdido. Fecha o issue #13044.

## Riscos e situação no upstream

Base v3.7 e status needs review; o fix é pequeno e isolado em pkg/udp/conn.go. Afeta só configurações com timeout muito baixo.

## Principais arquivos

- `pkg/udp/conn.go`
- `pkg/udp/conn_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13059/head:upstream-pr-13059`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
