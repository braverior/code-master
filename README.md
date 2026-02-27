# CodeMaster

AI 驱动的代码生成与审查平台。通过对接 Claude Code CLI，实现从需求到代码的自动生成、AI 审查、人工审查、合并请求的完整工作流。

## 功能特性

- **需求管理** — 创建需求、关联飞书文档、设置优先级与截止日期
- **AI 代码生成** — 基于 Claude Code CLI，根据需求自动生成代码并推送到 Git 分支
- **实时流式输出** — SSE 实时展示 Claude 的思考过程、工具调用、代码变更
- **AI 代码审查** — 自动对生成的代码进行质量评估、问题检测
- **人工审查** — 支持通过/拒绝/需修改的审查流程
- **合并请求** — 一键创建 GitLab/GitHub Merge Request
- **仓库分析** — 自动分析仓库技术栈、模块结构、代码风格
- **飞书集成** — 飞书 OAuth 登录、Bot 通知、文档内容抓取
- **项目协作** — 多项目、多成员、角色权限管理
介绍页：
<img width="2974" height="1542" alt="image" src="https://github.com/user-attachments/assets/bbd29163-14a7-462f-81dd-00225669740a" />
看板页：
<img width="2986" height="1534" alt="image" src="https://github.com/user-attachments/assets/97ab2323-3dd3-4919-be26-ee7426219ea6" />
项目需求页面：
<img width="2978" height="1522" alt="image" src="https://github.com/user-attachments/assets/6acb86dc-6034-4d15-a525-ebbc023d80bf" />
AI Coding页面：
<img width="2954" height="1514" alt="image" src="https://github.com/user-attachments/assets/802d95f3-b1b9-40bc-a073-6e0dbde282b9" />
设置页面：
<img width="2978" height="1544" alt="image" src="https://github.com/user-attachments/assets/6dec0f70-651c-4366-9bf1-a6524a60cb2d" />



## 技术栈

| 层 | 技术 |
|---|---|
| 前端 | React 18 + TypeScript + Vite + Tailwind CSS + shadcn/ui |
| 后端 | Go 1.22 + Gin + GORM |
| 数据库 | MySQL 8.0 |
| 缓存 | Redis 7 (SSE 事件流) |
| AI | Claude Code CLI + Anthropic API |
| Git | GitLab / GitHub (通过 token 认证) |

## 项目结构

```
code-master/
├── backend/                 # Go 后端
│   ├── cmd/server/          # 入口
│   ├── internal/
│   │   ├── config/          # 配置加载 (Viper)
│   │   ├── handler/         # HTTP Handler
│   │   ├── middleware/       # JWT 鉴权、RBAC
│   │   ├── model/           # 数据模型
│   │   ├── router/          # 路由
│   │   ├── service/         # 业务逻辑
│   │   ├── codegen/         # 代码生成 (Executor, Analyzer, Pool)
│   │   ├── review/          # AI 审查
│   │   ├── gitops/          # Git 操作 (clone, push, diff)
│   │   ├── sse/             # SSE Hub (Redis-backed)
│   │   ├── bot/             # 飞书 Bot
│   │   └── notify/          # 通知
│   └── pkg/                 # 工具包 (encrypt, feishu, claude)
├── frontend/                # React 前端
│   └── src/
│       ├── api/             # API 调用
│       ├── components/      # UI 组件
│       ├── hooks/           # React Hooks
│       ├── pages/           # 页面
│       └── types/           # TypeScript 类型
├── deploy/                  # 部署配置
│   ├── nginx.conf           # Nginx 配置
│   ├── entrypoint.sh        # 容器入口脚本
│   ├── deploy.sh            # 一键部署脚本
│   └── k8s/                 # Kubernetes 清单
├── docs/                    # API 文档
├── Dockerfile               # 多阶段构建
└── work/                    # 运行时工作目录 (codegen/analysis)
```

## 本地开发

### 前置条件

- Go 1.22+
- Node.js 20+
- MySQL 8.0+
- Redis 7+
- [Claude Code CLI](https://docs.anthropic.com/en/docs/claude-code) (`npm install -g @anthropic-ai/claude-code`)
- Git

### 后端

```bash
cd backend

# 复制并编辑配置
cp config.yaml config.local.yaml
# 修改 database、redis、feishu、jwt 等配置

# 启动
CONFIG_PATH=config.local.yaml go run ./cmd/server
```

后端默认监听 `:30003`。

### 前端

```bash
cd frontend
npm install
npm run dev
```

前端默认监听 `:3000`，自动代理 `/api` 到 `http://127.0.0.1:30003`。

## Kubernetes 部署

All-in-one 镜像包含：Go 后端 + 前端静态文件(Nginx) + Git + Claude Code CLI。

```
┌─── Pod: codemaster ──────────────────────────────────┐
│  Nginx (:80)                                         │
│    ├── /        → frontend SPA                       │
│    └── /api/*   → reverse proxy → Go backend (:30003)│
│  Go Backend + Git + Claude Code CLI                  │
│  /data/work (PVC)                                    │
└──────────────────────────────────────────────────────┘
   +  MySQL Pod  +  Redis Pod
```

### 快速部署

```bash
# 1. 编辑 Secret（数据库密码、JWT 密钥等）
vim deploy/k8s/secret.yaml

# 2. 编辑 ConfigMap（数据库地址、飞书配置等）
vim deploy/k8s/configmap.yaml

# 3. 编辑 Ingress（域名）
vim deploy/k8s/ingress.yaml

# 4. 构建并部署
cd deploy
./deploy.sh all

# 推送到私有仓库
REGISTRY=registry.example.com/team ./deploy.sh all
```

### 部署管理

```bash
./deploy.sh build    # 构建 Docker 镜像
./deploy.sh push     # 推送到镜像仓库
./deploy.sh apply    # 应用 K8s 清单
./deploy.sh delete   # 清理所有资源
```

详细 K8s 配置文件见 [`deploy/k8s/`](deploy/k8s/)。

## 配置说明

| 配置项 | 说明 |
|--------|------|
| `server.port` | 后端端口 (默认 30003) |
| `database.*` | MySQL 连接信息 |
| `redis.addr` | Redis 地址 |
| `jwt.secret` | JWT 签名密钥 |
| `codegen.max_workers` | 最大并行代码生成任务数 |
| `codegen.max_turns` | Claude 最大交互轮数 |
| `codegen.timeout_minutes` | 单次生成超时(分钟) |
| `codegen.work_dir` | 工作目录路径 |
| `codegen.use_local_git` | true=本地 git 凭证推送，false=token 推送 |
| `encrypt.aes_key` | AES 加密密钥 (用于加密 git token) |
| `feishu.*` | 飞书应用配置 (OAuth 登录、Bot) |
| `ai_chat.*` | AI 对话助手配置 |

用户可在「设置」页面配置个人的 LLM API Key 和 Git Token，按用户维度加密存储。

## API 文档

详见 [`docs/api-specification.md`](docs/api-specification.md)。
