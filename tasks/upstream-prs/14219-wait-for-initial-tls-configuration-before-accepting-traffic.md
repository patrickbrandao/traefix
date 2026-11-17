# Wait for initial TLS configuration before accepting traffic

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#14219](https://github.com/traefik/traefik/pull/14219) |
| Tipo | Correção |
| Área | server/startup, tls, acme |
| Prioridade | **P2** |
| Valor | alta |
| Esforço de port | alto |
| Autor | escooterclinic |
| Aberto em | 2026-11-03 (última atualização: 2026-11-13) |
| Draft | não |
| Branch-alvo | `v3.7` |
| Tamanho | +686/-34 em 11 arquivos |
| Labels | kind/enhancement, status/1-needs-design-review, area/acme, size/L, area/tls, area/server |
| Status | adiado |

## O que faz

Antes de aceitar conexões, o Traefik espera pelo primeiro snapshot dos providers e pela aplicação bem-sucedida de TLS e roteamento. Assim, uma conexão que chega cedo não recebe o certificado padrão gerado quando já existe certificado válido local. Erros de TLS na inicialização abortam o startup. Reloads continuam best-effort.

## Riscos e situação no upstream

Muda a semântica de startup e de listeners (+686/-34, 11 arquivos). Status: needs-design-review; é resubmissão de outro PR (#14218) via fork pessoal. Alto risco de regressão no boot, inclusive com providers lentos. Há testes de startup.

## Principais arquivos

- `pkg/server/server.go`
- `pkg/server/configurationwatcher.go`
- `cmd/traefik/traefik.go`
- `pkg/provider/acme/startup_test.go`
- `pkg/server/server_startup_test.go`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/14219/head:upstream-pr-14219`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: adiado.**

Achados:
- A motivação é legítima. Logo no boot, um handshake TLS pode chegar antes de os certificados e routers serem
  aplicados, e o cliente recebe o certificado default autogerado ou um 404. Um usuário mostrou, com
  `reusePort` em rollout surge, que o PR zera essas falhas.
- **Não há timeout de startup.** Os accept loops só começam depois que *todos* os providers da lista
  (`internal`, `file`, `http`, `docker`, `swarm` e cada resolver ACME) enviam o snapshot inicial. Se o socket
  do Docker estiver indisponível ou o endpoint do provider HTTP fora do ar no boot, o Traefik nunca começa a
  aceitar conexões, nem no entry point de `ping`/health. O orquestrador entra em loop de restart, enquanto hoje
  o Traefik sobe e serve o que tem. Isso troca uma janela de ~100 ms por uma indisponibilidade potencialmente
  total, sem opção para desligar o comportamento.
- **Falhas que antes eram toleradas passam a abortar o processo.** `initACMEProvider` agora retorna erro e
  derruba o boot quando um resolver ACME é inválido (antes era só logado e o resolver ignorado), e um erro
  inicial em `UpdateConfigsWithError` (TLS) também aborta o startup. Um único certificado ou resolver ruim
  derruba todas as rotas.
- A lista de providers obrigatórios é parcial (ficam de fora `kubernetesCRD`, `kubernetesIngress` e
  `kubernetesGateway`, como apontado nos comentários), então o problema continua exatamente no cenário mais
  comum de rollout.
- O PR mexe no ciclo de vida do servidor (`Server.Start`, fechamento de listeners não iniciados, listeners
  críticos no configuration watcher), com +686/-34. No upstream está em `status/1-needs-design-review`, sem
  review de maintainer.
- Não encontrei código suspeito.

Motivo do adiamento: a mudança de comportamento padrão é arriscada demais para disponibilidade (sem timeout,
sem opt-out, e erros que viram fatais), e o desenho ainda não foi aprovado no upstream. Reavaliar se o
upstream aceitar a abordagem com timeout configurável ou opt-in e com cobertura dos providers Kubernetes.

Testes: nenhum (não importado).
