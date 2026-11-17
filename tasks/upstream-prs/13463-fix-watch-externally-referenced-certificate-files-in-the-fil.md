# fix: watch externally referenced certificate files in the file provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13463](https://github.com/traefik/traefik/pull/13463) |
| Tipo | Correção |
| Área | provider/file |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | medio |
| Autor | aliamerj |
| Aberto em | 2026-07-04 (última atualização: 2026-08-26) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +549/-38 em 2 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/M, area/provider/file |
| Status | pendente |

## O que faz

O file provider passa a observar também os arquivos de certificado, chave e CA referenciados pela config dinâmica, mantidos em sincronia com a config. Assim, a renovação in-place reflete sem tocar no arquivo de config, e um certificado que só aparece depois do start também é detectado. Relacionado a #12504.

## Riscos e situação no upstream

Status 1-needs-design-review; review COMMENTED. Acrescenta muitos watchers de arquivo, com risco de custo e de recarga em rajada. Mudança de 549 linhas em file.go e testes, e a sincronização com a config precisa ser correta.

## Principais arquivos

- `pkg/provider/file/file.go`
- `pkg/provider/file/file_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13463/head:upstream-pr-13463`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
