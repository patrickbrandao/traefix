# fix(docker): add event stream timeout to prevent provider from silently hanging

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13578](https://github.com/traefik/traefik/pull/13578) |
| Tipo | Correção |
| Área | provider/docker (event stream) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | sijie-Z |
| Aberto em | 2026-07-28 (última atualização: 2026-09-30) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +14/-2 em 2 arquivos |
| Labels | status/2-needs-review, area/provider/docker, kind/bug/fix, size/S |
| Status | adiado |

## O que faz

Adiciona WatchTimeout (padrão 5 min) ao loop de eventos do Docker. Sem evento nesse prazo, o stream é fechado e operation() retorna erro, e o loop de retry reconecta. É uma defesa contra travamento silencioso que pode ter causado a issue #13549, mas a causa raiz ainda não foi confirmada pelo autor.

## Riscos e situação no upstream

Causa raiz não confirmada. Um revisor apontou que o guard só cobre o Stop do backoff e que cancelamento pode encerrar o retry antes dele. Esse ponto precisa ser resolvido antes de portar. Status review.

## Principais arquivos

- `pkg/provider/docker/pdocker.go`
- `pkg/provider/docker/shared.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13578/head:upstream-pr-13578`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- O próprio autor diz que a causa raiz do issue (upstream 13549, provider Docker que para de tentar reconectar depois de cerca de 20 h) **não foi confirmada**. A primeira versão do PR estava errada e foi retirada depois do review. A atual é uma correção "defensiva" sem reprodução e sem testes.
- A API de eventos do Docker não manda heartbeat. Por isso um timeout de inatividade não distingue "stream travado" de "host sem eventos". Com o padrão de 5 min, todo host tranquilo passaria a reconectar a cada 5 minutos, refazendo `ContainerList` + `inspect` de todos os containers e reenviando a configuração (o `configurationwatcher` descarta configuração idêntica, mas o custo de listagem e o log de erro `Provider error, retrying` ficam). É uma mudança de comportamento padrão para todos os usuários.
- O padrão de drenagem `if !watchdog.Stop() { <-watchdog.C }` é frágil. Com a semântica de timers do Go >= 1.23 (canal síncrono), o `Reset` direto já basta, e a drenagem manual pode bloquear, que é justamente o tipo de travamento que o PR quer evitar.
- O retorno por timeout passa pelo `notify` como erro, com backoff crescente, o que atrasa a reconexão em hosts sem eventos.
- Sem documentação da nova opção `watchTimeout` e sem testes.

**Veredito:** adiado. O #13903 (importado) já corrige um caminho concreto em que o provider ficava com configuração velha (erro de listagem durante o evento era ignorado). Para um stream realmente travado, a abordagem preferível é keepalive TCP ou um timeout opt-in (padrão desligado) que reconecte sem passar pelo backoff de erro, com teste reproduzindo o travamento.

**Testes:** não aplicável (nada importado).
