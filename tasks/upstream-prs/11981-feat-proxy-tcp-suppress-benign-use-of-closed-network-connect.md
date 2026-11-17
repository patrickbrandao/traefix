# feat(proxy/tcp): suppress benign "use of closed network connection" errors

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#11981](https://github.com/traefik/traefik/pull/11981) |
| Tipo | Correção |
| Área | proxy/tcp, logs |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | xpload32004 |
| Aberto em | 2025-08-06 (última atualização: 2025-08-07) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +57/-2 em 2 arquivos |
| Labels | status/1-needs-design-review, size/S, area/service |
| Status | pendente |

## O que faz

Centraliza a classificação de erros benignos de fechamento de conexão TCP (EOF, net.ErrClosed, conexão fechada) para que não poluam o log em nível de erro. Um comentário aponta que é duplicata em objetivo da PR 11838.

## Riscos e situação no upstream

Pequeno (2 arquivos, +57/-2). Possível duplicata da 11838, que deve ser checada antes. Status needs-design-review, sem reviews.

## Principais arquivos

- `pkg/tcp/proxy.go`
- `pkg/tcp/proxy_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/11981/head:upstream-pr-11981`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
