# 0012 — Support Gateway API listener filters

- **Status:** aplicada (`48bb2caec`, 2026-11-23)
- **Tipo:** feature
- **Prioridade:** baixa
- **Upstream:** traefik/traefik#14320 — commits `770d8bbf9` (branch v3.8)
- **Aplica limpo:** sim, depois de 0009 e 0010 (sozinho também aplica no texto, mas não compila: os testes usam `NewClient`, `k8sClient` e `BuildConfiguration` da 0010)
- **Conflito com Traefix:** não

## O que mudou no oficial
Provider Kubernetes Gateway: suporte aos filtros de listener (`spec.listeners[].filters`, GEP-5224, canal
Experimental), executados em toda requisição aceita num listener HTTP/HTTPS antes do casamento das rotas. Só o tipo
`ExtensionRef` para um `Middleware` do Traefik é aceito. Exige a opção `experimentalChannel`, que também faz o
`GatewayClass` anunciar a feature `GatewayListenerFilters`; `SupportedFeatures` passa a receber esse booleano
(`features.go`).

**Mudança de comportamento:** antes os filtros de listener eram ignorados. Agora um listener que declara filtros não
é aceito quando eles não podem ser aplicados: sem `experimentalChannel`, com filtro `ExternalAuth` ou com
`ExtensionRef` para um recurso não suportado (nota em `docs/content/migrate/v3.md`). Documentação em
`gateway-api.md`, fixture `listener_filters.yml`, relatórios de conformidade e testes de integração atualizados.

## Impacto no Traefix
Os arquivos do provider Gateway foram alterados só por PRs importados (ver 0009 e 0010). A documentação
`gateway-api.md` e `migrate/v3.md` tem alterações dos imports #13050 e #12921, sem sobreposição. Os chamadores de
`SupportedFeatures` são todos do oficial (`kubernetes.go` e os testes de conformidade em `integration/`).

## Plano de implementação
1. Aplicar antes a 0009 e a 0010.
2. `git cherry-pick -x 770d8bbf9`
3. Testes: `go vet ./pkg/provider/kubernetes/gateway/ ./integration/ && go test ./pkg/provider/kubernetes/gateway/`
   (passou no worktree de teste depois da 0010) e `make lint` (nova exclusão de `funlen` em `loadGatewayListeners`).
4. Se algum ambiente do Traefix usar Gateway API com filtros de listener, revisar a mudança de comportamento antes do
   deploy.

## Critério de pronto
- Filtros `ExtensionRef` → `Middleware` aplicados no listener com `experimentalChannel`; listener recusado nos casos
  documentados.
- Testes do pacote gateway e `make lint` sem erros.
