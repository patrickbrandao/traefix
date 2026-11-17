# Add socket option to the Tailscale certificate resolver

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14268](https://github.com/traefik/traefik/pull/14268) |
| Tipo | Recurso |
| Área | tls/certificate-resolver/tailscale |
| Prioridade | **P3** |
| Valor | baixa |
| Esforço de port | baixo |
| Autor | amalsaidov |
| Aberto em | 2026-11-11 (última atualização: 2026-11-11) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +69/-4 em 6 arquivos |
| Labels | status/2-needs-review, kind/bug/fix, size/S |
| Status | pendente |

## O que faz

Adiciona a opção socket ao certificate resolver Tailscale, para apontar o caminho do tailscaled quando ele não está no caminho padrão. Antes disso o resolver não funcionava nesses casos. Sem a opção, o comportamento permanece igual.

## Riscos e situação no upstream

Muda o tipo de configuração de *struct{} para *tailscale.Configuration, o que mexe na documentação gerada e em flags. Só relevante para quem usa o resolver Tailscale com socket fora do padrão. Sem review de maintainer ainda.

## Principais arquivos

- `resolver Tailscale (configuração do certificate resolver)`
- `documentação gerada de install configuration`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14268/head:upstream-pr-14268`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.
