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
| Status | importado |

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

## Revisão (2026-11-16)

**Achados no PR original**

- Melhoria de diagnóstico: um `certFile`/`keyFile` com caminho inexistente era lido como conteúdo inline e só falhava depois, com `tls: failed to find any PEM data`. Agora o file provider reporta o arquivo ausente.
- **Segredos:** o PR toma cuidado para não vazar conteúdo. A mensagem usa `GetTruncatedCertificateName()`, o erro do `os.Stat` (que traz o valor completo) não é encapsulado, e para `keyFile` só é citado o nome do certificado, nunca o valor da chave. Conferi que não há outro log do valor.
- Comportamento: em `tls.certificates` e nos `serversTransports` HTTP/TCP, o certificado ausente é descartado, como já acontecia com outros erros de leitura (antes ele também era descartado, só que mais adiante, no TLS manager). No caso do `serversTransport`, o transporte segue sem aquele certificado de cliente em vez de falhar. Um backend que exige mTLS continua recusando a conexão, então não há rebaixamento de segurança. O certificado padrão de um TLS store é mantido de propósito (só loga o erro), para não disparar a emissão do `defaultGeneratedCert` do ACME por causa de um erro de digitação. A decisão está correta e documentada no código.
- As fixtures apontavam para certificados inexistentes (`snitest1.com.cert` etc.) e passaram a usar pares que existem em `integration/fixtures/https/`. Só afetam os testes do pacote.
- Sobreposição com o #13463 (adiado), que mexe nas mesmas funções. Fica registrado na task do #13463 que um eventual port deve partir deste código e reaproveitar a mesma regra de não logar o valor.

**Ajustes:** nenhum.

**Testes:** `go build ./pkg/...` ok; `go test ./pkg/provider/file/` ok; `golangci-lint run ./pkg/provider/file/...` com 0 issues.
