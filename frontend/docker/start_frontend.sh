#!/bin/sh
set -eu

fail() { printf '%s\n' "$1" >&2; exit 1; }
valid_port() {
    case "$1" in ''|*[!0-9]*) return 1 ;; esac
    [ "$1" -ge 1 ] && [ "$1" -le 65535 ]
}

WEB_PORT="${PORT:-8080}"
valid_port "$WEB_PORT" || fail 'Invalid PORT'
target="${API_PROXY_TARGET:-http://backend:8080}"
target="${target%/}"
case "$target" in
    https://*) API_SCHEME=https; API_UPSTREAM="${target#https://}" ;;
    http://*) API_SCHEME=http; API_UPSTREAM="${target#http://}" ;;
    *) fail 'API_PROXY_TARGET must be an HTTP(S) origin' ;;
esac
case "$API_UPSTREAM" in *[!A-Za-z0-9.:-]*) fail 'Invalid characters in API_PROXY_TARGET' ;; esac
printf '%s' "$API_UPSTREAM" | grep -Eq '^[A-Za-z0-9]([A-Za-z0-9.-]*[A-Za-z0-9])?(:[0-9]{1,5})?$' || fail 'Invalid API_PROXY_TARGET hostname or port'
API_UPSTREAM="$(printf '%s' "$API_UPSTREAM" | tr '[:upper:]' '[:lower:]')"
API_HOST="${API_UPSTREAM%%:*}"
API_AUTHORITY="$API_UPSTREAM"
case "$API_UPSTREAM" in *:*) valid_port "${API_UPSTREAM##*:}" || fail 'Invalid upstream port' ;; esac
case "$API_HOST" in *.onrender.com) [ "$API_SCHEME" = https ] || fail 'Render public upstreams require HTTPS' ;; esac
if [ -n "${RENDER_EXTERNAL_HOSTNAME:-}" ] && [ "$API_HOST" = "$RENDER_EXTERNAL_HOSTNAME" ]; then
    fail 'API_PROXY_TARGET cannot point back to the frontend service'
fi

DNS_RESOLVER="${DNS_RESOLVER:-$(awk '/^nameserver[[:space:]]/{print $2; exit}' /etc/resolv.conf)}"
case "$DNS_RESOLVER" in *[!0-9A-Fa-f:.]*) fail 'Invalid system DNS resolver' ;; esac
case "$DNS_RESOLVER" in *:*) DNS_RESOLVER="[$DNS_RESOLVER]" ;; esac
printf '%s' "$DNS_RESOLVER" | grep -Eq '^([0-9]{1,3}\.){3}[0-9]{1,3}$|^\[[0-9A-Fa-f:]+\]$' || fail 'Invalid system DNS resolver'
export WEB_PORT API_SCHEME API_UPSTREAM API_HOST API_AUTHORITY DNS_RESOLVER
envsubst "\${WEB_PORT} \${API_SCHEME} \${API_UPSTREAM} \${API_HOST} \${API_AUTHORITY} \${DNS_RESOLVER}" \
    < /etc/nginx/techsnacks.template > /tmp/techsnacks-nginx.conf
nginx -t -c /tmp/techsnacks-nginx.conf
if [ "$#" -eq 0 ]; then set -- nginx -c /tmp/techsnacks-nginx.conf -g 'daemon off;'; fi
exec "$@"
