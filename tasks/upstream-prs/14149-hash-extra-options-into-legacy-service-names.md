# Hash extra options into legacy service names

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14149](https://github.com/traefik/traefik/pull/14149) |
| Tipo | Correção |
| Área | provider/kubernetes/crd |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | WlnJBrn09 |
| Aberto em | 2026-10-27 (última atualização: 2026-10-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +209/-43 em 6 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/crd |
| Status | importado com ajustes |

## O que faz

Nomes legados de Service no provider CRD passam a incluir um hash das opções de cada referência (sticky, healthCheck, serversTransport). Referências ao mesmo Service com opções diferentes deixam de se sobrescrever. Sem opções extras, o nome antigo é mantido. safeNaming não muda.

## Riscos e situação no upstream

Mudança de comportamento visível (dashboard, API, métricas, access logs) e para referências cross-provider; há nota de migração. Status: needs-review. Pode quebrar quem depende dos nomes atuais.

## Principais arquivos

- `pkg/provider/kubernetes/crd/kubernetes_http.go`
- `pkg/provider/kubernetes/crd/kubernetes_test.go`
- `pkg/provider/kubernetes/crd/kubernetes_http_test.go`
- `docs/content/migrate/v3.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14149/head:upstream-pr-14149`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- Segurança: com nomes legados, duas referências ao mesmo Service com opções diferentes geram o mesmo nome e a última construída vence em silêncio. Com `allowCrossNamespace`, uma IngressRoute de outro namespace pode sobrescrever as opções (`serversTransport`, `healthCheck`, `sticky`...) do serviço gerado para o dono do Service, por exemplo trocando o ServersTransport. O PR resolve isso com um sufixo de hash das opções; `safeNaming` já não tinha o problema.
- Bug no PR: o hash também era aplicado a referências `kind: TraefikService` com opções e a nomes `ns-nome@kubernetescrd`, que apontam para serviços já existentes. O nome com sufixo não existiria e a referência quebraria.
- Mudança visível de nomes (dashboard, API, métricas, access logs, referências cross-provider), mas só para referências com opções extras. A nota de migração citava versões da linha v3.7 do upstream.
- `hashstructure` já é dependência do projeto (`go.mod` sem mudanças).

Ajustes:
- O hash só é aplicado aos nomes gerados para Kubernetes Services. `TraefikService` e nomes qualificados por provider ficam como estavam. Teste novo para o caso `TraefikService`.
- `fullServiceName` reescrito com `switch` (gocritic) sem mudar a lógica restante.
- Nota de migração movida para a seção `v3.8.0`, com o impacto de segurança e a exceção de `TraefikService`.
- Testes no estilo do fork (`new(...)` no lugar de `ptr.To`).

Testes: `go test ./pkg/provider/kubernetes/crd/` ok; `golangci-lint run ./pkg/provider/kubernetes/crd/...` 0 issues.
