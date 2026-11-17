# kubernetes: ignore Node updates that do not affect addresses

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13271](https://github.com/traefik/traefik/pull/13271) |
| Tipo | Recurso |
| Área | provider/kubernetes/k8s |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | ishioni |
| Aberto em | 2026-06-02 (última atualização: 2026-06-08) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +78/-0 em 2 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/provider/k8s |
| Status | importado com ajustes |

## O que faz

Adiciona nodeChanged() para que atualizações de Node só disparem rebuild quando InternalIP/ExternalIP mudam, ignorando o heartbeat de status do kubelet (~10s) que o Traefik não lê. Com cluster-scope habilitado (padrão, necessário para NodePortLB), hoje cada heartbeat gera rebuild completo.

## Riscos e situação no upstream

Sobrepõe o #13255, que também adiciona nodeChanged: é preciso escolher um. Um maintainer disse que mudanças desse tipo devem ir para a v3.8 por possíveis efeitos colaterais. Sem review.

## Principais arquivos

- `pkg/provider/kubernetes/k8s/event_handler.go`
- `pkg/provider/kubernetes/k8s/event_handler_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13271/head:upstream-pr-13271`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- Correto e de baixo risco: o heartbeat do kubelet atualiza o status do Node a cada ~10s e hoje cada atualização dispara um rebuild completo da configuração (com recursos cluster-scope habilitados, o padrão). Conferido que no fork nenhum provider usa labels, spec ou condições do Node; só `Status.Addresses` (NodePortLB, inclusive o filtro do #12048, que depende de endpoints e não do Node). Criação e remoção de Node continuam disparando eventos.
- Sobrepunha o #13255, que foi fechado no upstream; sem conflito.
- O teste do PR usava `metav1` sem importar no arquivo atual do fork (não compilava).

Ajustes:
- Testes usam os campos promovidos (`ResourceVersion: "1"`), no estilo do fork.
- Comentário de *porquê* em `nodeChanged`.

Testes: `go test ./pkg/provider/kubernetes/k8s/` ok; `golangci-lint run ./pkg/provider/kubernetes/k8s/...` 0 issues.
