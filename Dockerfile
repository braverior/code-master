# ============================================================
# CodeMaster All-in-One Image
# Includes: Go backend + Frontend (Nginx) + Git + Claude Code CLI
# ============================================================

# ---------- Stage 1: Build frontend ----------
FROM node:20-alpine AS frontend-builder
WORKDIR /build
COPY frontend/package.json frontend/package-lock.json ./
RUN npm ci --ignore-scripts
COPY frontend/ ./
RUN npm run build

# ---------- Stage 2: Build backend ----------
FROM golang:1.22-alpine AS backend-builder
RUN apk add --no-cache gcc musl-dev
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/server ./cmd/server

# ---------- Stage 3: Runtime ----------
FROM node:20-slim

# Install runtime dependencies
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    nginx \
    tini \
    && rm -rf /var/lib/apt/lists/*

# Install Claude Code CLI
RUN npm install -g @anthropic-ai/claude-code \
    && npm cache clean --force

# Copy backend binary
COPY --from=backend-builder /app/server /usr/local/bin/codemaster-server

# Copy frontend dist to nginx root
COPY --from=frontend-builder /build/dist /usr/share/nginx/html

# Copy nginx config
COPY deploy/nginx.conf /etc/nginx/conf.d/default.conf
RUN rm -f /etc/nginx/sites-enabled/default

# Copy entrypoint
COPY deploy/entrypoint.sh /entrypoint.sh
RUN chmod +x /entrypoint.sh

# Create work directory
RUN mkdir -p /data/work

# Git global config (for codegen commits)
RUN git config --global init.defaultBranch main \
    && git config --global http.sslVerify true

# Default environment
ENV GIT_TERMINAL_PROMPT=0
ENV CONFIG_PATH=/etc/codemaster/config.yaml

EXPOSE 80
WORKDIR /app

ENTRYPOINT ["tini", "--"]
CMD ["/entrypoint.sh"]
