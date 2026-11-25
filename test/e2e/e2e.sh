#!/usr/bin/env bash
# End to end, on a real machine: the router, ports run, aliases, the hosts
# file, the background service and compose names — the paths that touch
# the system and so can't run on a developer's own machine casually. A
# CI runner is the machine here: disposable, passwordless sudo (an admin
# on Windows), Docker on Linux. Run it anywhere else only knowingly: it
# installs and removes the router's service and edits the hosts file.
set -euo pipefail

os=$(uname -s)
case "$os" in MINGW* | MSYS* | CYGWIN*) os=Windows ;; esac
py=python3
command -v python3 >/dev/null 2>&1 || py=python

group() { echo "::group::$*"; }
done_() { echo "::endgroup::"; }
fail() {
	echo "::error::$*"
	[ -f "$work/proxy.log" ] && { echo "--- router log"; cat "$work/proxy.log"; }
	exit 1
}

work=$(mktemp -d)
mkdir -p "$work/shop"
echo '{"name":"shop"}' >"$work/shop/package.json"
hosts=/etc/hosts
[ "$os" = Windows ] && hosts="$SYSTEMROOT/System32/drivers/etc/hosts"
cp "$hosts" "$work/hosts.before"

# an HTTP server on the port in $PORT, the way ports run hands one over
serve='import os, http.server as h; h.ThreadingHTTPServer(("127.0.0.1", int(os.environ["PORT"])), h.SimpleHTTPRequestHandler).serve_forever()'

routerPort() { ports proxy status --json | jq -r .port; }
status() { # status <host> → the HTTP code the router answers for it
	curl -s -o /dev/null -w '%{http_code}' -H "Host: $1" "http://127.0.0.1:$(routerPort)/" || true
}
resolves() { # resolves <host> → whether the system resolver (not curl's) finds it on loopback
	"$py" - "$1" <<'EOF'
import socket, sys
try:
    addrs = {a[4][0] for a in socket.getaddrinfo(sys.argv[1], 80)}
except OSError:
    sys.exit(1)
sys.exit(0 if addrs & {"127.0.0.1", "::1"} else 1)
EOF
}
waitFor() { # waitFor <seconds> <command…>
	local n=$(($1 * 5))
	shift
	for _ in $(seq "$n"); do "$@" >/dev/null 2>&1 && return 0; sleep 0.2; done
	return 1
}

group "router in the foreground"
ports proxy >"$work/proxy.log" 2>&1 &
waitFor 10 ports proxy status || fail "the router didn't come up"
echo "router on :$(routerPort)"
[ "$(status evil.example)" = 421 ] || fail "a foreign Host wasn't refused with 421"
done_

group "ports run names a server for as long as it runs"
(cd "$work/shop" && ports run site -- "$py" -c "$serve" >"$work/run.log" 2>&1 &)
ports wait site --http --timeout 20s || { cat "$work/run.log"; fail "site never answered"; }
[ "$(status site.localhost)" = 200 ] || fail "site.localhost didn't reach the server ports run started"
ports url site --local | grep -q '^http://localhost:[0-9]*$' || fail "ports url --local"
ports run site -- "$py" -c "$serve" 2>"$work/dup.err" && fail "a second ports run took a name that's in use"
grep -q "taken by another ports run" "$work/dup.err" || fail "the duplicate wasn't explained: $(cat "$work/dup.err")"
done_

group "hosts sync makes names resolve for every client"
ports alias e2e-alias 9 >/dev/null
resolves e2e-alias.localhost && echo "(this resolver already finds *.localhost on its own)"
ports hosts sync || fail "hosts sync"
resolves site.localhost || fail "site.localhost doesn't resolve after hosts sync"
resolves e2e-alias.localhost || fail "an alias doesn't resolve after hosts sync"
ports doctor --json | jq -e '.[] | select(.check == "names" and .status == "ok")' >/dev/null ||
	fail "doctor doesn't see the names resolve: $(ports doctor --json)"
ports hosts clean || fail "hosts clean"
cmp -s "$hosts" "$work/hosts.before" || { diff "$work/hosts.before" "$hosts" || true; fail "clean didn't leave the hosts file as it was"; }
ports alias e2e-alias --remove >/dev/null
done_

group "stop by name, guarded"
ports stop site --expect nope --json >/dev/null && fail "a stop the guard should refuse went through"
ports stop site --expect site --json >/dev/null || fail "stop site --expect site"
ports wait site --gone --timeout 10s >/dev/null || fail "site is still running"
ports url site >/dev/null 2>&1 && fail "site still has a name after it stopped"
ports proxy stop || fail "stopping the foreground router"
ports proxy status >/dev/null && fail "the router still runs after stop"
done_

group "the router in the background"
want=process
case "$os" in Linux) want=systemd ;; Darwin) want=launchd ;; esac
ports proxy start || fail "proxy start"
ports proxy start >/dev/null || fail "a second proxy start isn't a no-op"
managed=$(ports proxy status --json | jq -r .managed_by)
echo "managed by $managed"
[ "$managed" = "$want" ] || echo "::warning::wanted $want, got $managed (start fell back)"
"$py" -c "$(echo "$serve" | sed 's/int(os.environ\["PORT"\])/18765/')" >/dev/null 2>&1 &
web=$!
ports alias e2e-web 18765 >/dev/null
waitFor 10 test "$(status e2e-web.localhost)" = 200 || fail "the background router didn't route an alias"
kill "$web" 2>/dev/null || true
ports alias e2e-web --remove >/dev/null
ports proxy stop || fail "proxy stop"
ports proxy status >/dev/null && fail "the router still runs after proxy stop"
case "$os" in
Linux) [ ! -e "$HOME/.config/systemd/user/ports-proxy.service" ] || fail "the systemd unit is still there" ;;
Darwin) [ ! -e "$HOME/Library/LaunchAgents/tools.ports.proxy.plist" ] || fail "the launchd agent is still there" ;;
esac
done_

if [ "$os" = Linux ] && docker info >/dev/null 2>&1; then
	group "compose services are named after themselves"
	cat >"$work/shop/compose.yaml" <<'EOF'
services:
  web:
    image: nginx:alpine
    ports: ["18081:80"]
  cache:
    image: redis:alpine
    ports: ["16379:6379"]
EOF
	(cd "$work/shop" && docker compose up -d --quiet-pull)
	trap '(cd "$work/shop" && docker compose down -t 1 >/dev/null 2>&1) || true' EXIT
	ports proxy >"$work/proxy.log" 2>&1 &
	waitFor 10 ports proxy status || fail "the router didn't come up"
	waitFor 20 ports wait web.shop --timeout 1s || fail "web.shop never appeared: $(ports --json)"
	names=$(ports --json | jq -r '.[] | select(.compose_project == "shop") | "\(.port) \(.name) \(.name_url // "-")"')
	echo "$names"
	echo "$names" | grep -q '^18081 web.shop http://web.shop.localhost' || fail "web isn't web.shop with a URL"
	echo "$names" | grep -q '^16379 cache.shop -$' || fail "cache isn't cache.shop without a URL"
	waitFor 10 test "$(status web.shop.localhost)" = 200 || fail "web.shop.localhost didn't reach nginx"
	[ "$(status cache.shop.localhost)" != 200 ] || fail "redis was routed as if it spoke HTTP"
	ports proxy stop >/dev/null
	done_
fi

echo "end to end: all good on $os"
