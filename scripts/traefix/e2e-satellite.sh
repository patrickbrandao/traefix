#!/usr/bin/env bash
# Checks a satellite of the mesh behind a real NAT: network namespaces and iptables on Linux, with the real binary.
# Two centrals and a visitor share a network; the satellite sits behind a router that masquerades it, and no
# connection from outside reaches it.
#
#   scripts/traefix/e2e-satellite.sh                     # every scenario
#   scripts/traefix/e2e-satellite.sh cone remap          # only the named scenarios
#   SWARM=500 scripts/traefix/e2e-satellite.sh swarm     # how many satellites the swarm scenario runs (200)
#   KEEP=1 scripts/traefix/e2e-satellite.sh              # keeps the working directory, with the logs
#   LOG_LEVEL=DEBUG scripts/traefix/e2e-satellite.sh
#
# Scenarios: cone symmetric remap udp-blocked priority certs swarm
#
#   192.168.77.10 satellite ── 192.168.77.1 NAT 203.0.113.50 ── bridge ── 203.0.113.1 central1
#                                                                      ├─ 203.0.113.2 central2
#                                                                      └─ 203.0.113.100 visitor (and the CA)
#
# On Linux as root, it runs as it is and needs ip, iptables, conntrack, python3, curl and openssl. Elsewhere it builds
# the binaries for Linux and runs itself in a privileged container (Docker), where nothing touches the network of the
# host. The scenario certs needs Pebble, built from the module cache when it is there
# (go install github.com/letsencrypt/pebble/v2/cmd/pebble@latest).

set -u

ROOT=$(cd "$(dirname "$0")/../.." && pwd)
SCENARIOS="cone symmetric remap udp-blocked priority certs swarm"
# shellcheck disable=SC2086 # The names of the scenarios are words.
[ $# -eq 0 ] && set -- $SCENARIOS

# Outside: builds what the container needs, and runs this script in it.
if [ "${INSIDE:-0}" != 1 ] && { [ "$(uname -s)" != Linux ] || [ "$(id -u)" != 0 ]; }; then
  if ! command -v docker > /dev/null 2>&1; then
    echo "Needs Linux as root, or Docker to run in a privileged container" >&2
    exit 2
  fi

  WORK=${WORK:-$(mktemp -d "${TMPDIR:-/tmp}/traefix-e2e-satellite.XXXXXX")}
  ARCH=$(docker version --format '{{.Server.Arch}}')

  echo "Building the binaries for linux/$ARCH in $WORK"
  (cd "$ROOT" && GOOS=linux GOARCH=$ARCH CGO_ENABLED=0 go build -o "$WORK/traefik" ./cmd/traefik) || exit 2
  (cd "$ROOT" && GOOS=linux GOARCH=$ARCH CGO_ENABLED=0 go test -c -o "$WORK/mesh.test" ./pkg/mesh/) || exit 2

  pebble_dir=$(find "$(go env GOMODCACHE)/github.com/letsencrypt/pebble" -maxdepth 1 -name 'v2@v*' 2> /dev/null | sort | tail -1)
  if [ -n "$pebble_dir" ]; then
    (cd "$pebble_dir" && GOOS=linux GOARCH=$ARCH CGO_ENABLED=0 go build -o "$WORK/pebble" ./cmd/pebble) 2> /dev/null
  fi

  docker run --rm --privileged \
    -v "$ROOT/scripts/traefix:/scripts:ro" -v "$WORK:/work" \
    -e INSIDE=1 -e WORK=/work -e SWARM="${SWARM:-200}" -e LOG_LEVEL="${LOG_LEVEL:-INFO}" \
    alpine:3.22 sh -c 'apk add -q bash iproute2 iptables conntrack-tools python3 curl openssl procps > /dev/null && exec bash /scripts/e2e-satellite.sh "$@"' \
    e2e-satellite "$@"
  status=$?

  if [ "${KEEP:-0}" = 1 ] || [ "$status" != 0 ]; then
    echo "Working directory kept: $WORK"
  else
    rm -rf "$WORK"
  fi

  exit "$status"
fi

HELPERS="$(cd "$(dirname "$0")" && pwd)/e2e-mesh"
WORK=${WORK:-$(mktemp -d /tmp/traefix-e2e-satellite.XXXXXX)}
TRAEFIK=${TRAEFIK:-$WORK/traefik}
MESH_TEST=${MESH_TEST:-$WORK/mesh.test}
PEBBLE=${PEBBLE:-$WORK/pebble}
SWARM=${SWARM:-200}

TOKEN=cccccccccccccccccccccccccccccccccccccccc
NAMESPACES="wan c1 c2 v nat s"

FAILURES=0
CHECKS=0

for bin in ip iptables conntrack python3 curl openssl; do
  if ! command -v "$bin" > /dev/null 2>&1; then
    echo "Missing a tool the test needs (ip, iptables, conntrack, python3, curl, openssl): '$bin'" >&2
    exit 2
  fi
done

if [ ! -x "$TRAEFIK" ]; then
  echo "Building the binary"
  (cd "$(dirname "$0")/../.." && CGO_ENABLED=0 go build -o "$TRAEFIK" ./cmd/traefik) || exit 2
fi

ok() {
  CHECKS=$((CHECKS + 1))
  echo "  ok    $1"
}

ko() {
  CHECKS=$((CHECKS + 1))
  FAILURES=$((FAILURES + 1))
  echo "  FAIL  $1"
}

info() {
  echo "  info  $1"
}

check() {
  local desc=$1
  shift
  if "$@" > /dev/null 2>&1; then ok "$desc"; else ko "$desc"; fi
}

equal() {
  if [ "$2" = "$3" ]; then ok "$1"; else ko "$1 (expected '$2', got '$3')"; fi
}

wait_until() {
  local deadline=$(($(date +%s) + $1))
  shift
  until "$@" > /dev/null 2>&1; do
    [ "$(date +%s)" -ge "$deadline" ] && return 1
    sleep 0.5
  done
}

eventually() {
  local desc=$1 seconds=$2
  shift 2
  if wait_until "$seconds" "$@"; then ok "$desc"; else ko "$desc"; fi
}

end() {
  kill "-${2:-TERM}" "$1" 2> /dev/null
  wait "$1" 2> /dev/null
  return 0
}

# at <namespace> <command...>: runs the command in a namespace of the test.
at() {
  local ns=$1
  shift
  ip netns exec "tx-$ns" "$@"
}

# spawn <namespace> <command...>: at, for a command started in the background. $! is then the pid of the command
# itself, which a background function would not give: the subshell would be stopped, and the command left running.
spawn() {
  local ns=$1
  shift
  exec ip netns exec "tx-$ns" "$@"
}

net_down() {
  for ns in $NAMESPACES; do ip netns del "tx-$ns" 2> /dev/null; done
  rm -rf /etc/netns/tx-v
  return 0
}

# net_up: the namespaces, the bridge that stands for the Internet, and the network behind the NAT.
net_up() {
  net_down
  for ns in $NAMESPACES; do
    ip netns add "tx-$ns"
    at "$ns" ip link set lo up
  done

  at wan ip link add br0 type bridge
  at wan ip link set br0 up

  local pair
  for pair in c1:203.0.113.1 c2:203.0.113.2 v:203.0.113.100 nat:203.0.113.50; do
    local ns=${pair%%:*} address=${pair#*:}
    ip link add eth0 netns "tx-$ns" type veth peer name "$ns" netns tx-wan
    at wan ip link set "$ns" master br0 up
    at "$ns" ip addr add "$address/24" dev eth0
    at "$ns" ip link set eth0 up
  done

  ip link add eth0 netns tx-s type veth peer name lan0 netns tx-nat
  at nat ip addr add 192.168.77.1/24 dev lan0
  at nat ip link set lan0 up
  at s ip addr add 192.168.77.10/24 dev eth0
  at s ip link set eth0 up
  at s ip route add default via 192.168.77.1
  at nat sysctl -qw net.ipv4.ip_forward=1

  # The names of the routers resolve to central1, for the visitor and for the CA.
  mkdir -p /etc/netns/tx-v
  printf '127.0.0.1 localhost\n203.0.113.1 nas.example other.example\n' > /etc/netns/tx-v/hosts
}

# nat <cone|symmetric> [udp-blocked]: what the router of the satellite lets out, and how it maps it.
# Nothing from outside comes in, but what answers a connection from inside.
nat() {
  local random=""
  [ "$1" = symmetric ] && random="--random-fully"

  at nat iptables -F
  at nat iptables -t nat -F
  at nat iptables -P FORWARD DROP
  at nat iptables -A FORWARD -m conntrack --ctstate ESTABLISHED,RELATED -j ACCEPT
  [ "${2:-}" = udp-blocked ] && at nat iptables -A FORWARD -i lan0 -p udp -j DROP
  at nat iptables -A FORWARD -i lan0 -o eth0 -j ACCEPT
  # shellcheck disable=SC2086
  at nat iptables -t nat -A POSTROUTING -o eth0 -j MASQUERADE $random
  at nat conntrack -F > /dev/null 2>&1
  return 0
}

# Settings of the next mk calls.
MK_LINK_TIMEOUT=3s
MK_RESOLVER=0
MK_PRIORITY=""
MK_OWN_ROUTER=0

# mk <c1|c2|s>: writes the install and routing configuration of an instance.
mk() {
  local name=$1 dir="$WORK/$1"
  mkdir -p "$dir"

  {
    cat << EOF
entryPoints:
  web:
    address: ":80"
  websecure:
    address: ":443"
  dns:
    address: ":53/udp"
    udp:
      timeout: 3s
  traefik:
    address: "127.0.0.1:8080"
api:
  insecure: true
  debug: true
metrics:
  prometheus:
    entryPoint: traefik
log:
  level: ${LOG_LEVEL:-INFO}
providers:
  file:
    filename: $dir/dynamic.yml
cluster:
  token: $TOKEN
  mesh:
    address: ":7443"
    linkTimeout: $MK_LINK_TIMEOUT
EOF
    case $name in
      c1) printf '    advertise: ["203.0.113.1:7443"]\n  nodeID: central1\n' ;;
      c2) printf '    advertise: ["203.0.113.2:7443"]\n    peers: ["203.0.113.1:7443"]\n  nodeID: central2\n' ;;
      # The satellite knows one central: it learns the other one from it.
      s) printf '    satellite: true\n    peers: ["203.0.113.1:7443"]\n  nodeID: casa\n' ;;
    esac
    if [ "$MK_RESOLVER" = 1 ]; then
      cat << EOF
certificatesResolvers:
  le:
    acme:
      email: ops@example.com
      caServer: https://203.0.113.100:14000/dir
      caCertificates:
        - $WORK/pebble.crt
      storage: $dir/acme.json
      httpChallenge:
        entryPoint: web
EOF
    fi
  } > "$dir/traefik.yml"

  case $name in
    s) dyn_satellite > "$dir/dynamic.yml" ;;
    *) dyn_central "$name" > "$dir/dynamic.yml" ;;
  esac
}

dyn_satellite() {
  cat << EOF
http:
  routers:
    nas:
      rule: "Host(\`nas.example\`)"
      entryPoints: [web]
      service: nas
EOF
  [ -n "$MK_PRIORITY" ] && printf '      mesh:\n        priority: %s\n' "$MK_PRIORITY"
  cat << EOF
    big:
      rule: "Host(\`big.example\`)"
      entryPoints: [web]
      service: big
EOF
  if [ "$MK_RESOLVER" = 1 ]; then
    cat << EOF
    nas-secure:
      rule: "Host(\`nas.example\`)"
      entryPoints: [websecure]
      service: nas
      tls:
        certResolver: le
EOF
  fi
  cat << EOF
  services:
    nas:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:19003"
    big:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:19004"
tcp:
  routers:
    pass:
      rule: "HostSNI(\`pass.example\`)"
      entryPoints: [websecure]
      service: pass
      tls:
        passthrough: true
  services:
    pass:
      loadBalancer:
        servers:
          - address: "127.0.0.1:19513"
udp:
  routers:
    dns:
      entryPoints: [dns]
      service: dns
  services:
    dns:
      loadBalancer:
        servers:
          - address: "127.0.0.1:19703"
EOF
}

# dyn_central <c1|c2>: a central has routers of its own only when a scenario asks for them.
dyn_central() {
  echo "http:"
  if [ "$MK_OWN_ROUTER" = 1 ]; then
    cat << EOF
  routers:
    nas:
      rule: "Host(\`nas.example\`)"
      entryPoints: [web]
      service: nas
  services:
    nas:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:19001"
EOF
  elif [ "$MK_RESOLVER" = 1 ] && [ "$1" = c1 ]; then
    cat << EOF
  routers:
    other:
      rule: "Host(\`other.example\`)"
      entryPoints: [websecure]
      service: other
      tls:
        certResolver: le
  services:
    other:
      loadBalancer:
        servers:
          - url: "http://127.0.0.1:19001"
EOF
  else
    # An empty map is refused by the file provider: a middleware nobody uses stands for nothing.
    printf '  middlewares:\n    none:\n      headers:\n        customRequestHeaders:\n          X-Central: "%s"\n' "$1"
  fi
}

start() {
  local name=$1
  spawn "$name" "$TRAEFIK" --configFile="$WORK/$name/traefik.yml" >> "$WORK/$name/out.log" 2>&1 &
  echo $! > "$WORK/$name/pid"
  wait_until 20 at "$name" curl -sf http://127.0.0.1:8080/api/mesh
}

stop() {
  [ -f "$WORK/$1/pid" ] && end "$(cat "$WORK/$1/pid")"
  rm -f "$WORK/$1/pid"
}

# servers: the test servers of the satellite, and the one of central1.
servers() {
  if [ ! -f "$WORK/server.crt" ]; then
    openssl req -x509 -newkey rsa:2048 -nodes -keyout "$WORK/server.key" -out "$WORK/server.crt" -days 2 \
      -subj "/CN=pass.example" 2> /dev/null
    mkdir -p "$WORK/files"
    head -c $((256 << 20)) /dev/urandom > "$WORK/files/big.bin"
  fi
  spawn s python3 "$HELPERS/http_backend.py" nas 19003 > /dev/null 2>&1 &
  echo $! >> "$WORK/servers.pid"
  spawn s python3 -m http.server 19004 --bind 127.0.0.1 --directory "$WORK/files" > /dev/null 2>&1 &
  echo $! >> "$WORK/servers.pid"
  spawn s python3 "$HELPERS/tcp_backend.py" pass 19513 --tls "$WORK/server.crt" "$WORK/server.key" > /dev/null 2>&1 &
  echo $! >> "$WORK/servers.pid"
  spawn s python3 "$HELPERS/udp_backend.py" dns 19703 > /dev/null 2>&1 &
  echo $! >> "$WORK/servers.pid"
  spawn c1 python3 "$HELPERS/http_backend.py" central1 19001 > /dev/null 2>&1 &
  echo $! >> "$WORK/servers.pid"
  sleep 0.5
}

stop_all() {
  local name
  for name in c1 c2 s; do stop "$name"; done
  if [ -f "$WORK/servers.pid" ]; then
    while read -r pid; do end "$pid"; done < "$WORK/servers.pid"
  fi
  rm -f "$WORK/servers.pid"
}

keep_logs() {
  [ -n "${SCENARIO:-}" ] || return 0
  for log in "$WORK"/c1/out.log "$WORK"/c2/out.log "$WORK"/s/out.log; do
    if [ -f "$log" ] && grep -q "^panic:" "$log"; then
      ko "no instance crashed in the scenario $SCENARIO (a panic is in $SCENARIO/$(basename "$(dirname "$log")")/out.log)"
    fi
  done
  mkdir -p "$WORK/$SCENARIO"
  for dir in c1 c2 s ca; do
    [ -d "$WORK/$dir" ] && mv "$WORK/$dir" "$WORK/$SCENARIO/"
  done
  return 0
}

reset() {
  stop_all
  keep_logs
  MK_LINK_TIMEOUT=3s
  MK_RESOLVER=0
  MK_PRIORITY=""
  MK_OWN_ROUTER=0
}

# api <instance> <python expression on the answer of /api/mesh, as a>
api() {
  at "$1" curl -s -m 5 http://127.0.0.1:8080/api/mesh | python3 -c "
import json, sys
a = json.load(sys.stdin)
print($2)" 2> /dev/null
}

links_up() {
  api "$1" "sum(1 for l in a['links'] if l['up'])"
}

links_are() {
  [ "$(links_up "$1")" = "$2" ]
}

# satellite <central> <field>: a field of the link of the central back to the satellite.
satellite() {
  api "$1" "next((str(l.get('$2', '')) for l in a['links'] if l.get('nodeID') == 'casa'), '')"
}

satellite_up_over() {
  [ "$(satellite "$1" up)" = True ] && [ "$(satellite "$1" transport)" = "$2" ]
}

# metric <instance> <name> <label=value,...>
metric() {
  at "$1" curl -s -m 5 http://127.0.0.1:8080/metrics | python3 -c "
import re, sys
name, wanted = sys.argv[1], dict(kv.split('=', 1) for kv in sys.argv[2].split(',') if kv)
for line in sys.stdin:
    m = re.match(r'^(\w+)(?:\{(.*)\})? (\S+)$', line.strip())
    if not m or m.group(1) != name:
        continue
    labels = dict(re.findall(r'(\w+)=\"([^\"]*)\"', m.group(2) or ''))
    if labels == wanted:
        print(m.group(3)[:-2] if m.group(3).endswith('.0') else m.group(3))
        break" "$2" "$3"
}

metric_is() {
  [ "$(metric "$1" "$2" "$3")" = "$4" ]
}

# visit <central address> [field]: the visitor asks nas.example of a central; prints what the server saw.
visit() {
  at v curl -s -m 5 -H 'Host: nas.example' "http://$1/" | python3 -c "
import json, sys
try:
    d = json.load(sys.stdin)
    print({'backend': d['backend'], 'xff': d['headers'].get('x-forwarded-for')}[sys.argv[1]])
except Exception:
    print('none')" "${2:-backend}"
}

visit_is() {
  [ "$(visit "$1")" = "$2" ]
}

tcp_visit() {
  at v python3 "$HELPERS/tcp_client.py" "$1:443" --tls pass.example | cut -d'|' -f2 | awk '{print $1}'
}

udp_visit() {
  at v python3 "$HELPERS/udp_client.py" "$1:53" 100 1200 | cut -d'|' -f2,3
}

# reaches_all <central>: the visitor reaches the HTTP, the TCP and the UDP router of the satellite through the central.
reaches_all() {
  local central=$1 desc=$2
  eventually "$desc: HTTP" 20 visit_is "$central" nas
  equal "$desc: TCP with passthrough, TLS ends at the satellite" pass "$(tcp_visit "$central")"
  equal "$desc: UDP" " dns from=127.0.0.1 | 100B inteiro, 1200B inteiro" "$(udp_visit "$central" | sed 's/from=127.0.0.1:[0-9]*/from=127.0.0.1/')"
}

# throughput <description> <central>: downloads 256 MiB of the satellite through the central, and tells the speed.
throughput() {
  local result
  result=$(at v curl -s -o /dev/null -m 120 -w '%{size_download} %{speed_download}' -H 'Host: big.example' "http://$2/big.bin")
  if [ "${result%% *}" = $((256 << 20)) ]; then
    info "$1: $(echo "$result" | awk '{printf "%.0f MB/s", $2 / 1e6}')"
  else
    ko "$1: the download of 256 MiB came whole (got '$result')"
  fi
}

# unreachable_from_outside: a central cannot open a connection to the satellite, at the address of its NAT.
unreachable_from_outside() {
  ! at c1 curl -sk -m 2 -o /dev/null https://203.0.113.50:7443/ &&
    ! at c1 curl -s -m 2 -o /dev/null -H 'Host: nas.example' http://203.0.113.50/
}

scenario_cone() {
  echo "Scenario cone: the satellite behind MASQUERADE serves its HTTP, TCP and UDP routers through both centrals"
  reset
  nat cone
  servers
  mk c1
  mk c2
  mk s
  start c1
  start c2
  start s

  check "no connection from outside reaches the satellite" unreachable_from_outside
  eventually "the satellite links to both centrals, though it knows one" 40 links_are s 2
  eventually "central1 reaches the satellite back over QUIC" 20 satellite_up_over c1 quic
  eventually "central2 reaches the satellite back over QUIC" 20 satellite_up_over c2 quic
  equal "central1 sees the satellite at the address of its NAT, from the port of the mesh" \
    203.0.113.50:7443 "$(satellite c1 observed)"
  equal "the role of the satellite in the API of central1" satellite "$(satellite c1 role)"
  equal "metric: central1 reaches one satellite back" 1 "$(metric c1 traefik_mesh_satellites state=up)"
  equal "metric: the satellite counts the centrals among its peers" 2 "$(metric s traefik_mesh_peers state=up)"
  equal "metric: central1 counts central2 among its peers, and not the satellite" 1 "$(metric c1 traefik_mesh_peers state=up)"

  reaches_all 203.0.113.1 "through central1"
  reaches_all 203.0.113.2 "through central2"
  equal "the server of the satellite sees the address of the visitor" 203.0.113.100 "$(visit 203.0.113.1 xff)"

  info "baseline, the satellite to itself: $(at s curl -s -o /dev/null -w '%{speed_download}' -H 'Host: big.example' http://127.0.0.1/big.bin | awk '{printf "%.0f MB/s", $1 / 1e6}')"
  throughput "download through central1, the link back over QUIC" 203.0.113.1

  stop c1
  eventually "central1 stops: the satellite is reached through central2" 20 visit_is 203.0.113.2 nas
  eventually "and the satellite keeps one link, once the other one times out" 10 links_are s 1
}

scenario_symmetric() {
  echo "Scenario symmetric: MASQUERADE --random-fully, a port of its own for each central"
  reset
  nat symmetric
  servers
  mk c1
  mk c2
  mk s
  start c1
  start c2
  start s

  eventually "the satellite links to both centrals" 40 links_are s 2
  eventually "central1 reaches the satellite back over QUIC" 20 satellite_up_over c1 quic
  eventually "central2 reaches the satellite back over QUIC" 20 satellite_up_over c2 quic

  local one two
  one=$(satellite c1 observed)
  two=$(satellite c2 observed)
  info "central1 sees the satellite at $one, central2 at $two"
  if [ "${one%:*}" = 203.0.113.50 ] && [ "${two%:*}" = 203.0.113.50 ] && [ "$one" != "$two" ]; then
    ok "the NAT maps the satellite to another port for each central"
  else
    ko "the NAT maps the satellite to another port for each central (got '$one' and '$two')"
  fi

  reaches_all 203.0.113.1 "through central1"
  reaches_all 203.0.113.2 "through central2"
}

# observed_changed <central> <previous>
observed_changed() {
  local now
  now=$(satellite "$1" observed)
  [ -n "$now" ] && [ "$now" != "$2" ]
}

scenario_remap() {
  echo "Scenario remap: the NAT forgets its mappings, and the centrals follow the satellite to the new ones"
  reset
  nat symmetric
  servers
  mk c1
  mk s
  start c1
  start s

  eventually "central1 reaches the satellite back over QUIC" 30 satellite_up_over c1 quic
  eventually "the visitor reaches the satellite" 10 visit_is 203.0.113.1 nas

  local before
  before=$(satellite c1 observed)

  # Requests go on while the mappings go away; each one has a second at most.
  local log="$WORK/remap-requests.log"
  (
    for _ in $(seq 1 60); do
      at v curl -s -o /dev/null -m 1 -w '%{http_code}\n' -H 'Host: nas.example' http://203.0.113.1/
      sleep 0.2
    done
  ) > "$log" &
  local requests=$!

  sleep 2
  at nat conntrack -D -p udp > /dev/null 2>&1
  local flushed
  flushed=$(date +%s)

  eventually "central1 sees the satellite at a new address" 10 observed_changed c1 "$before"
  eventually "the visitor reaches the satellite again" 10 visit_is 203.0.113.1 nas
  info "back $(($(date +%s) - flushed)) s after the NAT forgot the mappings, at $(satellite c1 observed) (was $before)"

  wait "$requests"
  info "$(grep -c '^200$' "$log") of $(wc -l < "$log" | tr -d ' ') requests answered while the mappings changed"
}

scenario_udp-blocked() {
  echo "Scenario udp-blocked: no UDP leaves the satellite, and its traffic goes through the link back over TCP"
  reset
  nat cone udp-blocked
  servers
  mk c1
  mk c2
  mk s
  start c1
  start c2
  start s

  eventually "the satellite links to both centrals over TCP" 60 links_are s 2
  eventually "central1 reaches the satellite over the connection it opened" 30 satellite_up_over c1 tcp-reverse
  eventually "central2 reaches the satellite over the connection it opened" 30 satellite_up_over c2 tcp-reverse
  equal "metric: the link of central1 is up over tcp-reverse" 1 \
    "$(metric c1 traefik_mesh_link_up peer=casa,transport=tcp-reverse)"
  equal "the API of central1 tells no address for a satellite behind TCP" "" "$(satellite c1 observed)"

  reaches_all 203.0.113.1 "through central1"
  reaches_all 203.0.113.2 "through central2"
  throughput "download through central1, the link back over TCP" 203.0.113.1
}

scenario_priority() {
  echo "Scenario priority: a central and the satellite have the same router, and the priorities choose"
  reset
  nat cone
  servers
  MK_OWN_ROUTER=1 mk c1
  mk c2
  MK_PRIORITY=10 mk s
  start c1
  start c2
  start s

  eventually "central1 sends its own router to the satellite, which has the highest priority" 30 visit_is 203.0.113.1 nas
  eventually "central2 chooses the satellite too" 30 visit_is 203.0.113.2 nas

  stop s
  eventually "the satellite goes away: central1 serves the router itself" 20 visit_is 203.0.113.1 central1
  eventually "and central2 sends it to central1" 20 visit_is 203.0.113.2 central1
}

# issuer <namespace> <address> <server name>: the issuer and the serial of the certificate served there.
issuer() {
  at "$1" openssl s_client -connect "$2:443" -servername "$3" < /dev/null 2> /dev/null |
    openssl x509 -noout -issuer -serial 2> /dev/null | tr '\n' ' '
}

issued_by_pebble() {
  issuer "$1" "$2" "$3" | grep -q Pebble
}

scenario_certs() {
  echo "Scenario certs: central1 asks the CA for the certificate of the router of the satellite, and the satellite serves it"
  if [ ! -x "$PEBBLE" ]; then
    echo "  skipped: Pebble is not built"
    return
  fi
  reset
  nat cone
  servers

  mkdir -p "$WORK/ca"
  openssl req -x509 -newkey rsa:2048 -nodes -keyout "$WORK/pebble.key" -out "$WORK/pebble.crt" -days 2 \
    -subj "/CN=pebble" -addext "subjectAltName=IP:203.0.113.100" 2> /dev/null
  cat > "$WORK/ca/pebble.json" << EOF
{
  "pebble": {
    "listenAddress": "203.0.113.100:14000",
    "managementListenAddress": "127.0.0.1:15000",
    "certificate": "$WORK/pebble.crt",
    "privateKey": "$WORK/pebble.key",
    "httpPort": 80,
    "tlsPort": 443,
    "ocspResponderURL": "",
    "externalAccountBindingRequired": false
  }
}
EOF
  spawn v env PEBBLE_VA_NOSLEEP=1 PEBBLE_WFE_NONCEREJECT=0 "$PEBBLE" -config "$WORK/ca/pebble.json" > "$WORK/ca/out.log" 2>&1 &
  echo $! >> "$WORK/servers.pid"
  wait_until 10 at v curl -sk https://203.0.113.100:14000/dir

  MK_RESOLVER=1 mk c1
  MK_RESOLVER=1 mk s
  start c1
  start s

  eventually "central1 gets the certificate of the router of the satellite from the CA" 60 issued_by_pebble v 203.0.113.1 nas.example
  eventually "the satellite serves that certificate on its own entry point" 60 issued_by_pebble s 127.0.0.1 nas.example
  equal "the same certificate on both" "$(issuer v 203.0.113.1 nas.example)" "$(issuer s 127.0.0.1 nas.example)"
  eventually "central1 gets the certificate of its own router too" 60 issued_by_pebble v 203.0.113.1 other.example
  equal "the satellite only got the certificate of the domain it announces" nas.example \
    "$(api s "' '.join(d for c in a.get('certificates', []) for d in c['domains'])")"
  equal "metric: central1 delivered one certificate to the satellite" 1 \
    "$(metric c1 traefik_mesh_certificates_delivered peer=casa)"
  check "the satellite did not ask the CA itself" test ! -f "$WORK/s/acme.json"
}

# cpu_ticks <pid>: the CPU time of a process, in ticks of the clock.
cpu_ticks() {
  awk '{print $14 + $15}' "/proc/$1/stat"
}

rss_mib() {
  awk '/VmRSS/ {printf "%.0f", $2 / 1024}' "/proc/$1/status"
}

# idle_cost <description> <pid>: what the process uses over 20 seconds.
idle_cost() {
  local before after hz
  hz=$(getconf CLK_TCK 2> /dev/null || echo 100)
  before=$(cpu_ticks "$2")
  sleep 20
  after=$(cpu_ticks "$2")
  info "$1: $(awk -v t=$((after - before)) -v hz="$hz" 'BEGIN {printf "%.1f%% of a CPU", 100 * t / hz / 20}'), $(rss_mib "$2") MiB of memory"
}

# goroutines <instance>: how many goroutines the instance runs.
goroutines() {
  at "$1" curl -s -m 5 'http://127.0.0.1:8080/debug/pprof/goroutine?debug=1' | awk 'NR == 1 {print $NF}'
}

satellites_up_are() {
  [ "$(metric c1 traefik_mesh_satellites state=up)" = "$1" ]
}

# no_satellite: central1 forgot every satellite, and no longer tries to reach one back.
no_satellite() {
  satellites_up_are 0 && [ "$(metric c1 traefik_mesh_satellites state=down)" = 0 ]
}

scenario_swarm() {
  echo "Scenario swarm: $SWARM satellites behind the NAT link to one central"
  local SCENARIO_DIR=swarm
  if [ ! -x "$MESH_TEST" ]; then
    echo "  skipped: the test binary of the mesh is not built"
    return
  fi
  reset
  nat cone
  servers
  # The swarm runs with the timers of the real thing.
  MK_LINK_TIMEOUT=10s mk c1
  start c1

  local central
  central=$(cat "$WORK/c1/pid")
  sleep 5
  idle_cost "central1 alone" "$central"
  local alone
  alone=$(goroutines c1)

  mkdir -p "$WORK/swarm"
  spawn s env TRAEFIX_MESH_SWARM=203.0.113.1:7443 TRAEFIX_MESH_SWARM_TOKEN=$TOKEN TRAEFIX_MESH_SWARM_HOST=192.168.77.10 \
    TRAEFIX_MESH_SWARM_COUNT="$SWARM" TRAEFIX_MESH_SWARM_HOLD=30m \
    "$MESH_TEST" -test.run '^TestSwarm$' -test.v > "$WORK/swarm/out.log" 2>&1 &
  local swarm=$!
  echo "$swarm" >> "$WORK/servers.pid"

  local started
  started=$(date +%s)
  eventually "central1 reaches all the $SWARM satellites back" 180 satellites_up_are "$SWARM"
  info "all up $(($(date +%s) - started)) s after the swarm started"

  idle_cost "central1 with $SWARM satellites" "$central"
  info "central1 runs $(goroutines c1) goroutines, $alone alone"
  idle_cost "the $SWARM satellites, in one process" "$swarm"

  local answered=0 n took
  started=$EPOCHREALTIME
  for n in $(seq 0 $((SWARM / 10)) $((SWARM - 1))); do
    [ "$(at v curl -s -m 5 -H "Host: sat$n.example" http://203.0.113.1/)" = "sat$n" ] && answered=$((answered + 1))
  done
  took=$(awk -v a="$started" -v b="$EPOCHREALTIME" 'BEGIN {printf "%.0f", (b - a) * 1000}')
  equal "the visitor reaches the satellites through central1" 10 "$answered"
  info "10 requests to as many satellites in $took ms"
  info "the API of central1 answers in $(at c1 curl -s -o /dev/null -w '%{time_total}' http://127.0.0.1:8080/api/mesh) s"

  end "$swarm"
  eventually "the satellites go away: central1 forgets them" 60 no_satellite
  info "central1 after they left: $(rss_mib "$central") MiB of memory"
  # A satellite that is forgotten leaves nothing running behind it.
  local left
  left=$(goroutines c1)
  if [ "$left" -le $((alone + 20)) ]; then
    ok "central1 runs as many goroutines as before the swarm ($left, $alone before)"
  else
    ko "central1 runs as many goroutines as before the swarm ($left, $alone before; see $SCENARIO_DIR/c1/goroutines.txt)"
    at c1 curl -s -m 5 'http://127.0.0.1:8080/debug/pprof/goroutine?debug=1' > "$WORK/c1/goroutines.txt"
  fi
}

trap 'stop_all; net_down' EXIT

net_up
for scenario in "$@"; do
  if ! declare -F "scenario_$scenario" > /dev/null; then
    echo "Unknown scenario: $scenario (scenarios: $SCENARIOS)" >&2
    exit 2
  fi
  "scenario_$scenario"
  # The next reset moves the instances of this scenario aside, under its name.
  SCENARIO=$scenario
done
stop_all
keep_logs

echo
echo "$((CHECKS - FAILURES)) of $CHECKS checks passed"
[ "$FAILURES" = 0 ]
