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
| Status | importado com ajustes |

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

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- Hoje o `acme.json` é gravado com `os.WriteFile`, que trunca o arquivo antes de escrever. Se o processo morre ou
  o disco enche no meio da escrita, o arquivo fica truncado ou vazio, e as contas e chaves privadas ACME se
  perdem. O PR passa a gravar num arquivo temporário no mesmo diretório (`CreateTemp`, 0600), faz `fsync`,
  `rename` e `fsync` do diretório. Também corrige um bug real: um erro de `json.MarshalIndent` seguia em
  frente e gravava dados nil.
- Os fallbacks são seguros. Se o temporário não puder ser criado ou o rename falhar (por exemplo, arquivo com
  bind mount em container), o PR grava no lugar, como antes. Symlinks são resolvidos para substituir o alvo,
  e não o link, o mesmo comportamento do `os.WriteFile`.
- **Permissões.** O PR copia o modo do arquivo existente para o novo. Se o arquivo tivesse bits de grupo ou outros
  (por exemplo, `chmod` manual após o boot), eles seriam propagados para um arquivo que contém chaves
  privadas.
- Com o rename, a posse (owner/group) de um `acme.json` provisionado externamente passa a ser a do processo.
  Isso é aceitável e foi documentado aqui.
- O teste `failedWriteKeepsPreviousFile` reexecuta o próprio binário de teste com `RLIMIT_FSIZE` para simular o
  disco cheio. É o binário do próprio `go test`, nada externo.
- As mensagens de assert ("temporary file left behind") contrariavam a convenção.
- O review automatizado citou "file inclusion" no `ReadFile`, mas é falso positivo: o caminho vem da
  configuração de instalação.

Ajustes:
- O modo copiado do arquivo existente é mascarado com `& 0o700`, para nunca conceder acesso a grupo ou outros, e o
  porquê ficou em comentário.
- Removi as mensagens dos asserts.

Testes:
- `go build ./...`: ok.
- `go test ./pkg/provider/acme/`: ok, incluindo os testes novos de escrita atômica, de falha de escrita
  preservando o arquivo anterior e de preservação de modo.
- `golangci-lint run ./pkg/provider/acme/...`: 0 issues.
