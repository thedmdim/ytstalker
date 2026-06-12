#!/bin/sh

CERT=/acme.sh/${DOMAIN}_ecc/fullchain.cer
KEY=/acme.sh/${DOMAIN}_ecc/$DOMAIN.key
HAPROXY_CERT=/etc/haproxy/certs/front.pem

mkdir -p /etc/haproxy/certs

# начальная генерация — серт уже может лежать в volume
if [ -f "$CERT" ] && [ -f "$KEY" ]; then
    cat "$CERT" "$KEY" > "$HAPROXY_CERT"
fi

# запускаем haproxy
haproxy -f /usr/local/etc/haproxy/haproxy.cfg -W &

# ждём пока серт появится если его ещё нет
while [ ! -f "$CERT" ]; do
    echo "waiting for cert..."
    sleep 3s
done

# следим за обновлениями
while inotifywait -e close_write "$CERT"; do
    cat "$CERT" "$KEY" > "$HAPROXY_CERT"
    kill -USR2 1
    echo "cert reloaded: $(date)"
done