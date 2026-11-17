# feat(middleware): add IPAllowGroup middleware for OR-chaining

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13509](https://github.com/traefik/traefik/pull/13509) |
| Tipo | Recurso |
| Área | middleware/ipallowlist |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | Aisu2635 |
| Aberto em | 2026-07-11 (última atualização: 2026-10-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +359/-0 em 5 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/L, area/middleware |
| Status | adiado |

## O que faz

Adiciona o middleware IPAllowGroup, que combina vários IPAllowList com lógica OR: a requisição é liberada se o IP casar com qualquer uma das regras configuradas. Atende o caso de múltiplos parceiros, cada um com seu conjunto de IPs, citado na issue #12249.

## Riscos e situação no upstream

Adiciona campo novo em config dinâmica (exige regenerar zz_generated.deepcopy.go). Status needs-design-review: maintainers pediram análise de impacto e o autor cobrou retorno. Sem review ainda.

## Principais arquivos

- `pkg/middlewares/ipallowlist/ip_allowgroup.go`
- `pkg/middlewares/ipallowlist/ip_allowgroup_test.go`
- `pkg/config/dynamic/middlewares.go`
- `pkg/config/dynamic/zz_generated.deepcopy.go`
- `pkg/server/middleware/middlewares.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13509/head:upstream-pr-13509`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: adiado.**

Achados no PR original:
- O código em si é razoável. Os checkers e as estratégias são construídos uma vez, a avaliação é curto-circuito sobre regras `IPAllowList` e não há problema evidente de segurança. Ele resolve o mesmo caso de uso (#12249) do traefik/traefik#12393 (rejeitado) sem os problemas daquele PR.
- **Problemas de API e desenho:**
  - Cada regra tem seu próprio `rejectStatusCode`, mas só o da *última* regra é usado. É uma configuração confusa, que deveria ser um campo único do grupo.
  - O log de rejeição usa o IP da estratégia da primeira regra, que pode não ser o que as outras regras avaliaram.
  - `typeName` "IPAllowGrouper" é diferente do nome da opção (`ipAllowGroup`).
- **Incompleto para um middleware novo:**
  - Sem suporte no CRD Kubernetes (`traefikio/v1alpha1` e conversão em `crd/kubernetes.go`).
  - Sem docs de referência, sem atualização de `docs/content/reference/dynamic-configuration/*` (file/labels) e sem testes de labels, KV ou redactor.
  - A WebUI não conhece o tipo novo.
- Está em `status/1-needs-design-review` no upstream, e há duas propostas concorrentes (#12393 e #13509). A forma da API ainda não foi decidida, e importar agora criaria uma opção de configuração própria do Traefix que pode divergir do que o upstream adotar.

Motivo do adiamento: feature nova com desenho de API pendente no upstream e implementação incompleta (CRD, docs, WebUI). Reavaliar quando o upstream decidir o desenho, ou se o Traefix precisar do recurso. Nesse caso, unificar o `rejectStatusCode` no grupo e completar o suporte a CRD e docs.

Testes: nenhum, porque o PR não foi importado.
