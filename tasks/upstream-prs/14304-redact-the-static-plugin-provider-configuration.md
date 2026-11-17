# Redact the static plugin provider configuration

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14304](https://github.com/traefik/traefik/pull/14304) |
| Tipo | Correção |
| Área | logs/redactor (provider plugin) |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | jspdown |
| Aberto em | 2026-11-16 (última atualização: 2026-11-16) |
| Draft | não |
| Branch-alvo | `v2.11` |
| Tamanho | +535/-34 em 3 arquivos |
| Labels | status/0-needs-triage, kind/bug/fix, area/logs, size/M |
| Status | pendente |

## O que faz

A configuração dos plugins de provider (static.PluginConf) aparecia em texto puro no log DEBUG da configuração estática, enquanto a dos middlewares já era redigida. O PR faz RemoveCredentials zerar os valores dos plugins de provider, da mesma forma.

## Riscos e situação no upstream

Base é v2.11, então precisa de port para v3.8. A verificação de redactor.go só cobre dynamic.PluginConf (linha 73), o que confirma o problema no master. Mexe nos mesmos testdata que #14305, então pode haver conflito. Sem review de maintainer ainda.

## Principais arquivos

- `pkg/redactor/redactor.go`
- `pkg/redactor/redactor_config_test.go`
- `pkg/redactor/testdata/secured-static-config.json`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14304/head:upstream-pr-14304`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
