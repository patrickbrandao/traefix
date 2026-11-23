#!/usr/bin/env bash
# Checks the mesh of a Traefix cluster with the real binary: three instances on loopback, each with a file provider
# and test servers written in Python. Nothing runs in Docker: the script needs go, python3, openssl and curl.
#
#   scripts/traefix/e2e-mesh.sh                # every scenario
#   scripts/traefix/e2e-mesh.sh http tcp       # only the named scenarios
#   TRANSPORT=tcp scripts/traefix/e2e-mesh.sh  # the links over TLS over TCP (default: auto, which is QUIC here)
#   KEEP=1 scripts/traefix/e2e-mesh.sh         # keeps the working directory, with the logs
#   TRAEFIK=./traefik scripts/traefix/e2e-mesh.sh  # uses a binary already built
#   LOG_LEVEL=DEBUG scripts/traefix/e2e-mesh.sh
#
# Scenarios: formation token http priority tcp udp fallback config satellite satellite-tcp satellite-certs challenge
#
# The scenarios satellite-certs and challenge need Pebble as the CA: go install github.com/letsencrypt/pebble/v2/cmd/pebble@latest

set -u

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
HELPERS="$ROOT/scripts/traefix/e2e-mesh"
WORK=${WORK:-$(mktemp -d "${TMPDIR:-/tmp}/traefix-e2e-mesh.XXXXXX")}
TRANSPORT=${TRANSPORT:-auto}

PEBBLE=${PEBBLE:-$(command -v pebble || echo "$(go env GOPATH 2> /dev/null)/bin/pebble")}
PEBBLE_PORT=14000

TOKEN_A=aaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaaa
TOKEN_B=bbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbbb

FAILURES=0
CHECKS=0

for bin in go python3 openssl curl; do
  if ! command -v "$bin" > /dev/null 2>&1; then
    echo "Missing a tool the test needs (go, python3, openssl, curl): '$bin'" >&2
    exit 2
  fi
done

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

# eventually <description> <seconds> <command...>
eventually() {
  local desc=$1 seconds=$2
  shift 2
  if wait_until "$seconds" "$@"; then ok "$desc"; else ko "$desc"; fi
}

# end <pid> [signal]: stops a process of the test and waits for it, without the report of the shell.
end() {
  kill "-${2:-TERM}" "$1" 2> /dev/null
  wait "$1" 2> /dev/null
  return 0
}

# Settings of the next mk calls.
MK_TOKEN=$TOKEN_A
MK_ADDITIONAL=""
MK_TRANSPORT=$TRANSPORT
MK_PEERS="[]"
MK_PEERS_FILE=0
MK_EXTRA_MIDDLEWARE=0
MK_SATELLITE=0
MK_RESOLVER=0
# The CA may resolve a name under .localhost to ::1: the entry point of its validations listens on every address then.
MK_WEB_HOST=127.0.0.1

# mk <instance>: writes the install configuration of an instance.
mk() {
  local n=$1
  mkdir -p "$WORK/n$n"
  [ -f "$WORK/n$n/dynamic.yml" ] || dyn "$n"
  {
    cat << EOF
entryPoints:
  web:
    address: "${MK_WEB_HOST}:1808$n"
    http:
      middlewares:
        - entry@file
EOF
    [ "$MK_EXTRA_MIDDLEWARE" = 1 ] && echo "        - extra@file"
    cat << EOF
  tcp:
    address: "127.0.0.1:1700$n"
  tls:
    address: "127.0.0.1:1710$n"
  dns:
    address: "127.0.0.1:1530$n/udp"
    udp:
      timeout: 2s
  traefik:
    address: "127.0.0.1:1809$n"
api:
  insecure: true
metrics:
  prometheus:
    entryPoint: traefik
log:
  level: ${LOG_LEVEL:-INFO}
providers:
  file:
    filename: $WORK/n$n/dynamic.yml
cluster:
  nodeID: srv$n
  token: $MK_TOKEN
EOF
    [ -n "$MK_ADDITIONAL" ] && printf '  additionalTokens:\n    - %s\n' "$MK_ADDITIONAL"
    cat << EOF
  mesh:
    address: "127.0.0.1:744$n"
    transport: $MK_TRANSPORT
    linkTimeout: 3s
    peers: $MK_PEERS
EOF
    [ "$MK_SATELLITE" = 1 ] && echo "    satellite: true"
    [ "$MK_PEERS_FILE" = 1 ] && echo "    peersFile: $WORK/n$n/peers.json"
    if [ "$MK_RESOLVER" = 1 ]; then
      cat << EOF
certificatesResolvers:
  le:
    acme:
      email: ops@example.com
      caServer: https://127.0.0.1:$PEBBLE_PORT/dir
      caCertificates:
        - $WORK/pebble.crt
      storage: $WORK/n$n/acme.json
      httpChallenge:
        entryPoint: web
EOF
    fi
  } > "$WORK/n$n/traefik.yml"
}

# dyn <instance> [http=<priority>] [tcp=<priority>] [udp=<priority>] [own-udp]: writes the routing configuration.
# The routers use the servers of the instance; own-udp is a UDP router of its own that the instance does not announce.
dyn() {
  local n=$1 http="" tcp="" udp="" own=0
  shift
  for arg in "$@"; do
    case $arg in
      http=*) http=${arg#http=} ;;
      tcp=*) tcp=${arg#tcp=} ;;
      udp=*) udp=${arg#udp=} ;;
      own-udp) own=1 ;;
    esac
  done
  mkdir -p "$WORK/n$n"
  {
    cat << EOF
http:
  middlewares:
    entry:
      headers:
        customRequestHeaders:
          X-Entry: "srv$n"
    extra:
      headers:
        customRequestHeaders:
          X-Extra: "srv$n"
    own:
      headers:
        customRequestHeaders:
          X-Own: "srv$n"
EOF
    if [ -n "$http" ]; then
      cat << EOF
  routers:
    app:
      rule: "Host(\`app.localhost\`)"
      entryPoints: [web]
      middlewares: [own]
      service: app
      mesh:
        priority: $http
  services:
    app:
      loadBalancer:
        healthCheck:
          path: /
          interval: 1s
          timeout: 1s
        servers:
          - url: "http://127.0.0.1:1900$n"
EOF
    fi
    if [ -n "$tcp" ]; then
      cat << EOF
tcp:
  middlewares:
    onlyten:
      ipAllowList:
        sourceRange: ["10.0.0.0/8"]
  serversTransports:
    pp:
      proxyProtocol:
        version: 1
  routers:
    plain:
      rule: "HostSNI(\`*\`)"
      entryPoints: [tcp]
      service: plain
      mesh:
        priority: $tcp
    pass:
      rule: "HostSNI(\`pass.localhost\`)"
      entryPoints: [tls]
      service: pass
      tls:
        passthrough: true
    term:
      rule: "HostSNI(\`term.localhost\`)"
      entryPoints: [tls]
      service: term
      tls: {}
    denied:
      rule: "HostSNI(\`denied.localhost\`)"
      entryPoints: [tls]
      middlewares: [onlyten]
      service: term
      tls: {}
  services:
    plain:
      loadBalancer:
        serversTransport: pp
        servers:
          - address: "127.0.0.1:1950$n"
    pass:
      loadBalancer:
        servers:
          - address: "127.0.0.1:1951$n"
    term:
      loadBalancer:
        servers:
          - address: "127.0.0.1:1952$n"
EOF
    fi
    if [ -n "$udp" ] || [ "$own" = 1 ]; then
      echo "udp:"
      echo "  routers:"
      if [ -n "$udp" ]; then
        cat << EOF
    dns:
      entryPoints: [dns]
      service: dns
      mesh:
        priority: $udp
EOF
      fi
      if [ "$own" = 1 ]; then
        cat << EOF
    own:
      entryPoints: [dns]
      service: own
      mesh:
        export: false
EOF
      fi
      echo "  services:"
      if [ -n "$udp" ]; then
        printf '    dns:\n      loadBalancer:\n        servers:\n          - address: "127.0.0.1:1970%s"\n' "$n"
      fi
      if [ "$own" = 1 ]; then
        printf '    own:\n      loadBalancer:\n        servers:\n          - address: "127.0.0.1:19709"\n'
      fi
    fi
  } > "$WORK/n$n/dynamic.yml.tmp"
  mv "$WORK/n$n/dynamic.yml.tmp" "$WORK/n$n/dynamic.yml"
}

start() {
  local n=$1
  "$WORK/traefik" --configFile="$WORK/n$n/traefik.yml" >> "$WORK/n$n/out.log" 2>&1 &
  echo $! > "$WORK/n$n/pid"
  wait_until 20 curl -sf "http://127.0.0.1:1809$n/api/mesh"
}

stop() {
  [ -f "$WORK/n$1/pid" ] && end "$(cat "$WORK/n$1/pid")" "${2:-TERM}"
  rm -f "$WORK/n$1/pid"
}

# server <kind> <instance>: starts a test server of the instance (http, tcp, udp or own-udp).
server() {
  local kind=$1 n=$2
  case $kind in
    http) python3 "$HELPERS/http_backend.py" "backend$n" "1900$n" > /dev/null 2>&1 & ;;
    tcp)
      python3 "$HELPERS/tcp_backend.py" "plain$n" "1950$n" --proxy > /dev/null 2>&1 &
      echo $! >> "$WORK/servers.pid"
      python3 "$HELPERS/tcp_backend.py" "pass$n" "1951$n" --tls "$WORK/server.crt" "$WORK/server.key" > /dev/null 2>&1 &
      echo $! >> "$WORK/servers.pid"
      python3 "$HELPERS/tcp_backend.py" "term$n" "1952$n" > /dev/null 2>&1 &
      ;;
    udp) python3 "$HELPERS/udp_backend.py" "dns$n" "1970$n" > /dev/null 2>&1 & ;;
    own-udp) python3 "$HELPERS/udp_backend.py" "own$n" 19709 > /dev/null 2>&1 & ;;
  esac
  echo $! >> "$WORK/servers.pid"
  echo $! > "$WORK/server-$kind-$n.pid"
  sleep 0.3
}

stop_server() {
  [ -f "$WORK/server-$1-$2.pid" ] && end "$(cat "$WORK/server-$1-$2.pid")"
  rm -f "$WORK/server-$1-$2.pid"
}

stop_all() {
  for n in 1 2 3; do stop "$n"; done
  if [ -f "$WORK/servers.pid" ]; then
    while read -r pid; do end "$pid"; done < "$WORK/servers.pid"
  fi
  rm -f "$WORK/servers.pid" "$WORK"/server-*.pid
}

# keep_logs: moves the instances of the scenario that ran last aside, with their logs.
keep_logs() {
  [ -n "${SCENARIO:-}" ] || return 0
  for log in "$WORK"/n[0-9]/out.log; do
    if [ -f "$log" ] && grep -q "^panic:" "$log"; then
      ko "no instance crashed in the scenario $SCENARIO (a panic is in $SCENARIO/$(basename "$(dirname "$log")")/out.log)"
    fi
  done
  mkdir -p "$WORK/$SCENARIO"
  for dir in "$WORK"/n[0-9]; do
    [ -d "$dir" ] && mv "$dir" "$WORK/$SCENARIO/"
  done
  return 0
}

reset() {
  stop_all
  keep_logs
  MK_TOKEN=$TOKEN_A
  MK_ADDITIONAL=""
  MK_TRANSPORT=$TRANSPORT
  MK_PEERS="[]"
  MK_PEERS_FILE=0
  MK_EXTRA_MIDDLEWARE=0
  MK_SATELLITE=0
  MK_RESOLVER=0
  MK_WEB_HOST=127.0.0.1
}

# mesh <instance> <python expression on the answer of /api/mesh, as a>: prints the value.
mesh() {
  curl -s -m 5 "http://127.0.0.1:1809$1/api/mesh" | python3 -c "
import json, sys
a = json.load(sys.stdin)
print($2)" 2> /dev/null
}

links_up() {
  mesh "$1" "sum(1 for l in a['links'] if l['up'])"
}

# links_are <instance> <count>
links_are() {
  [ "$(links_up "$1")" = "$2" ]
}

# metric <instance> <name> <label=value,...>: prints the value of the series with these labels, or nothing.
metric() {
  curl -s -m 5 "http://127.0.0.1:1809$1/metrics" | python3 -c "
import re, sys
name, wanted = sys.argv[1], dict(kv.split('=', 1) for kv in sys.argv[2].split(',') if kv)
for line in sys.stdin:
    m = re.match(r'^(\w+)(?:\{(.*)\})? (\S+)$', line.strip())
    if not m or m.group(1) != name:
        continue
    labels = dict(re.findall(r'(\w+)=\"([^\"]*)\"', m.group(2) or ''))
    if labels == wanted:
        print(m.group(3).replace('.0', '') if m.group(3).endswith('.0') else m.group(3))
        break" "$2" "$3"
}

# via <instance> [field...]: requests app.localhost through an instance, and prints what the server saw.
via() {
  local n=$1
  shift
  curl -s -m 5 -H 'Host: app.localhost' "http://127.0.0.1:1808$n/" | python3 -c "
import json, sys
fields = sys.argv[1:] or ['backend']
try:
    d = json.load(sys.stdin)
    h = d['headers']
    values = {'backend': d['backend'], 'entry': h.get('x-entry'), 'own': h.get('x-own'), 'xff': h.get('x-forwarded-for')}
    print(' '.join(str(values[f]) for f in fields))
except Exception:
    print('none')" "$@"
}

# via_is <instance> <backend>
via_is() {
  [ "$(via "$1")" = "$2" ]
}

status_of() {
  curl -s -o /dev/null -m 5 -w '%{http_code}' -H 'Host: app.localhost' "http://127.0.0.1:1808$1/"
}

status_is() {
  [ "$(status_of "$1")" = "$2" ]
}

tcp_client() {
  python3 "$HELPERS/tcp_client.py" "$@"
}

udp_client() {
  python3 "$HELPERS/udp_client.py" "$@"
}

# tcp_imported_are <instance> <count>: the instance imported that many TCP routers.
tcp_imported_are() {
  [ "$(mesh "$1" "sum(1 for r in a['imported'] if r['kind'] == 'tcp' and r['status'] == 'imported')")" = "$2" ]
}

# tcp_answers <port> <server>: a connection to the port reaches the server.
tcp_answers() {
  [ "$(tcp_client "$1" | cut -d'|' -f2 | awk '{print $1}')" = "$2" ]
}

udp_server_is() {
  [ "$(udp_client "$1" 100 | cut -d'|' -f2 | awk '{print $1}')" = "$2" ]
}

scenario_formation() {
  echo "Scenario formation: the instances find each other from one address, and again from the file of the peers"
  reset
  MK_PEERS="[]" mk 1
  MK_PEERS='["127.0.0.1:7441"]' mk 2
  MK_PEERS='["127.0.0.1:7441"]' MK_PEERS_FILE=1 mk 3
  for n in 1 2 3; do start "$n"; done

  for n in 1 2 3; do
    eventually "srv$n links to the two other instances" 20 links_are "$n" 2
  done

  eventually "srv3 writes the instances it learned to its file" 10 grep -q srv2 "$WORK/n3/peers.json"

  # A link over QUIC leaves from the port of the mesh, the one a satellite is reached back through.
  if [ "$TRANSPORT" != tcp ]; then
    equal "the links of srv2 over QUIC leave from its mesh port" "127.0.0.1:7442" \
      "$(mesh 1 "' '.join(i['remote'] for i in a['inbound'] if i['nodeID'] == 'srv2')")"
  fi

  # Without the list, the instance finds the others again from its file.
  stop 3
  MK_PEERS="[]" MK_PEERS_FILE=1 mk 3
  start 3
  eventually "srv3, restarted without peers, links to both from its file" 20 links_are 3 2
}

scenario_token() {
  echo "Scenario token: an instance with another token is refused, and the token changes in three steps"
  reset
  MK_PEERS='["127.0.0.1:7442"]' mk 1
  MK_TOKEN=$TOKEN_B MK_PEERS='["127.0.0.1:7441"]' mk 2
  start 1
  start 2
  sleep 6
  equal "srv1 does not link to the instance of another token" 0 "$(links_up 1)"
  check "the link of srv1 tells that the token differs" \
    grep -q "does not hold a token of this cluster" <(mesh 1 "a['links'][0].get('error', '')")

  # The same token again, then the three steps of the change, one instance at a time.
  step() {
    local n=$1 token=$2 additional=$3 desc=$4 peers
    peers='["127.0.0.1:7442"]'
    [ "$n" = 2 ] && peers='["127.0.0.1:7441"]'
    stop "$n"
    MK_TOKEN=$token MK_ADDITIONAL=$additional MK_PEERS=$peers mk "$n"
    start "$n"
    if wait_until 20 links_are 1 1 && wait_until 20 links_are 2 1; then ok "$desc"; else ko "$desc"; fi
  }

  step 2 "$TOKEN_A" "" "srv2 back on the token of srv1: both link"
  step 1 "$TOKEN_A" "$TOKEN_B" "step 1, srv1 accepts the new token too: both link"
  step 2 "$TOKEN_A" "$TOKEN_B" "step 1, srv2 accepts the new token too: both link"
  step 1 "$TOKEN_B" "$TOKEN_A" "step 2, srv1 shows the new token: both link"
  step 2 "$TOKEN_B" "$TOKEN_A" "step 2, srv2 shows the new token: both link"
  step 1 "$TOKEN_B" "" "step 3, srv1 drops the old token: both link"
  step 2 "$TOKEN_B" "" "step 3, srv2 drops the old token: both link"
}

scenario_http() {
  echo "Scenario http: the requests reach the only instance that has the router"
  reset
  server http 1
  dyn 1 http=0
  MK_PEERS="[]" mk 1
  MK_PEERS='["127.0.0.1:7441"]' mk 2
  MK_PEERS='["127.0.0.1:7441"]' mk 3
  for n in 1 2 3; do start "$n"; done

  eventually "srv3 imports the router of srv1" 20 via_is 3 backend1
  equal "srv1 serves its router" "backend1 srv1 srv1" "$(via 1 backend entry own)"
  equal "through srv2: entry point middleware of srv2, router middleware of srv1" "backend1 srv2 srv1" "$(via 2 backend entry own)"
  equal "through srv3: entry point middleware of srv3, router middleware of srv1" "backend1 srv3 srv1" "$(via 3 backend entry own)"
  equal "the server sees the address of the client" "127.0.0.1" "$(via 3 xff)"
  equal "the router of the mesh is named after the router and its provider" "imported app-file@mesh" \
    "$(mesh 2 "' '.join(r['status'] + ' ' + r.get('router', '') for r in a['imported'] if r['kind'] == 'http')")"

  head -c 1048576 /dev/urandom > "$WORK/body.bin"
  equal "a body of 1 MB through srv2" 1048576 "$(curl -s -m 10 -H 'Host: app.localhost' --data-binary "@$WORK/body.bin" \
    http://127.0.0.1:18082/ | python3 -c 'import json,sys; print(json.load(sys.stdin)["bodyBytes"])')"

  equal "an Upgrade through srv3" "HTTP/1.1 101 Switching Protocols|echo backend1 one|echo backend1 two" \
    "$(python3 "$HELPERS/upgrade_client.py" 18083 app.localhost 2> /dev/null | paste -sd'|' -)"

  equal "metric: requests of srv3 forwarded to srv1" 3 \
    "$(metric 3 traefik_mesh_requests_total kind=http,peer=srv1,result=forwarded)"

  # The router goes away: the other instances drop theirs.
  dyn 1
  eventually "the other instances drop the router when srv1 does" 10 status_is 2 404
}

scenario_priority() {
  echo "Scenario priority: two instances have the router, the priorities and the health of the servers choose"
  reset
  server http 1
  server http 2
  dyn 1 http=0
  dyn 2 http=0
  MK_PEERS="[]" mk 1
  MK_PEERS='["127.0.0.1:7441"]' mk 2
  MK_PEERS='["127.0.0.1:7441"]' mk 3
  for n in 1 2 3; do start "$n"; done
  wait_until 20 links_are 3 2

  eventually "default priority: srv1 serves its requests" 10 via_is 1 backend1
  equal "default priority: srv2 serves its requests" backend2 "$(via 2)"

  dyn 2 http=10
  eventually "priority 10 on srv2: the requests through srv1 go to srv2" 10 via_is 1 backend2
  equal "priority 10 on srv2: the requests through srv3 go to srv2" backend2 "$(via 3)"

  stop_server http 2
  eventually "the server of srv2 stops: the health check sends the requests through srv2 to srv1" 10 via_is 2 backend1
  # srv2 knows at once; srv1 knows when srv2 announces it, a moment later.
  eventually "the server of srv2 stops: the requests through srv1 stay on srv1" 10 via_is 1 backend1
  check "srv1 sees the router of srv2 down" grep -q True <(mesh 1 "any(r.get('down') for r in a['imported'])")

  server http 2
  eventually "the server of srv2 is back: the requests go to srv2 again" 10 via_is 1 backend2

  stop 2 KILL
  eventually "srv2 killed: the requests through srv1 go to srv1" 10 via_is 1 backend1
  eventually "srv2 killed: the requests through srv3 go to srv1" 10 via_is 3 backend1
}

scenario_tcp() {
  echo "Scenario tcp: the connections of the TCP routers reach the instance that has them"
  reset
  server tcp 1
  dyn 1 tcp=0
  MK_PEERS="[]" mk 1
  MK_PEERS='["127.0.0.1:7441"]' mk 2
  start 1
  start 2
  eventually "srv2 imports the TCP routers of srv1" 30 tcp_imported_are 2 4
  # A loaded machine takes a while to answer the first connection.
  eventually "the TCP router of srv1 answers through srv2" 30 tcp_answers 17002 plain1

  local answer local_address
  answer=$(tcp_client 17002)
  local_address=${answer%% |*}
  equal "TCP through srv2, with the PROXY protocol: the server sees the address of the client" \
    "plain1 src=$local_address dst=127.0.0.1:17002" "$(echo "$answer" | cut -d'|' -f2 | sed 's/^ //;s/ $//')"

  equal "TLS passthrough through srv2: TLS ends at the server" "pass1 src=- dst=- | verificado pela CA do servidor" \
    "$(tcp_client 17102 --tls pass.localhost --ca "$WORK/server.crt" | cut -d'|' -f2- | sed 's/^ //')"
  equal "TLS terminated by srv2" "term1 src=- dst=-" "$(tcp_client 17102 --tls term.localhost | cut -d'|' -f2 | sed 's/^ //;s/ $//')"
  check "the ipAllowList of the router runs on srv1 and refuses" \
    grep -q "falhou\|sem resposta" <(tcp_client 17102 --tls denied.localhost)

  check "a connection of 8 seconds through srv2" grep -q "conexao aberta ate o fim" <(tcp_client 17002 --hold 8)
}

scenario_udp() {
  echo "Scenario udp: the sessions of the UDP routers reach the instance that has them"
  reset
  server udp 1
  server own-udp 3
  dyn 1 udp=0
  dyn 3 own-udp
  MK_PEERS="[]" mk 1
  MK_PEERS='["127.0.0.1:7441"]' mk 2
  MK_PEERS='["127.0.0.1:7441"]' mk 3
  for n in 1 2 3; do start "$n"; done

  eventually "srv2 imports the UDP router of srv1" 20 udp_server_is 15302 dns1
  equal "srv3 keeps the UDP router it has on the entry point" skipped \
    "$(mesh 3 "' '.join(r['status'] for r in a['imported'] if r['kind'] == 'udp')")"
  equal "srv3 serves its own UDP router" own3 "$(udp_client 15303 100 | cut -d'|' -f2 | awk '{print $1}')"

  equal "packets of 100, 1200, 1400 and 8000 bytes through srv2" "100B inteiro, 1200B inteiro, 1400B inteiro, 8000B inteiro" \
    "$(udp_client 15302 100 1200 1400 8000 | cut -d'|' -f3 | sed 's/^ //')"
  equal "a session of 100 packets through srv2" "dns1: 100" "$(udp_client 15302 --count 100)"

  # The packets are counted when the session ends, after the timeout of the entry point.
  local path=datagram
  [ "$TRANSPORT" = tcp ] && path=stream
  eventually "metric: packets that srv2 sent to srv1 in $path" 10 \
    test -n "$(metric 2 traefik_mesh_udp_packets_total direction=sent,path=$path,peer=srv1)"
}

scenario_fallback() {
  echo "Scenario fallback: an instance that only listens over TCP is reached over TCP"
  reset
  server http 1
  dyn 1 http=0
  MK_PEERS='["127.0.0.1:7442"]' MK_TRANSPORT=auto mk 1
  MK_PEERS='["127.0.0.1:7441"]' MK_TRANSPORT=tcp mk 2
  start 1
  start 2

  eventually "srv1 links to srv2, which has no QUIC, after the wait for QUIC" 30 links_are 1 1
  equal "the link of srv1 to srv2 is over TCP" tcp "$(mesh 1 "a['links'][0].get('transport')")"
  eventually "the requests through srv2 reach srv1" 20 via_is 2 backend1
}

# link_address <instance> <nodeID>: prints the address of the link up from the instance to the other one.
link_address() {
  mesh "$1" "' '.join(l.get('address', '') for l in a['links'] if l.get('nodeID') == '$2' and l['up'])"
}

# forgot <instance> <nodeID>: the instance has no link to the other one, up or down.
forgot() {
  [ "$(mesh "$1" "sum(1 for l in a['links'] if l.get('nodeID') == '$2')")" = 0 ]
}

# downtime <instance> <backend> <seconds>: requests the router every 100 ms for that long, and prints how many failed.
downtime() {
  local n=$1 backend=$2 failed=0 deadline
  deadline=$(($(date +%s) + $3))
  while [ "$(date +%s)" -lt "$deadline" ]; do
    [ "$(via "$n")" = "$backend" ] || failed=$((failed + 1))
    sleep 0.1
  done
  echo "$failed"
}

scenario_satellite-tcp() {
  echo "Scenario satellite-tcp: a satellite that only gets out over TCP opens the link back itself"
  reset
  server http 3
  server udp 3
  dyn 3 http=0 udp=0
  MK_PEERS="[]" mk 1
  MK_PEERS='["127.0.0.1:7441"]' MK_SATELLITE=1 MK_TRANSPORT=tcp mk 3
  start 1
  start 3

  eventually "the router of the satellite is served by srv1" 20 via_is 1 backend3
  equal "srv1 uses the link the satellite opened over TCP" tcp-reverse \
    "$(mesh 1 "' '.join(l.get('transport', '') for l in a['links'] if l.get('nodeID') == 'srv3' and l['up'])")"
  eventually "the UDP router of the satellite is served by srv1, in the stream of the flow" 10 udp_server_is 15301 dns3
  equal "packets of 100 and 4000 bytes through srv1" "100B inteiro, 4000B inteiro" \
    "$(udp_client 15301 100 4000 | cut -d'|' -f3 | sed 's/^ //')"

  stop 3
  eventually "the satellite stops: srv1 forgets it" 20 forgot 1 srv3
  eventually "the router of the satellite leaves srv1" 10 status_is 1 404
}

# issuer_of <port> <server name>: prints the issuer and the serial of the certificate served at the port for the name.
issuer_of() {
  openssl s_client -connect "127.0.0.1:$1" -servername "$2" < /dev/null 2> /dev/null |
    openssl x509 -noout -issuer -serial 2> /dev/null | tr '\n' ' '
}

# pebble_issued <port> <server name>: the certificate served at the port for the name comes from Pebble.
pebble_issued() {
  issuer_of "$1" "$2" | grep -q Pebble
}

# start_pebble <port of the HTTP-01 validations>: starts Pebble, which validates the challenges at that port.
start_pebble() {
  openssl req -x509 -newkey rsa:2048 -nodes -keyout "$WORK/pebble.key" -out "$WORK/pebble.crt" -days 2 \
    -subj "/CN=localhost" -addext "subjectAltName=DNS:localhost,IP:127.0.0.1" 2> /dev/null
  cat > "$WORK/pebble.json" << EOF
{
  "pebble": {
    "listenAddress": "127.0.0.1:$PEBBLE_PORT",
    "managementListenAddress": "127.0.0.1:15000",
    "certificate": "$WORK/pebble.crt",
    "privateKey": "$WORK/pebble.key",
    "httpPort": $1,
    "tlsPort": 18998,
    "ocspResponderURL": "",
    "externalAccountBindingRequired": false
  }
}
EOF
  PEBBLE_VA_NOSLEEP=1 PEBBLE_WFE_NONCEREJECT=0 PEBBLE_AUTHZREUSE=0 "$PEBBLE" -config "$WORK/pebble.json" > "$WORK/pebble.log" 2>&1 &
  echo $! >> "$WORK/servers.pid"
  wait_until 10 curl -sk "https://127.0.0.1:$PEBBLE_PORT/dir"
}

scenario_challenge() {
  echo "Scenario challenge: without a Redis server, the instance the CA reaches asks the others for the HTTP-01 challenge"
  if [ ! -x "$PEBBLE" ]; then
    echo "  skipped: Pebble is not installed"
    return
  fi
  reset

  # The CA validates at the web entry point of srv2, and srv1 asks for the certificate.
  start_pebble 18082

  server http 1
  dyn 1
  dyn 2
  MK_PEERS="[]" MK_RESOLVER=1 MK_WEB_HOST="" mk 1
  MK_PEERS='["127.0.0.1:7441"]' MK_RESOLVER=1 MK_WEB_HOST="" mk 2
  start 1
  start 2
  wait_until 20 links_are 2 1

  # The router comes once both instances are up: a validation that finds srv2 down makes the CA client wait.
  cp "$WORK/n1/dynamic.yml" "$WORK/n1/dynamic.yml.next"
  cat >> "$WORK/n1/dynamic.yml.next" << EOF
  routers:
    chal:
      rule: "Host(\`chal.localhost\`)"
      entryPoints: [tls]
      service: chal
      tls:
        certResolver: le
      mesh:
        export: false
  services:
    chal:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:19001"
EOF
  mv "$WORK/n1/dynamic.yml.next" "$WORK/n1/dynamic.yml"

  eventually "srv1 gets its certificate, though the CA validated at srv2" 60 pebble_issued 17101 chal.localhost
  eventually "metric: srv2 answered the challenge with the answer of srv1, from the mesh" 10 \
    test -n "$(metric 2 traefik_acme_cluster_challenge_lookups_total result=mesh)"
  check "srv2 did not ask for the certificate itself" no_certificate_for 2 chal.localhost
}

# no_certificate_for <instance> <domain>: the ACME storage of the instance has no certificate for the domain.
no_certificate_for() {
  ! grep -q "$2" "$WORK/n$1/acme.json" 2> /dev/null
}

scenario_satellite-certs() {
  echo "Scenario satellite-certs: the central asks the CA for the certificate of a router of the satellite, and the satellite serves it"
  if [ ! -x "$PEBBLE" ]; then
    echo "  skipped: Pebble is not installed"
    return
  fi
  reset

  start_pebble 18081

  server http 3
  dyn 1
  dyn 3
  # The router of the satellite names the resolver of the central, on an entry point both have.
  cat >> "$WORK/n3/dynamic.yml" << EOF
  routers:
    nas:
      rule: "Host(\`nas.localhost\`)"
      entryPoints: [tls]
      service: nas
      tls:
        certResolver: le
  services:
    nas:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:19003"
EOF
  MK_PEERS="[]" MK_RESOLVER=1 MK_WEB_HOST="" mk 1
  MK_PEERS='["127.0.0.1:7441"]' MK_SATELLITE=1 MK_RESOLVER=1 mk 3
  start 1
  start 3

  eventually "the central gets the certificate of the router of the satellite from the CA" 60 pebble_issued 17101 nas.localhost
  eventually "the satellite serves that certificate on its own entry point" 60 pebble_issued 17103 nas.localhost
  equal "the same certificate on both" "$(issuer_of 17101 nas.localhost)" "$(issuer_of 17103 nas.localhost)"
  check "the satellite did not ask the CA itself" test ! -f "$WORK/n3/acme.json"
  equal "the API of the satellite lists the certificate" "nas.localhost" \
    "$(mesh 3 "' '.join(d for c in a.get('certificates', []) for d in c['domains'])")"
  check "no error about the resolver on the satellite" no_resolver_error 3

  stop 1
  sleep 2
  check "the central stops: the satellite still serves the certificate" pebble_issued 17103 nas.localhost
}

# no_resolver_error <instance>: the log of the instance has no router with an unknown certificate resolver.
no_resolver_error() {
  ! grep -q "nonexistent certificate resolver" "$WORK/n$1/out.log"
}

scenario_satellite() {
  echo "Scenario satellite: an instance behind a NAT serves its router through the link it opens"
  if [ "$TRANSPORT" = tcp ]; then
    echo "  skipped: a satellite is reached back over QUIC"
    return
  fi
  reset
  server http 3
  dyn 3 http=0
  MK_PEERS="[]" mk 1
  python3 "$HELPERS/udp_nat.py" 17499 127.0.0.1:7441 "$WORK/nat.mappings" > /dev/null 2>&1 &
  echo $! >> "$WORK/servers.pid"
  echo $! > "$WORK/server-nat-1.pid"
  MK_PEERS='["127.0.0.1:17499"]' MK_SATELLITE=1 MK_TRANSPORT=quic mk 3
  start 1
  start 3

  eventually "the router of the satellite is served by srv1" 20 via_is 1 backend3
  equal "entry point middleware on srv1, router middleware on the satellite" "backend3 srv1 srv3" "$(via 1 backend entry own)"
  equal "srv1 reaches the satellite back at the address of the NAT" "$(head -1 "$WORK/nat.mappings")" "$(link_address 1 srv3)"
  equal "srv1 sees srv3 as a satellite" satellite "$(mesh 1 "' '.join(l.get('role', '') for l in a['links'] if l.get('nodeID') == 'srv3')")"

  local before failed
  before=$(link_address 1 srv3)
  kill -USR1 "$(cat "$WORK/server-nat-1.pid")"
  failed=$(downtime 1 backend3 8)
  eventually "the NAT maps the satellite to another port: srv1 follows" 10 test "$(link_address 1 srv3)" != "$before"
  check "requests through srv1 while the NAT remaps: $failed of about 80 failed" test "$failed" -lt 40
  eventually "the router of the satellite is served again after the remapping" 10 via_is 1 backend3

  stop 3
  eventually "the satellite stops: srv1 forgets it" 20 forgot 1 srv3
  eventually "the router of the satellite leaves srv1" 10 status_is 1 404
}

scenario_config() {
  echo "Scenario config: the instances compare their entry points, and count their routers"
  reset
  server http 1
  dyn 1 http=0
  MK_PEERS="[]" mk 1
  MK_PEERS='["127.0.0.1:7441"]' MK_EXTRA_MIDDLEWARE=1 mk 2
  start 1
  start 2
  wait_until 20 links_are 2 1

  eventually "metric: srv1 sees one entry point of srv2 with another configuration" 10 \
    test "$(metric 1 traefik_mesh_config_mismatch peer=srv2)" = 1
  equal "the API of srv1 names the difference" "web: middlewares [entry@file] here, [entry@file, extra@file] there" \
    "$(mesh 1 "' '.join(l.get('configMismatch', [''])[0] for l in a['links'])")"
  check "srv2 warns about the difference" grep -q "another configuration" "$WORK/n2/out.log"

  equal "metric: srv1 announces one HTTP router" 1 "$(metric 1 traefik_mesh_exported_routes kind=http)"
  eventually "metric: srv2 counts the HTTP router of srv1" 10 test "$(metric 2 traefik_mesh_routes kind=http,peer=srv1)" = 1

  # The same entry points: the difference is gone.
  stop 2
  MK_PEERS='["127.0.0.1:7441"]' mk 2
  start 2
  eventually "srv2 restarted with the same entry points: no difference" 20 test "$(metric 1 traefik_mesh_config_mismatch peer=srv2)" = 0
}

cleanup() {
  stop_all 2> /dev/null
  keep_logs
  if [ "${KEEP:-0}" = 1 ] || [ "$FAILURES" -gt 0 ]; then
    echo "Logs kept in $WORK"
  else
    rm -rf "$WORK"
  fi
}
trap cleanup EXIT

if [ -n "${TRAEFIK:-}" ]; then
  cp "$TRAEFIK" "$WORK/traefik" || exit 2
else
  echo "Building the binary..."
  (cd "$ROOT" && go build -o "$WORK/traefik" ./cmd/traefik) || exit 2
fi

# The first start of a binary that was just built takes a while on macOS.
"$WORK/traefik" version > /dev/null 2>&1

openssl req -x509 -newkey rsa:2048 -nodes -keyout "$WORK/server.key" -out "$WORK/server.crt" -days 2 \
  -subj "/CN=pass.localhost" -addext "subjectAltName=DNS:pass.localhost" 2> /dev/null || exit 2

[ $# -eq 0 ] && set -- formation token http priority tcp udp fallback config satellite satellite-tcp satellite-certs challenge

echo "Links of the mesh: $TRANSPORT"

for name in "$@"; do
  if ! declare -F "scenario_$name" > /dev/null; then
    echo "Unknown scenario: $name" >&2
    exit 2
  fi
  "scenario_$name"
  SCENARIO=$name
done

# The last scenario is checked for crashes before the count.
stop_all
keep_logs
SCENARIO=""

echo
echo "$((CHECKS - FAILURES)) of $CHECKS checks passed"
[ "$FAILURES" -eq 0 ]
