# Apply server-alias to every Ingress of the host

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14124](https://github.com/traefik/traefik/pull/14124) |
| Tipo | Correção |
| Área | provider/kubernetes/ingress-nginx |
| Prioridade | **P2** |
| Valor | media |
| Esforço de port | medio |
| Autor | amazon7737 |
| Aberto em | 2026-10-22 (última atualização: 2026-10-27) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +682/-15 em 6 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/M, area/provider/k8s/ingress-nginx |
| Status | importado com ajustes |

## O que faz

A anotação server-alias passa a valer para todo o host, e não só para o Ingress que a declara, como faz o ingress-nginx. A propriedade segue a regra do ingress-nginx: o primeiro Ingress, por ordem de criação, que declara aliases para o host é o dono. Complementa o PR 14110.

## Riscos e situação no upstream

Depende do 14110 para o caso de Prefix envolvendo Exact. Muda o comportamento de aliases entre Ingresses do mesmo host. Status: aguardando review. Docs atualizadas.

## Principais arquivos

- `pkg/provider/kubernetes/ingress-nginx/build.go`
- `pkg/provider/kubernetes/ingress-nginx/model.go`
- `pkg/provider/kubernetes/ingress-nginx/kubernetes_test.go`
- `docs/content/reference/routing-configuration/kubernetes/ingress-nginx.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14124/head:upstream-pr-14124`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Achados no PR original**

- O PR aplica o `server-alias` a **todas** as locations do host, como o ingress-nginx, e não só às do Ingress que declara a annotation. Quando vários Ingresses do mesmo host declaram aliases, vale o do Ingress mais antigo.
- **Segurança, influência entre namespaces:** na forma original, um usuário com permissão para criar Ingress num namespace `A` podia criar um Ingress para o host `vitima.example.com` com `server-alias: atacante.example.com`. Se fosse o primeiro (ou o único) a declarar alias para esse host, as rotas dos Ingresses do namespace `B` para `vitima.example.com` passavam a responder também em `atacante.example.com`, um hostname controlado pelo atacante. Isso permite expor a aplicação de outro time num domínio de terceiros (phishing, cookies emitidos para outro domínio, contorno de controles feitos por hostname). Confirmei com um teste: sem o ajuste, as rotas do `victim-ingress` (namespace `other`) recebiam o alias do `alias-ingress` (namespace `default`).
- Os testes do PR usavam o formato antigo de chaves de router e não compilariam no código atual.

**Ajustes**

- O alias do host só é aplicado às locations de Ingresses do **mesmo namespace** do Ingress dono do alias (helper `hostAliases`). Para Ingresses de outros namespaces, o comportamento é o anterior ao PR (sem alias). É uma divergência intencional e documentada em relação ao ingress-nginx.
- Testes novos e focados (`TestLoadIngresses_serverAliasScope`): host compartilhado no mesmo namespace (o alias vale para todos) e entre namespaces (o alias não vaza), com a fixture nova `ingress-with-server-alias-cross-namespace.yml`. As fixtures do PR foram mantidas.
- Documentação da annotation atualizada com o escopo por namespace.

**Testes**

- `go build ./...` ok.
- `go test ./pkg/provider/kubernetes/ingress-nginx/` ok. O caso entre namespaces falha sem o ajuste.
- `golangci-lint run ./pkg/provider/kubernetes/ingress-nginx/...`: 0 issues.
