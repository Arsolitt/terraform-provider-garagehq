#!/usr/bin/env bash
# Start a single-node Garage for the acceptance tests, and give the node a
# role in the cluster layout.
#
# With GARAGE_BIN set, run that binary. Else run the Docker image
# GARAGE_IMAGE. The admin API is then on 127.0.0.1:3903, with the token
# GARAGE_TOKEN. Stop Garage with `scripts/acc-garage.sh stop`.
set -euo pipefail

GARAGE_IMAGE="${GARAGE_IMAGE:-dxflrs/garage:v2.3.0}"
GARAGE_TOKEN="${GARAGE_TOKEN:-acceptance-test-token}"
GARAGE_DIR="${GARAGE_DIR:-${TMPDIR:-/tmp}/garage-acc}"
CONTAINER=garage-acc

if [ "${1:-}" = stop ]; then
	if [ -n "${GARAGE_BIN:-}" ]; then
		[ -f "$GARAGE_DIR/pid" ] && kill "$(cat "$GARAGE_DIR/pid")" || true
	else
		docker rm -f "$CONTAINER" >/dev/null 2>&1 || true
	fi
	rm -rf "$GARAGE_DIR"
	exit 0
fi

if [ -n "${GARAGE_BIN:-}" ]; then
	if [ -f "$GARAGE_DIR/pid" ]; then
		pid="$(cat "$GARAGE_DIR/pid")"
		if kill -0 "$pid" 2>/dev/null; then
			echo "error: Garage is already running with pid $pid (pid file $GARAGE_DIR/pid); run 'scripts/acc-garage.sh stop' first" >&2
			exit 1
		fi
	fi
else
	if docker inspect "$CONTAINER" >/dev/null 2>&1; then
		echo "error: container $CONTAINER already exists; run 'scripts/acc-garage.sh stop' first" >&2
		exit 1
	fi
fi

mkdir -p "$GARAGE_DIR/meta" "$GARAGE_DIR/data"
if [ -n "${GARAGE_BIN:-}" ]; then
	root="$GARAGE_DIR"
else
	root=/var/lib/garage
fi
cat >"$GARAGE_DIR/garage.toml" <<TOML
metadata_dir = "$root/meta"
data_dir = "$root/data"
db_engine = "sqlite"
replication_factor = 1
rpc_bind_addr = "127.0.0.1:3901"
rpc_public_addr = "127.0.0.1:3901"
rpc_secret = "$(openssl rand -hex 32)"

[s3_api]
s3_region = "garage"
api_bind_addr = "127.0.0.1:3900"
root_domain = ".s3.garage.localhost"

[s3_web]
bind_addr = "127.0.0.1:3902"
root_domain = ".web.garage.localhost"

[admin]
api_bind_addr = "127.0.0.1:3903"
admin_token = "$GARAGE_TOKEN"
TOML

if [ -n "${GARAGE_BIN:-}" ]; then
	"$GARAGE_BIN" -c "$GARAGE_DIR/garage.toml" server >"$GARAGE_DIR/log" 2>&1 &
	echo $! >"$GARAGE_DIR/pid"
	garage() { "$GARAGE_BIN" -c "$GARAGE_DIR/garage.toml" "$@"; }
else
	docker run -d --name "$CONTAINER" --network host \
		-v "$GARAGE_DIR/garage.toml:/etc/garage.toml:ro" \
		-v "$GARAGE_DIR/meta:$root/meta" -v "$GARAGE_DIR/data:$root/data" \
		"$GARAGE_IMAGE" >/dev/null
	garage() { docker exec "$CONTAINER" /garage "$@"; }
fi

ready=false
for _ in $(seq 60); do
	if garage status >/dev/null 2>&1; then
		ready=true
		break
	fi
	sleep 1
done

if [ "$ready" != true ]; then
	if [ -z "${GARAGE_BIN:-}" ]; then
		docker logs "$CONTAINER" >"$GARAGE_DIR/log" 2>&1 || true
	fi
	echo "error: Garage did not become ready after 60 attempts; see $GARAGE_DIR/log" >&2
	exit 1
fi

node_id="$(garage node id -q | cut -d@ -f1)"
garage layout assign "$node_id" -z dc1 -c 1G -t acc-a -t acc-b >/dev/null
garage layout apply --version 1 >/dev/null
echo "$node_id"
