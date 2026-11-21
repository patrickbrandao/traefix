#!/usr/bin/env bash
# Checks the ACME certificates of a Traefix cluster with the real binary, a Redis (or Valkey) server and Pebble as the CA.
# Nothing runs in Docker: the script needs go, redis-server (or valkey-server) with its CLI, pebble, openssl and curl,
# and a resolver that answers 127.0.0.1 for the names under .localhost (macOS, systemd-resolved).
#
#   scripts/traefix/e2e-cluster.sh              # every scenario
#   scripts/traefix/e2e-cluster.sh replicas kill # only the named scenarios
#   KEEP=1 scripts/traefix/e2e-cluster.sh       # keeps the working directory, with the logs
#   LOG_LEVEL=DEBUG scripts/traefix/e2e-cluster.sh
#
# Install Pebble with: go install github.com/letsencrypt/pebble/v2/cmd/pebble@latest

set -u

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
WORK=${WORK:-$(mktemp -d "${TMPDIR:-/tmp}/traefix-e2e.XXXXXX")}

REDIS_PORT=16399
PEBBLE_PORT=14000
PEBBLE_MGMT_PORT=15000
# A port nobody listens on, for the validations that must fail.
CLOSED_PORT=18999

REDIS_SERVER=${REDIS_SERVER:-$(command -v valkey-server || command -v redis-server)}
REDIS_CLI=${REDIS_CLI:-$(command -v valkey-cli || command -v redis-cli)}
PEBBLE=${PEBBLE:-$(command -v pebble || echo "$(go env GOPATH 2> /dev/null)/bin/pebble")}

FAILURES=0
CHECKS=0

for bin in go openssl curl "$REDIS_SERVER" "$REDIS_CLI" "$PEBBLE"; do
  if [ -z "$bin" ] || ! command -v "$bin" > /dev/null 2>&1; then
    echo "Missing a tool the test needs (go, openssl, curl, redis-server or valkey-server, their CLI, pebble): '$bin'" >&2
    exit 2
  fi
done

rcli() {
  "$REDIS_CLI" -p "$REDIS_PORT" "$@"
}

ok() {
  CHECKS=$((CHECKS + 1))
  echo "  ok    $1"
}

ko() {
  CHECKS=$((CHECKS + 1))
  FAILURES=$((FAILURES + 1))
  echo "  FAIL  $1"
}

# check <description> <command...>: the command succeeds.
check() {
  local desc=$1
  shift
  if "$@" > /dev/null 2>&1; then ok "$desc"; else ko "$desc"; fi
}

# equal <description> <expected> <actual>
equal() {
  if [ "$2" = "$3" ]; then ok "$1"; else ko "$1 (expected '$2', got '$3')"; fi
}

# wait_until <seconds> <command...>: the command succeeds before the time is over.
wait_until() {
  local deadline=$(($(date +%s) + $1))
  shift
  until "$@" > /dev/null 2>&1; do
    [ "$(date +%s)" -ge "$deadline" ] && return 1
    sleep 0.5
  done
}

# end <pid> [signal]: stops a process of the test and waits for it, without the report of the shell.
end() {
  kill "-${2:-TERM}" "$1" 2> /dev/null
  wait "$1" 2> /dev/null
  return 0
}

start_redis() {
  "$REDIS_SERVER" --port "$REDIS_PORT" --bind 127.0.0.1 --save "" --appendonly no --dir "$WORK" > "$WORK/redis.log" 2>&1 &
  echo $! > "$WORK/redis.pid"
  wait_until 10 rcli ping
}

stop_redis() {
  [ -f "$WORK/redis.pid" ] && end "$(cat "$WORK/redis.pid")"
  rm -f "$WORK/redis.pid"
}

# start_pebble <port of the HTTP-01 validations> [validity of the certificates in seconds]
start_pebble() {
  stop_pebble
  cat > "$WORK/pebble.json" << EOF
{
  "pebble": {
    "listenAddress": "127.0.0.1:$PEBBLE_PORT",
    "managementListenAddress": "127.0.0.1:$PEBBLE_MGMT_PORT",
    "certificate": "$WORK/pebble.crt",
    "privateKey": "$WORK/pebble.key",
    "httpPort": $1,
    "tlsPort": 18998,
    "ocspResponderURL": "",
    "externalAccountBindingRequired": false,
    "profiles": {
      "default": {
        "description": "The profile of the test",
        "validityPeriod": ${2:-7776000}
      }
    }
  }
}
EOF
  PEBBLE_VA_NOSLEEP=1 PEBBLE_WFE_NONCEREJECT=0 PEBBLE_AUTHZREUSE=0 "$PEBBLE" -config "$WORK/pebble.json" > "$WORK/pebble.log" 2>&1 &
  echo $! > "$WORK/pebble.pid"
  wait_until 10 curl -sk "https://127.0.0.1:$PEBBLE_PORT/dir"
}

stop_pebble() {
  [ -f "$WORK/pebble.pid" ] && end "$(cat "$WORK/pebble.pid")"
  rm -f "$WORK/pebble.pid"
}

issued() {
  grep -c "Issued certificate serial" "$WORK/pebble.log"
}

# routers <instance> <domain...>: the routers the file provider of an instance sees.
routers() {
  local n=$1
  shift
  {
    echo "http:"
    if [ $# -gt 0 ]; then
      echo "  routers:"
      for domain in "$@"; do
        echo "    r-${domain%%.*}:"
        echo "      rule: \"Host(\`$domain\`)\""
        echo "      service: app"
        echo "      tls:"
        echo "        certResolver: le"
      done
    fi
    echo "  services:"
    echo "    app:"
    echo "      loadBalancer:"
    echo "        servers:"
    echo "          - url: \"http://127.0.0.1:9\""
  } > "$WORK/n$n/dynamic.yml.tmp"
  mv "$WORK/n$n/dynamic.yml.tmp" "$WORK/n$n/dynamic.yml"
}

# Settings of the next mk calls.
MK_CLUSTER=1
MK_WEB=""
MK_DURATION=2160
MK_DELAY=0
MK_LOCK_TTL=10s

# mk <instance> <priority>: writes the install configuration of an instance.
mk() {
  local n=$1 prio=$2
  local web=${MK_WEB:-1808$n}
  mkdir -p "$WORK/n$n"
  [ -f "$WORK/n$n/dynamic.yml" ] || routers "$n"
  {
    cat << EOF
entryPoints:
  web:
    address: ":$web"
  websecure:
    address: ":1844$n"
  metrics:
    address: "127.0.0.1:1809$n"
log:
  level: ${LOG_LEVEL:-INFO}
metrics:
  prometheus:
    entryPoint: metrics
providers:
  file:
    filename: $WORK/n$n/dynamic.yml
EOF
    if [ "$MK_CLUSTER" = 1 ]; then
      cat << EOF
cluster:
  nodeID: srv$n
  priority: $prio
  heartbeatInterval: 1s
  nodeTimeout: 3s
  redis:
    endpoints:
      - "127.0.0.1:$REDIS_PORT"
EOF
    fi
    cat << EOF
certificatesResolvers:
  le:
    acme:
      email: ops@example.com
      caServer: https://127.0.0.1:$PEBBLE_PORT/dir
      caCertificates:
        - $WORK/pebble.crt
      certificatesDuration: $MK_DURATION
      storage: $WORK/n$n/acme.json
      httpChallenge:
        entryPoint: web
        delay: $MK_DELAY
EOF
    if [ "$MK_CLUSTER" = 1 ]; then
      cat << EOF
      certStore:
        primary: redis
        dir: $WORK/n$n/certs
        export:
          - dir
        election:
          step: 2s
          lockTTL: $MK_LOCK_TTL
EOF
    fi
  } > "$WORK/n$n/traefik.yml"
}

# start <instance> [log name]
start() {
  "$WORK/traefik" --configFile="$WORK/n$1/traefik.yml" >> "$WORK/n$1/${2:-out}.log" 2>&1 &
  echo $! > "$WORK/n$1/pid"
}

# stop <instance> [signal]
stop() {
  [ -f "$WORK/n$1/pid" ] || return 0
  end "$(cat "$WORK/n$1/pid")" "${2:-TERM}"
  rm -f "$WORK/n$1/pid"
}

stop_all() {
  for n in 1 2 3; do stop "$n"; done
}

# reset: stops the instances and forgets what they and the Redis server hold.
reset() {
  stop_all
  rcli flushall > /dev/null 2>&1
  rm -rf "$WORK"/n1 "$WORK"/n2 "$WORK"/n3
  MK_CLUSTER=1
  MK_WEB=""
  MK_DURATION=2160
  MK_DELAY=0
  MK_LOCK_TTL=10s
}

# logged <instance> <text> [log name]: how many lines of the log of an instance have the text.
logged() {
  sed $'s/\x1b\\[[0-9;]*m//g' "$WORK/n$1/${3:-out}.log" 2> /dev/null | grep -c "$2"
}

has_logged() {
  [ "$(logged "$@")" -gt 0 ]
}

# served <instance> <domain>: the issuer and the serial of the certificate an instance serves for a domain.
served() {
  echo | openssl s_client -connect "127.0.0.1:1844$1" -servername "$2" 2> /dev/null | openssl x509 -noout -issuer -serial 2> /dev/null | tr '\n' ' '
}

serial() {
  served "$1" "$2" | sed -n 's/.*serial=\([0-9A-Fa-f]*\).*/\1/p'
}

serves_ca() {
  served "$1" "$2" | grep -q "Pebble"
}

# serves_other <instance> <domain> <serial>: the instance serves a certificate of the CA that is not this one.
serves_other() {
  local current
  current=$(serial "$1" "$2")
  serves_ca "$1" "$2" && [ -n "$current" ] && [ "$current" != "$3" ]
}

# metric <instance> <name with labels>: the value of a metric of an instance.
metric() {
  curl -s "http://127.0.0.1:1809$1/metrics" | grep -F "$2" | grep -v '^#' | awk '{print $NF}' | head -1
}

has_failure() {
  [ -n "$(rcli keys 'traefix:acme:{le}:fail:*')" ]
}

has_certs() {
  [ "$(rcli scard 'traefix:acme:{le}:certs')" -ge 1 ]
}

requests() {
  local total=0
  for n in "$@"; do total=$((total + $(logged "$n" "This instance requests the certificate"))); done
  echo "$total"
}

scenario_replicas() {
  echo "== Three replicas behind one name: one request to the CA, answered by any replica"
  reset
  start_pebble 18083
  for n in 1 2 3; do mkdir -p "$WORK/n$n" && routers "$n" replicas.localhost; done
  mk 1 100 && mk 2 50 && mk 3 50
  start 3 && start 2 && start 1

  for n in 1 2 3; do
    check "srv$n serves the certificate of the CA" wait_until 40 serves_ca "$n" replicas.localhost
  done

  equal "only the instance with the highest priority asked the CA" "1 0" "$(requests 1) $(requests 2 3)"
  equal "the CA issued one certificate" 1 "$(issued)"
  equal "the three instances serve the same certificate" "$(serial 1 replicas.localhost)" "$(serial 2 replicas.localhost)"
  equal "the three instances serve the same certificate (srv3)" "$(serial 1 replicas.localhost)" "$(serial 3 replicas.localhost)"
  check "srv3 answered the validation of a challenge it did not start" \
    test "$(metric 3 'traefik_acme_cluster_challenge_lookups_total{result="redis"}')" -ge 1

  local dir="$WORK/n2/certs/replicas.localhost"
  for file in fullchain.pem cert.pem chain.pem privkey.pem cert.md5 cert.json; do
    check "srv2 wrote $file" test -s "$dir/$file"
  done
  check "cert.json names the domain" grep -q '"replicas.localhost"' "$dir/cert.json"
  equal "the traefik-certs copy is in the Redis server" 1 "$(rcli exists tcerts:cert:replicas.localhost)"
  equal "the answer of the challenge was removed" "" "$(rcli keys 'traefix:acme:http:*')"

  equal "metric: three instances" 3 "$(metric 1 traefik_cluster_nodes)"
  equal "metric: the Redis server is up" 1 "$(metric 1 traefik_cluster_redis_up)"
  equal "metric: one certificate obtained by srv1" 1 \
    "$(metric 1 'traefik_acme_cluster_issuances_total{kind="obtain",resolver="le",result="success"}')"
  check "metric: srv2 wrote its directory" \
    test "$(metric 2 'traefik_acme_store_writes_total{resolver="le",result="success",target="dir"}')" -ge 1

  echo "== The instance with the highest priority is gone: one of the others asks"
  stop 1
  sleep 4
  for n in 2 3; do routers "$n" replicas.localhost second.localhost; done

  for n in 2 3; do
    check "srv$n serves the new certificate" wait_until 40 serves_ca "$n" second.localhost
  done
  equal "one of the two instances asked the CA" 1 "$(requests 2 3)"
  equal "the CA issued two certificates in all" 2 "$(issued)"
}

scenario_zero() {
  echo "== Every instance has the priority 0: no request, and the log says why"
  reset
  start_pebble 18081
  for n in 1 2; do mkdir -p "$WORK/n$n" && routers "$n" zero.localhost; done
  mk 1 0 && mk 2 0
  start 1 && start 2

  check "an instance warns that nobody requests the certificates" \
    wait_until 20 has_logged 1 "No instance of the cluster requests the certificates"
  equal "no instance asked the CA" 0 "$(requests 1 2)"
  equal "the CA issued nothing" 0 "$(issued)"
}

scenario_kill() {
  echo "== The elected instance is killed during the request: another one takes over"
  reset
  start_pebble 18082
  for n in 1 2; do mkdir -p "$WORK/n$n" && routers "$n" kill.localhost; done
  # The delay keeps the request of srv1 open long enough to kill it in the middle.
  MK_DELAY=20s mk 1 100
  mk 2 50
  start 1 && start 2

  check "srv1 takes the request" wait_until 20 has_logged 1 "This instance requests the certificate"
  stop 1 KILL

  check "srv2 serves the certificate" wait_until 60 serves_ca 2 kill.localhost
  equal "srv2 asked the CA after srv1" 1 "$(requests 2)"
  equal "the CA issued one certificate" 1 "$(issued)"
}

scenario_failure() {
  echo "== The validation fails: the failure is shared, and nobody tries again at once"
  reset
  start_pebble "$CLOSED_PORT"
  for n in 1 2; do mkdir -p "$WORK/n$n" && routers "$n" failure.localhost; done
  mk 1 100 && mk 2 50
  start 1 && start 2

  check "the failure is recorded in the Redis server" wait_until 40 has_failure

  local record
  record=$(rcli get "$(rcli keys 'traefix:acme:{le}:fail:*' | head -1)")
  check "the record names srv1 and its first attempt" \
    sh -c "echo '$record' | grep -q '\"node\":\"srv1\"' && echo '$record' | grep -q '\"attempts\":1'"

  sleep 12
  equal "no instance tried again" 1 "$(requests 1 2)"
  equal "the CA issued nothing" 0 "$(issued)"
  equal "metric: one failed request" 1 \
    "$(metric 1 'traefik_acme_cluster_issuances_total{kind="obtain",resolver="le",result="failure"}')"
}

scenario_lone() {
  echo "== Only one instance has the router: the elected one obtains the certificate for it"
  reset
  start_pebble 18082
  mkdir -p "$WORK/n1" "$WORK/n2"
  routers 1
  routers 2 lone.localhost
  mk 1 100 && mk 2 50
  start 1 && start 2

  check "srv2 serves the certificate" wait_until 40 serves_ca 2 lone.localhost
  equal "srv1, which has no router for it, asked the CA" "1 0" "$(requests 1) $(requests 2)"
  check "srv1 holds the certificate in its directory" test -s "$WORK/n1/certs/lone.localhost/privkey.pem"

  echo "== The Redis server stops: the instances keep serving, and start from their files"
  local before
  before=$(serial 2 lone.localhost)
  stop_redis
  sleep 2
  equal "srv2 still serves the certificate" "$before" "$(serial 2 lone.localhost)"

  stop 2
  sleep 1
  start 2 restart
  check "srv2 restarted without the Redis server serves the certificate" wait_until 30 serves_ca 2 lone.localhost
  equal "it is the same certificate" "$before" "$(serial 2 lone.localhost)"
  check "metric: the Redis server is down" wait_until 10 test "$(metric 2 traefik_cluster_redis_up)" = 0

  echo "== The Redis server is back without its data: the instances give the certificates back"
  start_redis
  check "the certificate is in the Redis server again" wait_until 40 has_certs
  equal "the CA issued one certificate in all" 1 "$(issued)"
}

scenario_import() {
  echo "== Two instances with their own storage file join a cluster: nothing is asked to the CA"
  reset
  # Every instance answers the validations on the same port, one after the other.
  start_pebble 18080 3600

  for n in 1 2; do
    mkdir -p "$WORK/n$n" && routers "$n" "import$n.localhost"
    MK_CLUSTER=0 MK_WEB=18080 MK_DURATION=1 mk "$n" 100
    start "$n" alone
    check "srv$n alone obtains its certificate" wait_until 40 serves_ca "$n" "import$n.localhost"
    stop "$n"
    sleep 1
  done
  equal "the CA issued two certificates, for two accounts" 2 "$(issued)"

  # In the cluster, the validations reach srv2, and srv1 is the one that asks the CA.
  local first second
  for n in 1 2; do routers "$n" import1.localhost import2.localhost; done
  MK_DURATION=1 mk 1 100
  MK_DURATION=1 MK_WEB=18080 mk 2 50
  start 1 && start 2

  for n in 1 2; do
    for domain in import1.localhost import2.localhost; do
      check "srv$n serves $domain" wait_until 30 serves_ca "$n" "$domain"
    done
  done
  equal "the CA issued nothing more" 2 "$(issued)"
  equal "the instances share one account" 1 "$(rcli exists 'traefix:acme:{le}:account')"
  first=$(serial 1 import1.localhost)
  second=$(serial 1 import2.localhost)

  echo "== The certificates are due: the highest priority renews them, even the one of the other account"
  stop_all
  MK_DURATION=24 mk 1 100
  MK_DURATION=24 MK_WEB=18080 mk 2 50
  start 1 renew && start 2 renew

  for n in 1 2; do
    check "srv$n serves a renewed import1.localhost" wait_until 60 serves_other "$n" import1.localhost "$first"
    check "srv$n serves a renewed import2.localhost" wait_until 60 serves_other "$n" import2.localhost "$second"
  done
  equal "the CA issued two renewals" 4 "$(issued)"
  equal "srv2 renewed nothing" 0 "$(logged 2 "Renewing ACME certificate" renew)"
  check "the CA refused to replace the certificate of the other account, which was renewed as a new one" \
    has_logged 1 "refuses to replace the certificate" renew
  equal "metric: two renewals by srv1" 2 \
    "$(metric 1 'traefik_acme_cluster_issuances_total{kind="renew",resolver="le",result="success"}')"
}

cleanup() {
  stop_all 2> /dev/null
  stop_pebble
  stop_redis
  if [ "${KEEP:-0}" = 1 ] || [ "$FAILURES" -gt 0 ]; then
    echo "Logs kept in $WORK"
  else
    rm -rf "$WORK"
  fi
}
trap cleanup EXIT

echo "Building the binary..."
(cd "$ROOT" && go build -o "$WORK/traefik" ./cmd/traefik) || exit 2

openssl req -x509 -newkey rsa:2048 -nodes -keyout "$WORK/pebble.key" -out "$WORK/pebble.crt" -days 2 \
  -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" 2> /dev/null || exit 2

start_redis || {
  echo "The Redis server did not start" >&2
  exit 2
}

[ $# -eq 0 ] && set -- replicas zero kill failure lone import

for name in "$@"; do
  if ! declare -F "scenario_$name" > /dev/null; then
    echo "Unknown scenario: $name" >&2
    exit 2
  fi
  "scenario_$name"
done

echo
echo "$((CHECKS - FAILURES)) of $CHECKS checks passed"
[ "$FAILURES" -eq 0 ]
