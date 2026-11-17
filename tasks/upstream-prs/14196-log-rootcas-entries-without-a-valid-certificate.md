# Log rootCAs entries without a valid certificate

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14196](https://github.com/traefik/traefik/pull/14196) |
| Tipo | Correção |
| Área | service/transport, tcp/dialer |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | sarankumar1325 |
| Aberto em | 2026-10-31 (última atualização: 2026-11-11) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +113/-2 em 4 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/tcp, area/service |
| Status | importado com ajustes |

## O que faz

Um rootCAs de servers transport HTTP ou TCP sem certificado PEM válido era descartado em silêncio, e as requisições falhavam com x509 unknown authority sem pista no log. Agora há log de erro com o nome do arquivo (ou 'content' para valor inline). O comportamento de seguir com as outras CAs se mantém.

## Riscos e situação no upstream

Só logs, sem mudança de comportamento. Status: needs-review. Ajuda diagnóstico em produção.

## Principais arquivos

- `pkg/server/service/transport.go`
- `pkg/tcp/dialer.go`
- `pkg/server/service/transport_test.go`
- `pkg/tcp/dialer_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14196/head:upstream-pr-14196`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: importado com ajustes.**

Achados no PR original:

- A melhoria é correta. `createRootCACertPool` (HTTP e TCP) ignorava o retorno de `AppendCertsFromPEM`. Uma entrada `rootCAs` sem certificado válido (arquivo errado, chave privada, conteúdo quebrado) sumia em silêncio, e o problema só aparecia como `unknown authority` em toda requisição. O comportamento continua igual (loga e segue), só ganha o log.
- Segurança: o conteúdo inline nunca é logado, o que é bom: pode ser uma chave privada colada por engano. O caminho do arquivo era interpolado com `Msgf("... %s", cert)` na mensagem.
- Os testes usavam `LocalhostCert`, que não existe neste código (aqui a fixture é `localhostCert`).

Ajustes:

- O caminho do arquivo vai num campo estruturado (`Str("file", ...)`) em vez de ir interpolado na mensagem, para não misturar dado de configuração com o texto do log.
- Testes adaptados ao campo estruturado e à fixture `localhostCert`. Também foi resolvido o conflito de imports em `transport_test.go`.

Testes:

- `go test ./pkg/server/service/ ./pkg/tcp/`: ok.
- `golangci-lint run ./pkg/server/service/ ./pkg/tcp/...`: 1 issue que já existia e não vem deste PR (`errcheck` em `pkg/tcp/proxy.go:62`, arquivo não tocado). Nenhum issue nos arquivos alterados.
- `go build ./pkg/...`: ok.
