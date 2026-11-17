# feat(k8s/crd): add Middleware status conditions and sourceRange CIDR validation

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#12995](https://github.com/traefik/traefik/pull/12995) |
| Tipo | Recurso |
| Área | provider/kubernetes/crd, middlewares/ipallowlist |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | SergeyMuha |
| Aberto em | 2026-04-16 (última atualização: 2026-11-13) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +185/-1 em 7 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/M, area/provider/k8s/crd |
| Status | rejeitado |

## O que faz

Adiciona a condição de status Valid ao CRD Middleware, para que erros de configuração apareçam em kubectl get/describe e não só nos logs. Também adiciona validação de sourceRange no schema OpenAPI e validação em runtime dos CIDRs do ipAllowList.

## Riscos e situação no upstream

Label needs-design-review sem revisão de maintainer; autor relata que a falta do merge causou um incidente. Muda CRDs e zz_generated deepcopy (regenerar com make generate-crd) e pede RBAC de patch/update em middlewares/status.

## Principais arquivos

- `pkg/provider/kubernetes/crd/kubernetes.go`
- `pkg/provider/kubernetes/crd/client.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/middleware.go`
- `pkg/provider/kubernetes/crd/traefikio/v1alpha1/zz_generated.deepcopy.go`
- `docs/content/reference/dynamic-configuration/traefik.io_middlewares.yaml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/12995/head:upstream-pr-12995`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: rejeitado.**

Achados:
- Loop de escrita no API server: `writeMiddlewareStatus` roda a cada rebuild da configuração, para todos os Middlewares, sempre com `LastTransitionTime: metav1.Now()`. Cada patch muda o `resourceVersion`, o informer de Middlewares (cujo `objChanged` só compara `resourceVersion`) dispara um novo evento, que gera novo rebuild e novo patch. Resultado: rebuilds e escritas contínuas no API server (limitados só pelo `throttleDuration`), carga desnecessária no cluster e no Traefik.
- Sem eleição de líder: todas as réplicas do Traefik escreveriam o status, brigando entre si.
- Escrita síncrona no API server dentro do caminho de build da configuração, sem timeout; um API server lento atrasa toda a configuração.
- Exige RBAC novo (`patch/update` em `middlewares/status`); sem ele, cada rebuild loga um erro por Middleware.
- Status `needs-design-review` sem decisão de maintainer.

A validação de CIDR do `ipAllowList` no provider tem valor, mas hoje um `sourceRange` inválido já falha na criação do middleware (o router que o usa fica em erro); o ganho isolado é pequeno.

Motivo: o desenho de status gera um loop de escrita e reconstrução. Um desenho aceitável precisaria comparar o status antes de escrever, preservar `LastTransitionTime` quando a condição não muda, ignorar mudanças só de status no event handler e escrever apenas no líder.

Testes: nenhum (não importado).
