# Write the ACME local store file atomically

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14004](https://github.com/traefik/traefik/pull/14004) |
| Tipo | Correção |
| Área | tls/acme |
| Prioridade | **P1** |
| Valor | alta |
| Esforço de port | baixo |
| Autor | KR-Ravindra |
| Aberto em | 2026-10-04 (última atualização: 2026-10-25) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +168/-1 em 3 arquivos |
| Labels | status/2-needs-review, area/acme, kind/bug/fix, size/M |
| Status | pendente |

## O que faz

Grava acme.json de forma atomica (arquivo temporario 0600 no mesmo diretorio, Sync e os.Rename), com fallback para escrita in-place quando o rename nao e possivel (ex.: bind mount de arquivo unico). Evita perder todos os certificados e a chave da conta se a escrita for interrompida.

## Riscos e situação no upstream

Escopo pequeno (3 arquivos de pkg/provider/acme). Fallback in-place reintroduz o risco quando o rename falha. Autor reposicionou o PR para v3.7 e declarou uso de agente de IA. Tres reviews COMMENTED.

## Principais arquivos

- `pkg/provider/acme/local_store.go`
- `pkg/provider/acme/local_store_test.go`
- `pkg/provider/acme/local_store_unix_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14004/head:upstream-pr-14004`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
