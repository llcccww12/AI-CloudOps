#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")/.."

echo "==> 停止占用 8889 的进程"
PIDS="$(lsof -t -iTCP:8889 -sTCP:LISTEN 2>/dev/null || true)"
if [[ -n "${PIDS}" ]]; then
  # shellcheck disable=SC2086
  kill -9 ${PIDS} || true
  sleep 1
fi

echo "==> 编译 /tmp/cacops-main"
export GOMODCACHE="${GOMODCACHE:-$HOME/go/pkg/mod}"
export GOPROXY="${GOPROXY:-https://goproxy.cn,direct}"
go build -o /tmp/cacops-main .

echo "==> 启动后端"
: >/tmp/cacops-main.log
nohup env APP_ENV=development /tmp/cacops-main >>/tmp/cacops-main.log 2>&1 &
echo "pid=$!"
sleep 3
lsof -iTCP:8889 -sTCP:LISTEN -n -P || true
tail -n 20 /tmp/cacops-main.log
echo "==> 完成。日志: /tmp/cacops-main.log"
