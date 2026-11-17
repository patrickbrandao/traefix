# Return 400 for requests with an invalid chunked body

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13528](https://github.com/traefik/traefik/pull/13528) |
| Tipo | Correção |
| Área | proxy/httputil (tratamento de erro) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | larrasket |
| Aberto em | 2026-07-16 (última atualização: 2026-08-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +95/-0 em 2 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/server |
| Status | pendente |

## O que faz

Requisição com corpo chunked malformado fazia o ComputeStatusCode cair em 500. Agora retorna 400 Bad Request, igual ao net/http. Os erros do leitor chunked não são exportados, então o PR casa as mensagens manualmente. Resolve a issue #11087.

## Riscos e situação no upstream

Casamento de erros por string, que pode quebrar em versões futuras do Go. Status needs-review. PR base v3.7.

## Principais arquivos

- `pkg/proxy/httputil/proxy.go`
- `pkg/proxy/httputil/proxy_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13528/head:upstream-pr-13528`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
