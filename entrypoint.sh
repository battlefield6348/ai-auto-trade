#!/bin/sh
set -e

# 如果有 DSN 設定，就執行 Migration
if [ -n "$DB_DSN" ] || [ -f "config.yaml" ]; then
    echo "Running database migrations..."
    # 這裡假設 DSN 可能在環境變數或 config.yaml
    # 我們嘗試執行，如果失敗但不影響啟動可以改為 non-fatal，
    # 但通常 Migration 失敗應該要停止。
    migrate -dir db/migrations || echo "Migration skipped or failed (check DSN)"
fi

echo "Starting application..."
exec ai-auto-trade
