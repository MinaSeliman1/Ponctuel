#!/bin/sh
set -eu

port="${PORT:-10000}"
case "$port" in
    *[!0-9]*|'')
        echo "PORT must be a number" >&2
        exit 1
        ;;
esac

mkdir -p /tmp/nginx/client_temp /tmp/nginx/proxy_temp /tmp/nginx/fastcgi_temp /tmp/nginx/uwsgi_temp /tmp/nginx/scgi_temp
sed "s/__PORT__/${port}/g" /app/nginx.conf.template > /tmp/nginx.conf

/app/migrate

/app/ingester &
ingester_pid=$!
/app/api &
api_pid=$!
nginx -c /tmp/nginx.conf -g 'daemon off;' &
nginx_pid=$!

cleanup() {
    kill "$nginx_pid" "$api_pid" "$ingester_pid" 2>/dev/null || true
    wait "$nginx_pid" "$api_pid" "$ingester_pid" 2>/dev/null || true
}
trap cleanup INT TERM EXIT

while kill -0 "$nginx_pid" 2>/dev/null &&
      kill -0 "$api_pid" 2>/dev/null &&
      kill -0 "$ingester_pid" 2>/dev/null; do
    sleep 2
done

exit 1
