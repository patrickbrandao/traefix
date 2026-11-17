# Expose HTTP2 receive window sizes on entry points

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14238](https://github.com/traefik/traefik/pull/14238) |
| Tipo | Recurso |
| Área | entrypoint/http2 |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | Bafoleco |
| Aberto em | 2026-11-07 (última atualização: 2026-11-09) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +26/-6 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/server |
| Status | importado com ajustes |

## O que faz

Expõe maxReceiveBufferPerStream e maxReceiveBufferPerConnection nas entrypoints, repassando-os às configurações HTTP/2 do Go. Valores omitidos mantêm o comportamento atual. Útil para upload em links de alta banda com RTT alto.

## Riscos e situação no upstream

O próprio autor marca como draft para discussão, com contribuição assistida por IA. Afeta só janelas de recebimento no lado da requisição, e aumenta a memória por conexão/stream se configurado. Sem review de maintainer ainda (status/2-needs-review). Requer make generate.

## Principais arquivos

- `pkg/config/static (configuração de entrypoint HTTP2, a confirmar)`
- `pkg/server (configuração do servidor HTTP/2)`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14238/head:upstream-pr-14238`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

PR aberto, status "review"; o autor o descreve como "draft for discussion".

### Achados no PR original

- Repassa `MaxReceiveBufferPerConnection` e `MaxReceiveBufferPerStream` ao `http.HTTP2Config` do Go. O padrão (`0`)
  mantém o comportamento atual. A validação rejeita conexão abaixo de 65535 e valores de stream negativos.
- **Risco de memória (configuração do operador):** janelas maiores permitem que cada cliente envie mais dados ainda não
  lidos. O pior caso por conexão é `maxReceiveBufferPerConnection`, e um valor alto multiplicado por muitas conexões abre
  espaço para DoS de memória. O PR não avisava sobre isso.
- O teste só confere o repasse dos valores, sem cobrir a validação.

### Ajustes

- Doc: aviso sobre o consumo de memória por conexão nas duas opções.
- Testes de validação (conexão abaixo do mínimo, stream negativo).

### Testes

- `go test ./pkg/server/ -run 'TestHTTP2Config|TestHTTP2ReceiveBufferValidation'` ok; `go test ./pkg/config/static/` ok.
- `golangci-lint run ./pkg/server/... ./pkg/config/static/...`: 0 issues.
