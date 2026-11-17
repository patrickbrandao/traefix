# feat: Server HealthCheck Retry Mechanism

| Campo | Valor |
|---|---|
| PR upstream | [traefik/traefik#13387](https://github.com/traefik/traefik/pull/13387) |
| Tipo | Recurso |
| Área | healthcheck/service |
| Prioridade | **P2** |
| Valor | alta |
| Esforço de port | alto |
| Autor | hanxiantao |
| Aberto em | 2026-06-20 (última atualização: 2026-11-10) |
| Draft | não |
| Branch-alvo | `master` |
| Tamanho | +852/-45 em 37 arquivos |
| Labels | status/2-needs-review, area/webui, size/L, area/healthcheck, area/service |
| Status | adiado |

## O que faz

Adiciona os thresholds fails e passes ao active health check (HTTP e TCP). fails define quantas falhas consecutivas marcam o server como unhealthy; passes define quantos sucessos seguidos o recolocam como healthy. Ambos padrão 1, o que preserva o comportamento atual. Atinge file, labels, KV, CRD e WebUI. Reduz a retirada de servers por falha transitória.

## Riscos e situação no upstream

Tamanho L: 37 arquivos, +852 linhas, com CRDs gerados, docs e WebUI. Status 2-needs-review sem review desde a abertura; autor reclama de falta de follow-up. Mudança de schema de configuração, então precisa de cuidado com compatibilidade.

## Principais arquivos

- `docs/content/reference/routing-configuration/http/load-balancing/service.md`
- `docs/content/reference/dynamic-configuration/file.toml`
- `docs/content/reference/dynamic-configuration/kubernetes-crd-definition-v1.yml`
- `docs/content/reference/dynamic-configuration/traefik.io_traefikservices.yaml`
- `docs/content/reference/routing-configuration/other-providers/docker.md`

## Como importar

1. Leia a discussão e os reviews do PR, porque um maintainer pode ter pedido mudanças ou rejeitado a abordagem.
2. Traga o PR para uma branch local: `git fetch upstream pull/13387/head:upstream-pr-13387`. Depois faça cherry-pick dos commits ou merge com squash.
3. Resolva os conflitos dando prioridade às alterações locais do Traefix.
4. Se o PR mexer em código gerado, rode `make generate` ou `make generate-crd`.
5. Rode `make test-unit` e `make lint`. Se for preciso, valide no servidor de testes com `scripts/traefix/deploy-dev.sh`.
6. Quando o PR for mergeado no upstream, marque a task como absorvida, para evitar conflito no próximo sync.

## Revisão (2026-11-16)

**Veredito: adiado.**

O recurso (`fails`/`passes`, como o `max_fails` do nginx) é útil e a lógica central (`status.go`) está correta. Mas o PR é grande (+1982 linhas em 36 arquivos). Ele mexe em CRDs, código gerado do clientset, WebUI e na documentação de todos os providers, e não teve nenhum review do upstream. Importá-lo agora criaria uma API própria (`fails`/`passes`) com grande chance de conflito, se o upstream escolher outros nomes ou outra semântica.

Achados no PR original:

- Logs barulhentos: cada sucesso abaixo do limiar gera um log `Info` ("pass threshold not reached") a cada intervalo. Deveria ser `Debug`.
- Um alvo que falha abaixo do limiar volta para o canal de onde veio. Assim, um alvo "healthy" que começa a falhar continua sendo checado no `interval`, e não no `unhealthyInterval`. É defensável, mas não está documentado.
- O estado dos contadores é perdido a cada reload: todos os alvos voltam a UP com contadores zerados. Com `fails > 1`, um servidor morto continua recebendo tráfego por `(fails-1) × interval` depois de cada reload. Isso piora o problema corrigido pelo #13223 e não está documentado.
- O diff regenera os manifests de CRD e o `applyconfiguration` com controller-gen. Neste ambiente (macOS) `script/code-gen.sh` não roda sem ajuste, porque usa `sed -i` no formato GNU. A WebUI muda em `webui/src`, mas os assets embutidos exigem `make generate-webui` (Docker).
- O loop de checagem conflita com o #12485 (concorrência) e o #13223 (primeira checagem imediata), já importados. `handleHealthCheckResult` teria de ser adaptado ao novo `checkTarget`. O mutex `targetsMu` deixa a parte do estado segura para concorrência.

Reavaliar quando o upstream revisar o PR. Se for importar localmente, adaptar `checkTarget` (HTTP) e o loop TCP, rebaixar o log para `Debug` e documentar o efeito no reload.

Nenhum código importado e nenhum teste rodado.
