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
| Status | adiado |

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

## Revisão (2026-11-16)

**Achados no PR original**

- No upstream está em `design-review`. Há só um comentário de review (sugestão de `RWMutex`, já aplicada).
- **Vazamento de segredo em log:** `looksLikeFilePath` trata como caminho qualquer valor de uma linha que não comece com `-----BEGIN`. Desde a mudança em que um valor que não é arquivo existente passou a ser tratado como conteúdo inline (possivelmente secreto), um conteúdo inline sem armadura PEM (chave em base64, por exemplo) seria tratado como caminho. O PR então faz `watcher.Add(filepath.Dir(valor))` e, se falhar, loga `directory=<parte do valor>` em nível `Error` (e em `Debug` quando dá certo). Ou seja, parte do segredo vai para o log.
- **Watch amplo demais:** o filtro de eventos aceita qualquer evento no diretório pai de um arquivo referenciado, não só nos arquivos referenciados. Um certificado em `/tmp`, `/etc/ssl/certs` ou num diretório de logs dispara uma reconstrução completa da configuração a cada arquivo escrito ali. Um valor sem `/` resolve para `.` e passa a vigiar o diretório de trabalho do processo.
- O número de diretórios vigiados é limitado aos referenciados (o PR sincroniza e remove os que saem da configuração), mas não há limite nem validação: qualquer diretório citado na configuração vira um watch do inotify, consumindo o limite `max_user_watches` do host.
- O PR remove o parâmetro `parseTemplate` de `loadFileConfig`. Não há chamada com `false` hoje, mas é uma refatoração fora do escopo.
- Sobreposição com o #14214 (importado), que mexe nas mesmas funções de `file.go` para reportar arquivos de certificado ausentes. Os dois teriam que ser reconciliados.

**Veredito:** adiado. O problema (renovação de certificado não recarrega) é real, mas o PR tem vazamento de segredo em log e um watch amplo demais, além de estar sem decisão de design no upstream. Para trazer no futuro: (1) vigiar só valores com `IsPath()` verdadeiro ou caminhos absolutos cujo diretório pai exista; (2) filtrar eventos pelo nome exato dos arquivos referenciados; (3) nunca logar o valor configurado (usar `GetTruncatedCertificateName`, como faz o #14214); (4) rebasear sobre o #14214.

**Testes:** não aplicável (nada importado).
