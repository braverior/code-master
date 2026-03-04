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
ENV GOPROXY=https://goproxy.cn,direct
WORKDIR /build
COPY backend/go.mod backend/go.sum ./
RUN go mod download
COPY backend/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /app/server ./cmd/server

# ---------- Stage 3: Runtime ----------
FROM node:20-slim

# Use China mirror for faster apt downloads
RUN sed -i 's|deb.debian.org|mirrors.aliyun.com|g' /etc/apt/sources.list.d/debian.sources

# Layer 1: System base tools (almost never changes)
RUN apt-get update && apt-get install -y --no-install-recommends \
    git \
    ca-certificates \
    nginx \
    tini \
    procps \
    vim \
    curl \
    bash-completion \
    && rm -rf /var/lib/apt/lists/*

# Layer 2: Language runtimes via apt (change when adding/upgrading languages)
RUN apt-get update && apt-get install -y --no-install-recommends \
    default-jdk-headless \
    php-cli \
    php-common \
    redis \
    default-mysql-server \
    default-mysql-client \
    && rm -rf /var/lib/apt/lists/*

# Go: copy from official image instead of apt-get (much faster, version 1.22 vs apt's 1.19)
COPY --from=golang:1.22-alpine /usr/local/go /usr/local/go

# Go environment
ENV GOPATH=/root/go
ENV PATH=$GOPATH/bin:/usr/local/go/bin:$PATH
ENV GOPROXY=https://goproxy.cn,direct


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

EXPOSE 8080
WORKDIR /app

ENTRYPOINT ["tini", "--"]
CMD ["/entrypoint.sh"]
