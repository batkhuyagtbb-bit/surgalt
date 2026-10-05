#!/bin/sh
# Локал (Docker-гүй) ClickHouse: нэг binary-г татаад ./data/clickhouse дотор ажиллуулна.
#   sh deploy/clickhouse-local.sh          # асаана (:9000 native, :8123 http)
#   CLICKHOUSE_DSN=clickhouse://127.0.0.1:9000/surgalt ./surgalt
set -e
DIR="${CLICKHOUSE_HOME:-$HOME/.local/clickhouse}"
mkdir -p "$DIR/data"
if [ ! -x "$DIR/clickhouse" ]; then
  echo "ClickHouse татаж байна → $DIR/clickhouse"
  curl -fsSL https://clickhouse.com/ | sh -s -- >/dev/null 2>&1 || true
  [ -x ./clickhouse ] && mv ./clickhouse "$DIR/clickhouse"
fi
cat > "$DIR/config.xml" <<XML
<clickhouse>
  <logger><level>warning</level><console>1</console></logger>
  <listen_host>127.0.0.1</listen_host>
  <tcp_port>${CLICKHOUSE_TCP_PORT:-9000}</tcp_port>
  <http_port>${CLICKHOUSE_HTTP_PORT:-8123}</http_port>
  <path>$DIR/data/</path>
  <tmp_path>$DIR/data/tmp/</tmp_path>
  <user_files_path>$DIR/data/user_files/</user_files_path>
  <users><default><password></password><networks><ip>::/0</ip></networks><profile>default</profile><quota>default</quota><access_management>1</access_management></default></users>
  <profiles><default/></profiles><quotas><default/></quotas>
</clickhouse>
XML
echo "ClickHouse эхэлж байна: native :${CLICKHOUSE_TCP_PORT:-9000}, http :${CLICKHOUSE_HTTP_PORT:-8123} (өгөгдөл: $DIR/data)"
exec "$DIR/clickhouse" server --config-file="$DIR/config.xml"
