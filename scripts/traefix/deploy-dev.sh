#!/usr/bin/env bash
# Deploy rápido do Traefix no servidor de testes.
#
# Compila o binário linux/amd64 localmente, envia via rsync (delta + compressão)
# para $DEPLOY_DIR/traefik e reinicia o container, que monta esse arquivo em
# /traefik (bind mount criado por /storage/traefix-app/run.sh no servidor).
#
#   scripts/traefix/deploy-dev.sh            # build + deploy
#   WEBUI=1 scripts/traefix/deploy-dev.sh    # força rebuild da WebUI
#   scripts/traefix/deploy-dev.sh rollback   # volta o binário anterior
#   scripts/traefix/deploy-dev.sh logs       # segue os logs do container
#
# Config em .private/test-server.env (fora do git).
set -euo pipefail

cd "$(git rev-parse --show-toplevel)"
# shellcheck disable=SC1091
source .private/test-server.env

CP="/tmp/traefix-ssh-%C"
SSH_OPTS=(-p "$DEPLOY_PORT" -o ControlMaster=auto -o ControlPath="$CP" -o ControlPersist=10m)
SSH=(ssh "${SSH_OPTS[@]}" "$DEPLOY_USER@$DEPLOY_HOST")
C="$DEPLOY_CONTAINER"
D="$DEPLOY_DIR"

restart() { # $1 = versão esperada (opcional)
  "${SSH[@]}" "set -e
    docker restart -t 5 $C >/dev/null
    for i in \$(seq 1 20); do
      if [ \"\$(docker inspect -f '{{.State.Running}}' $C)\" = true ]; then
        v=\$(docker exec $C /traefik version 2>/dev/null | awk '/^Version/{print \$2}')
        [ -z \"$1\" ] || [ \"\$v\" = \"$1\" ] && { sleep 1; docker inspect -f '{{.State.Running}}' $C | grep -q true && { echo \"OK: \$v\"; exit 0; }; }
      fi
      sleep 0.5
    done
    echo 'FALHOU: container não subiu'; docker logs --tail 30 $C; exit 1"
}

case "${1:-deploy}" in
  logs)
    exec "${SSH[@]}" "docker logs -f --tail 50 $C"
    ;;
  rollback)
    echo ">> rollback para $D/traefik.prev"
    "${SSH[@]}" "cp -f $D/traefik.prev $D/traefik.tmp && mv -f $D/traefik.tmp $D/traefik"
    restart ""
    exit
    ;;
esac

# WebUI (embutida no binário) — só reconstrói se não existir ou WEBUI=1
if [[ "${WEBUI:-0}" == "1" || ! -f webui/static/index.html ]]; then
  echo ">> build WebUI"
  docker buildx build -f Dockerfile.traefix --target webui \
    --output type=local,dest=dist/webui-out . >/dev/null
  find webui/static -mindepth 1 ! -name DONT-EDIT-FILES-IN-THIS-DIRECTORY.md -delete
  cp -R dist/webui-out/src/webui/static/. webui/static/
  rm -rf dist/webui-out
fi

VERSION="dev-$(git rev-parse --short HEAD)$(git diff --quiet HEAD -- . ':!webui/static' || echo -dirty)"
echo ">> go build linux/amd64 ($VERSION)"
mkdir -p dist/deploy
CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -trimpath -ldflags "-s -w \
  -X github.com/traefik/traefik/v3/pkg/version.Version=$VERSION \
  -X github.com/traefik/traefik/v3/pkg/version.Codename=traefix \
  -X github.com/traefik/traefik/v3/pkg/version.BuildDate=$(date -u '+%Y-%m-%d_%H:%M:%S')" \
  -o dist/deploy/traefik ./cmd/traefik

echo ">> rsync"
"${SSH[@]}" "mkdir -p $D && [ ! -f $D/traefik ] || cp -f $D/traefik $D/traefik.prev"
rsync -az -e "ssh ${SSH_OPTS[*]}" dist/deploy/traefik "$DEPLOY_USER@$DEPLOY_HOST:$D/traefik"

echo ">> restart $C"
restart "$VERSION"
