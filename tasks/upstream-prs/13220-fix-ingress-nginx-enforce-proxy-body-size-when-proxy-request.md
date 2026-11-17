# fix(ingress-nginx): enforce proxy-body-size when proxy-request-buffering is off

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13220](https://github.com/traefik/traefik/pull/13220) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx, middlewares/buffering |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | filip2mac |
| Aberto em | 2026-05-21 (última atualização: 2026-09-30) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +275/-15 em 6 arquivos |
| Labels | contributor/waiting-for-corrections, kind/bug/fix, size/M, area/provider/k8s/ingress-nginx |
| Status | adiado |

## O que faz

Corrige proxy-body-size ignorado quando proxy-request-buffering está off. O limite passa a ser lido antes do early return e força o buffering de requisição, como o client_max_body_size do Nginx. Corrige o comportamento que hoje deixa passar bodies grandes.

## Riscos e situação no upstream

CHANGES_REQUESTED de um maintainer, e a última rodada de testes quebrou porque ProxyBodySize tem padrão de 1MB e o buffering passou a ser aplicado a todos. Precisa de retrabalho antes de revisão. Toca buffering.go e proxy/httputil/proxy.go.

## Principais arquivos

- `pkg/middlewares/buffering/buffering.go`
- `pkg/middlewares/buffering/buffering_test.go`
- `pkg/provider/kubernetes/ingress-nginx/middleware.go`
- `pkg/proxy/httputil/proxy.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13220/head:upstream-pr-13220`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados**

- O bug é real e tem impacto de segurança, do tipo *fail-open*: no provider ingress-nginx o `proxyRequestBuffering` vem desligado por padrão, e com o buffering de requisição desligado o `proxy-body-size` (annotation ou default de 1MB) não é aplicado. Uma annotation de limite de corpo é ignorada sem aviso, ao contrário do `client_max_body_size` do NGINX.
- A correção do PR muda a semântica do middleware genérico `Buffering`: com `disableRequestBuffer: true`, `maxRequestBodyBytes` passa a ser aplicado via `http.MaxBytesReader`. Isso contradiz a documentação atual ("When `true`, `maxRequestBodyBytes` is not enforced") e um teste explícito do upstream (`TestBuffering_DisableRequestBuffer`, "bypasses MaxRequestBodyBytes enforcement"), afetando quem usa o middleware fora do ingress-nginx.
- O PR ainda está em andamento: houve CHANGES_REQUESTED, depois um maintainer pediu a correção de vários testes, porque o default de 1MB fazia o middleware ser anexado a todos os Ingress. Um maintainer sugeriu um middleware dedicado para limitar o corpo.
- A parte de `ComputeStatusCode` (mapear `*http.MaxBytesError` para 413) está correta e não tem risco.

**Tentativa local**

Implementei uma variante em que o middleware só é criado quando a annotation `proxy-body-size` é explícita, com testes de streaming de ponta a ponta. Ela quebrava o teste do upstream que documenta o bypass, então foi revertida para não divergir da semântica pública do middleware sem decisão do upstream.

**Motivo do adiamento**

Mudança de semântica num middleware público, sem decisão dos maintainers sobre a abordagem (flag no Buffering ou middleware novo). Reavaliar quando o upstream fechar o design. Enquanto isso, quem depende de `proxy-body-size` deve habilitar `proxy-request-buffering: "on"`.

**Testes**

A variante revertida passava em `go test ./pkg/proxy/httputil/` e falhava em `TestBuffering_DisableRequestBuffer` (teste do upstream). Nada foi mantido.
