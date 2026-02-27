#!/bin/sh
set -e

echo "=== CodeMaster All-in-One ==="

# ---- Patch config.yaml with Secret env vars (if set) ----
CONFIG_SRC="${CONFIG_PATH:-/etc/codemaster/config.yaml}"
CONFIG_RUN="/tmp/config.yaml"
cp "$CONFIG_SRC" "$CONFIG_RUN"

# Database password
if [ -n "$DB_PASSWORD" ]; then
  sed -i "s|password: \"\"|password: \"${DB_PASSWORD}\"|" "$CONFIG_RUN"
fi
# JWT secret
if [ -n "$JWT_SECRET" ]; then
  sed -i "s|secret: \"change-me-to-a-random-string\"|secret: \"${JWT_SECRET}\"|" "$CONFIG_RUN"
fi
# AES key
if [ -n "$AES_KEY" ]; then
  sed -i "s|aes_key: \"change-me-32-char-hex-aes-key!!\"|aes_key: \"${AES_KEY}\"|" "$CONFIG_RUN"
fi
# Feishu
if [ -n "$FEISHU_APP_ID" ]; then
  sed -i "s|app_id: \"\"|app_id: \"${FEISHU_APP_ID}\"|" "$CONFIG_RUN"
fi
if [ -n "$FEISHU_APP_SECRET" ]; then
  sed -i "s|app_secret: \"\"|app_secret: \"${FEISHU_APP_SECRET}\"|" "$CONFIG_RUN"
fi
# AI Chat
if [ -n "$AI_CHAT_API_KEY" ]; then
  sed -i "s|api_key: \"\"|api_key: \"${AI_CHAT_API_KEY}\"|" "$CONFIG_RUN"
fi

export CONFIG_PATH="$CONFIG_RUN"

# ---- Start Nginx (background) ----
echo "Starting Nginx..."
nginx

# ---- Start backend (foreground) ----
echo "Starting CodeMaster backend..."
exec codemaster-server
