# Support middleware with TraefikService CRD

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13018](https://github.com/traefik/traefik/pull/13018) |
| Tipo | Correção |
| Área | provider/kubernetes/crd, routing-configuration/services |
| Prioridade | **P2** |
| Valor | alta |
| Esforço de port | alto |
| Autor | sdelicata |
| Aberto em | 2026-04-21 (última atualização: 2026-05-11) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +1325/-50 em 15 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/L, area/provider/k8s/crd |
| Status | rejeitado |

## O que faz

Corrige middlewares declarados em filhos weighted/HRW/mirroring/failover que apontam para outro TraefikService, que são descartados em silêncio no runtime. Um middleware de autenticação, por exemplo, poderia ser ignorado sem aviso. O PR acrescenta o campo middlewares nesses tipos e envolve o handler filho com a cadeia.

## Riscos e situação no upstream

Maintainer comentou que a abordagem está errada: middlewares só deveriam ser permitidos em LoadBalancerSpec quando apontam para um Service Kubernetes, e para TraefikService devem ser definidos no próprio serviço alvo. O PR precisa ser redesenhado. Base v3.7, +1325 linhas em 15 arquivos.

## Principais arquivos

- `pkg/config/dynamic/http_config.go`
- `pkg/provider/kubernetes/crd/kubernetes_http.go`
- `pkg/config/dynamic/zz_generated.deepcopy.go`
- `docs/content/reference/routing-configuration/kubernetes/crd/http/traefikservice.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13018/head:upstream-pr-13018`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: rejeitado.**

Achados:
- Um maintainer (juliens) disse que a abordagem está errada: middlewares só deveriam ser aceitos direto no `LoadBalancerSpec` quando ele aponta para um Service Kubernetes; para um TraefikService, devem ser definidos no próprio serviço alvo. Sugeriu, no lugar, emitir um aviso quando um TraefikService é referenciado com middlewares.
- O PR acrescenta um campo `middlewares` em `WRRService`, `HRWService`, `MirrorService` e `Failover` da configuração dinâmica (todos os providers, não só o CRD) e envolve handlers filhos no service manager: superfície grande (+1325 linhas, base v3.7) para uma API que o upstream não quer.
- Do ponto de vista de segurança, o problema descrito (um middleware de autenticação ignorado em silêncio) é real, mas a correção certa é o aviso/erro sugerido pelo maintainer, não uma nova API.

Motivo: abordagem rejeitada pelo upstream; adotar no fork criaria uma API divergente difícil de manter. Se for necessário, implementar só o aviso quando houver middlewares num filho que referencia TraefikService.

Testes: nenhum (não importado).
