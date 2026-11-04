#!/usr/bin/env bash
# Builds siot, starts a scratch instance seeded from demo.yaml, captures the
# documentation screenshots with shots.mjs straight into docs/, and stops
# everything again. Run from anywhere inside the repo.
#
#   scripts/capture.sh            # every screenshot
#   scripts/capture.sh 'modbus'   # only the shots whose name matches
#
# Ports can be changed with SIOT_SHOT_NATS_PORT, SIOT_SHOT_HTTP_PORT, and
# SIOT_SHOT_UPDATE_PORT when the defaults are busy.
set -euo pipefail

here=$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)
root=$(cd "$here/../../../.." && pwd)
nats_port=${SIOT_SHOT_NATS_PORT:-15322}
http_port=${SIOT_SHOT_HTTP_PORT:-15323}
update_port=${SIOT_SHOT_UPDATE_PORT:-15330}
work=$(mktemp -d -t siot-shots.XXXXXX)

cleanup() {
	[ -n "${siot_pid:-}" ] && kill "$siot_pid" 2>/dev/null || true
	[ -n "${http_pid:-}" ] && kill "$http_pid" 2>/dev/null || true
	rm -rf "$work"
}
trap cleanup EXIT

cd "$root"
[ -d ui/node_modules/@playwright/test ] || {
	echo "Playwright is missing: run 'source envsetup.sh && siot_setup' first" >&2
	exit 1
}

# the version shows in the root node, so build with a real one
echo "building siot ..."
go build -ldflags="-X main.version=$(git describe --tags HEAD)" -o "$work/siot" ./cmd/siot

# the update node lists the files an update server offers
mkdir -p "$work/updates" "$work/data/provisioning"
printf 'odroid-c4-hardkernel_%s.upd\n' 0.3.2 0.3.3 0.4.0 0.4.1 0.4.2 >"$work/updates/files.txt"
(cd "$work/updates" && python3 -m http.server "$update_port" --bind 127.0.0.1 >/dev/null 2>&1) &
http_pid=$!

# a provisioning file so the provisioning node has something to show
cp "$here/provisioning.yaml" "$work/data/provisioning/10-site.yaml"

echo "starting siot on http://localhost:$http_port ..."
SIOT_DATA="$work/data" SIOT_NATS_PORT="$nats_port" SIOT_HTTP_PORT="$http_port" \
	"$work/siot" serve >"$work/serve.log" 2>&1 &
siot_pid=$!
for _ in $(seq 1 50); do
	curl -sf -o /dev/null "http://localhost:$http_port/" && break
	sleep 0.2
done

echo "importing demo tree ..."
# the dashboard demo first, so demo.yaml's root description wins; it also
# gives the admin user a home dashboard, which the dashboard shot relies on
"$work/siot" import -natsServer "nats://localhost:$nats_port" <"$root/testdata/dashboard-demo.yaml" 2>&1 | grep -i 'error' || true
sed "s/__UPDATE_PORT__/$update_port/" "$here/demo.yaml" |
	"$work/siot" import -natsServer "nats://localhost:$nats_port" 2>&1 | grep -i 'error' || true
# metrics, the GPS simulator, and the signal generators need a few seconds
# to publish their first values
sleep 8

echo "capturing ..."
TZ=America/New_York SIOT_SHOT_URL="http://localhost:$http_port" node "$here/shots.mjs" --out "$root/docs" "${1:-}"

echo
echo "changed images:"
git -C "$root" status --short -- docs | grep -i '\.png$' || echo "(none)"
