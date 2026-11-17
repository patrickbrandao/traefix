# Add upstream keep-alive connections option to ingress-nginx provider

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13229](https://github.com/traefik/traefik/pull/13229) |
| Tipo | Recurso |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | baixo |
| Autor | amazon7737 |
| Aberto em | 2026-05-22 (última atualização: 2026-10-20) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +108/-14 em 5 arquivos |
| Labels | kind/enhancement, status/2-needs-review, size/S, area/provider/k8s/ingress-nginx |
| Status | importado com ajustes |

## O que faz

Adiciona a opção upstreamKeepaliveConnections ao provider Kubernetes Ingress NGINX, mapeada para ServersTransport.MaxIdleConnsPerHost (padrão 320, igual ao ingress-nginx). Sem isso, os transports gerados caem no padrão do Go (2 conexões ociosas por host). Fecha a lacuna do par com upstreamKeepaliveTimeout. Fixes #13123.

## Riscos e situação no upstream

Sem review dos maintainers (só comentário 'Any update?'); status 2-needs-review. Mudança pequena e isolada, inclui testes e docs. Sem conflito conhecido.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `docs/content/reference/install-configuration/providers/kubernetes/kubernetes-ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13229/head:upstream-pr-13229`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- O problema é real: o provider ingress-nginx cria um ServersTransport por Ingress sem `MaxIdleConnsPerHost`, então vale o default do Go (2 conexões ociosas por host). Com carga, isso abre conexões novas o tempo todo. O PR usa o default do NGINX (320).
- **Bug latente que o PR deixava exposto:** `upstreamKeepaliveConnections: 0` vira `MaxIdleConnsPerHost: -1`, e o fast proxy (`pkg/proxy/fast/connpool.go`) faz `make(chan *conn, -1)`, o que causa **panic** em runtime. O mesmo vale para qualquer `maxIdleConnsPerHost` negativo vindo de CRD ou arquivo (o valor é documentado como "If negative, disables connection reuse").
- Valores negativos configurados pelo usuário eram repassados como estavam.
- `TestBuildServersTransportKeepAlive` testava o `http.Transport` da stdlib, não o código do Traefik.

**Ajustes**

- `newConnPool` limita a capacidade a `max(n, 0)`, sem panic; teste novo em `TestConnPool_MaxIdleConn`.
- `<= 0` passa a ser tratado como "desligado" (-1).
- Removi o teste da stdlib; o teste unitário cobre 320, 42 e 0 → -1.
- Docs (YAML, TOML, CLI e tabela) e `configuration-options.md` regenerados.
- O helper do PR que preenche o default 320 nas expectativas de `TestLoadIngresses` foi mantido, para não reescrever centenas de fixtures.

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress-nginx/ ./pkg/proxy/fast/` ok.
- `golangci-lint` nos dois pacotes: sem issues novas. Há 2 `errcheck` em `pkg/proxy/fast/upgrade.go` e `proxy_test.go`, que já existiam (arquivos não tocados).
