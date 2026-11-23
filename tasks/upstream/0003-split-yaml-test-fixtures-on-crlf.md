# 0003 — Split YAML test fixtures on CRLF document separators

- **Status:** aplicada (`2b558060d`, 2026-11-23)
- **Tipo:** bugfix
- **Prioridade:** baixa
- **Upstream:** traefik/traefik#13866 — commits `afd25cee2` (branch v3.7)
- **Aplica limpo:** sim
- **Conflito com Traefix:** não

## O que mudou no oficial
`pkg/provider/kubernetes/k8s/parser.go` (`MustParseYaml`, usado pelos testes dos providers Kubernetes): o
separador `---` passa a ser reconhecido também com `\r\n` e espaços no fim da linha, por regex. Antes, um checkout
com CRLF descartava em silêncio todos os documentos depois do primeiro separador. Teste novo em `parser_test.go`.

## Impacto no Traefix
Nenhuma alteração local nesses arquivos. Só afeta testes.

## Plano de implementação
1. `git cherry-pick -x afd25cee2`
2. Nenhum ajuste.
3. Testes: `go test ./pkg/provider/kubernetes/...`

## Critério de pronto
- `go test ./pkg/provider/kubernetes/k8s/` passa, e os demais testes dos providers Kubernetes seguem passando.
