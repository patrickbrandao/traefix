# Report missing certificate files

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14214](https://github.com/traefik/traefik/pull/14214) |
| Tipo | Recurso |
| Área | provider/file |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | ShikhJohari |
| Aberto em | 2026-11-03 (última atualização: 2026-11-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +234/-46 em 7 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/M, area/provider/file |
| Status | pendente |

## O que faz

O provider file passa a logar erro claro quando certFile ou keyFile aponta para um caminho inexistente, em vez de deixar o TLS manager reclamar de 'failed to find any PEM data'. O certificado é descartado, como já ocorre com outros erros de leitura. Para o default store, o valor é mantido para não disparar emissão no ACME.

## Riscos e situação no upstream

Melhora diagnóstico sem mudar o roteamento. Status: needs-review, base master. Pequena lógica de os.Stat no provider file.

## Principais arquivos

- `pkg/provider/file/file.go`
- `pkg/provider/file/file_test.go`
- `pkg/provider/file/fixtures/toml/simple_file_01.toml`
- `pkg/provider/file/fixtures/yaml/simple_file_01.yml`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14214/head:upstream-pr-14214`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
