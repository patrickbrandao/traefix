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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

**Achados no PR original**

- **Vazamento de segredo em log (confirmado):** `doOnStruct` só reconhecia `dynamic.PluginConf`. Como `static.PluginConf` é outro tipo, a configuração dos plugins de provider (`providers.plugin.<nome>`), que pode conter tokens e senhas, saía em claro no log DEBUG `Static configuration loaded`. A correção trata os dois tipos com `resetPlugin`: as chaves são mantidas e os valores esvaziados.
- O PR é de um maintainer (jspdown) e foi aberto contra `v2.11` (imports `traefik/v2`), então não aplica direto no v3.
- O PR também remove os helpers internos `anonymize`/`removeCredentials` com o parâmetro `indent`. É uma refatoração de testes sem efeito funcional.
- Sobreposição com o #14305 (mesmo autor), que também cria `TestRemoveCredentials_staticConfiguration` e `secured-static-config.json`.

**Ajustes (port manual para v3)**

- Só a correção funcional em `redactor.go` (checagem de `static.PluginConf`). A remoção dos helpers com `indent` ficou de fora para não ampliar o diff.
- Testes: `TestDo_staticConfiguration` foi dividido em `fullStaticConfiguration()` + teste; um plugin de provider foi incluído na configuração completa; novo `TestRemoveCredentials_staticConfiguration` com `testdata/secured-static-config.json`, gerado com `-update_expected` e revisado (o plugin aparece como `"foo": {"foo": {}}`).
- O #14305 reaproveita esse teste e esse arquivo em vez de criar os seus.

**Testes:** `go build ./pkg/...` ok; `go test ./pkg/redactor/` ok; `golangci-lint run ./pkg/redactor/...` com 0 issues.
