# Reuse cached plugin archive when plugins.traefik.io is unreachable

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13006](https://github.com/traefik/traefik/pull/13006) |
| Tipo | Correção |
| Área | plugins |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | 5queezer |
| Aberto em | 2026-04-19 (última atualização: 2026-11-01) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +105/-4 em 2 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, size/M, area/plugins |
| Status | pendente |

## O que faz

Quando o download de um plugin falha e já existe um archive em cache para a mesma versão, usa o cache em vez de dar erro. Hoje a falha de rede no boot faz manager.ResetAll() derrubar todos os middlewares de plugin, e a correção mantém o hash pinning obrigatório.

## Riscos e situação no upstream

Base v3.7 (autor fez rebase para v3.7.17), então precisa adaptar para master. Um maintainer respondeu que o comportamento atual é escolha de design intencional, então a aceitação não está garantida. Comentários de vários usuários relatam outages reais após reboot; não há revisão formal.

## Principais arquivos

- `pkg/plugins/manager.go`
- `pkg/plugins/manager_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13006/head:upstream-pr-13006`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
