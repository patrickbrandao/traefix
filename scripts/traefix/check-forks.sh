#!/usr/bin/env bash
# Confere os forks traefix-* do go.mod: cada replace para github.com/patrickbrandao/traefix-<nome> tem de estar
# na mesma versão do upstream que o require do módulo original. Sem isso, um bump do upstream no require (por
# exemplo vindo do Traefik oficial) seria anulado em silêncio pelo replace, que continuaria na base antiga.
#
#   scripts/traefix/check-forks.sh          # sai com 1 se algum fork estiver fora de sincronia
#   scripts/traefix/check-forks.sh -v       # também lista os forks em dia
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"

verbose=0
[[ "${1:-}" == "-v" ]] && verbose=1

status=0
while read -r orig fork forkVersion; do
  base="${forkVersion%-traefix.*}"
  # O require vem direto do go.mod: go list devolveria a versão do replace.
  required="$(awk -v m="$orig" '$1 == m && $2 ~ /^v/ { print $2; exit }' go.mod)"
  if [[ -z "$required" ]]; then
    echo "ERRO: $orig tem replace para $fork mas não aparece no require" >&2
    status=1
  elif [[ "$required" != "$base" ]]; then
    echo "FORA DE SINCRONIA: $orig require $required, fork $fork $forkVersion (base $base)" >&2
    echo "  -> rebasear o branch traefix de $fork em $required e marcar $required-traefix.1" >&2
    status=1
  elif [[ "$verbose" == 1 ]]; then
    echo "ok: $orig $required -> $fork $forkVersion"
  fi
done < <(awk '$2 == "=>" && $3 ~ /^github.com\/patrickbrandao\/traefix-/ { print $1, $3, $4 }' go.mod)

exit "$status"
