# Resolve signing intermediate issuer for OCSP stapling entries

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14213](https://github.com/traefik/traefik/pull/14213) |
| Tipo | Correção |
| Área | tls/ocsp |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | AdamMagued |
| Aberto em | 2026-11-03 (última atualização: 2026-11-05) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +386/-13 em 4 arquivos |
| Labels | status/1-needs-design-review, kind/bug/fix, size/L, area/tls |
| Status | importado com ajustes |

## O que faz

Para o stapling de OCSP, o emissor é hoje assumido como cert.Certificate[1] ou cai no leaf. O PR busca o emissor real por DN, SubjectKeyId/AuthorityKeyId e verificação criptográfica (CheckSignatureFrom), e também cobre responders delegados. Atualiza o emissor de entradas em cache.

## Riscos e situação no upstream

Toca o gerenciador TLS e o cache OCSP. Status: needs-design-review. Tem testes unitários e de integração; é bom para cadeias fora de ordem.

## Principais arquivos

- `pkg/tls/ocsp.go`
- `pkg/tls/ocsp_test.go`
- `pkg/tls/tlsmanager.go`
- `pkg/tls/tlsmanager_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14213/head:upstream-pr-14213`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados:
- Correção legítima. Hoje o emissor usado no OCSP stapling é sempre `cert.Certificate[1]`. Com bundles fora de
  ordem (leaf, root, intermediate), a requisição OCSP vai com o issuer errado e a resposta, sobretudo quando
  vem de um responder delegado, falha na verificação de assinatura. O resultado é que o certificado nunca
  ganha staple. O PR procura na cadeia o certificado cujo subject é o issuer do leaf, cujo SKI bate com o AKI
  e cuja chave verifica a assinatura do leaf (`CheckSignatureFrom`).
- Não há enfraquecimento de validação. O issuer serve para montar a requisição e para validar a resposta em
  `ocsp.ParseResponseForCert`, e um issuer errado só faz o staple falhar (fail-safe). A cadeia é fornecida
  pelo operador.
- **Data race introduzida pelo PR.** `Upsert` alterava `entry.issuer` de uma entrada já em cache, que é lida
  sem lock pela goroutine de atualização de staples.
- **Data race preexistente no upstream**, exposta pelo teste novo com `-race`: `updateStaple` grava
  `entry.staple` e `entry.nextUpdate` sem sincronização, enquanto `GetStaple` lê o staple em cada handshake TLS.
- O segundo fallback do `findIssuer` (casar só pelo nome, sem verificar assinatura) não serve para nada:
  se a assinatura não confere, o certificado não é o emissor.
- Achado não corrigido, fora do escopo: `updateStaple` lê a resposta do responder com `io.ReadAll`, sem limite de
  tamanho. O risco é baixo, porque o responder vem do certificado ou de um override do operador.

Ajustes:
- `Upsert` não altera mais a entrada. Quando o issuer muda (`!issuer.Equal(old)`), a entrada é substituída por
  uma nova (copy-on-write), sem o staple antigo, que será buscado de novo com o issuer correto pelo
  `ForceStapleUpdates` já disparado em `UpdateConfigs`.
- `ocspEntry` ganhou um `sync.RWMutex` que protege `staple`/`nextUpdate`, usado em `GetStaple`, `updateStaples` e
  `updateStaple`. Isso corrige a race preexistente.
- `findIssuer` foi simplificado: aceita só o candidato verificado por assinatura e, se não achar, cai no
  comportamento anterior (`Certificate[1]`, retornando erro se não fizer parse, ou o próprio leaf). Assim não
  há regressão para cadeias que hoje funcionam.
- O teste do manager usa `GetStaple` em vez de ler `entry.staple` direto. Adicionei o teste
  `TestOCSPStapler_Upsert_issuerChange`.

Testes:
- `go build ./...`: ok.
- `go test -race ./pkg/tls/`: ok. Sem os ajustes, `TestManager_UpdateConfigs_OCSPMultiCertChain` acusava
  DATA RACE.
- `golangci-lint run ./pkg/tls/...`: 0 issues.
