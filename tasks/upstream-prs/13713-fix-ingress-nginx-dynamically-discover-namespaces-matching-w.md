# fix(ingress-nginx): dynamically discover namespaces matching watchNamespaceSelector

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13713](https://github.com/traefik/traefik/pull/13713) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | luisdeniz |
| Aberto em | 2026-08-18 (última atualização: 2026-10-23) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +542/-1 em 3 arquivos |
| Labels | status/2-needs-review, size/L, area/provider/k8s, area/provider/k8s/ingress-nginx |
| Status | adiado |

## O que faz

Com watchNamespaceSelector, o provider ingress-nginx hoje só descobre namespaces que casam com o seletor na inicialização. Esta mudança adiciona um informer de Namespace que inicia/para informers por namespace conforme labels mudam, sem precisar reiniciar o pod.

## Riscos e situação no upstream

Base v3.7, então precisa de adaptação para master. Status status/2-needs-review, sem review de maintainer ainda. Toca o mesmo tema de seleção de namespaces do #13806. Adiciona watch cluster-scoped de Namespace (exige RBAC de namespaces).

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/client.go`
- `pkg/provider/kubernetes/ingress-nginx/client_test.go`
- `docs/content/reference/install-configuration/providers/kubernetes/kubernetes-ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13713/head:upstream-pr-13713`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados**

- **Deadlock e vazamento de goroutines em `removeNamespace`:** as factories de cada namespace são iniciadas com o `stopCh` global (`c.stopCh`, o `ctx.Done()` do provider). `SharedInformerFactory.Shutdown()` bloqueia até as goroutines dos informers terminarem, e isso só acontece quando o canal com que foram iniciadas é fechado. Quando um namespace deixa de casar com o seletor (ou é apagado), `Shutdown()` trava para sempre o handler do informer de Namespaces. Novos eventos de namespace param de ser processados e os informers do namespace removido continuam rodando. Para corrigir, cada namespace precisa do seu próprio canal de parada.
- **Regressão de memória:** as factories criadas dinamicamente não usam `kinformers.WithTransform(k8s.StripManagedFields)`, que o `WatchAll` atual aplica em todos os informers.
- **Duplicação:** `addNamespace` copia toda a montagem de informers do `WatchAll`. Qualquer mudança futura (transforms, filtros de Secret, novos tipos) teria de ser feita em dois lugares; o caminho certo é extrair um helper único.
- `OnDelete` não trata `cache.DeletedFinalStateUnknown`, então namespaces apagados durante uma desconexão nunca são removidos.
- `WaitForCacheSync` roda dentro do handler de eventos e bloqueia a fila de eventos de namespace enquanto sincroniza.
- Do ponto de vista de segurança, não há aumento de superfície: só quem pode rotular namespaces (privilégio de cluster) controla o seletor.
- No upstream: sem revisão de maintainer, só o status "review".

**Motivo do adiamento**

A ideia é boa (a descoberta dinâmica corrige um problema real do `watchNamespaceSelector`, que hoje só lista namespaces na inicialização), mas o PR tem um deadlock e uma regressão de memória. Corrigir exige reestruturar o cliente com um canal de parada por namespace e um helper de criação de informers compartilhado. Reavaliar depois da revisão do upstream ou como tarefa própria do Traefix.

**Testes**

Nenhum (nada importado).
