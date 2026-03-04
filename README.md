# CodeMaster

AI 驱动的代码生成与审查平台。通过对接 Claude Code CLI，实现从需求到代码的自动生成、AI 审查、人工审查、合并请求的完整工作流。

## 功能特性

- **需求管理** — 创建需求、关联飞书文档、设置优先级与截止日期
- **AI 代码生成** — 基于 Claude Code CLI，根据需求自动生成代码并推送到 Git 分支
- **会话恢复 (Session Resume)** — 二次生成时可恢复上一次 Claude 会话，保留上下文实现迭代开发
- **实时流式输出** — SSE 实时展示 Claude 的思考过程、工具调用、代码变更
- **AI 代码审查** — 自动对生成的代码进行质量评估、问题检测
- **人工审查** — 支持通过/拒绝/需修改的审查流程
- **合并请求** — 一键创建 GitLab/GitHub Merge Request
- **仓库分析** — 自动分析仓库技术栈、模块结构、代码风格
- **飞书集成** — 飞书 OAuth 登录、Bot 通知、文档内容抓取
- **项目协作** — 多项目、多成员、角色权限管理

* Dashbaord
<img width="2982" alt="image" src="https://github.com/user-attachments/assets/ff75aeca-2c3e-4e44-b517-4f4f0139ea2e" />

* Guide
<img width="2988" alt="image" src="https://github.com/user-attachments/assets/7ea66735-5594-4fd6-9283-54e36ea508d8" />

* Coding
<img width="2982" alt="image" src="https://github.com/user-attachments/assets/28649b74-cdfa-4a49-bbd0-84920f946669" />


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
| `codegen.session_dir` | Claude HOME 目录，持久化 session 数据 (空=系统 HOME；K8s 推荐 `/data/work/claude-home`) |
| `encrypt.aes_key` | AES 加密密钥 (用于加密 git token) |
| `feishu.*` | 飞书应用配置 (OAuth 登录、Bot) |
| `ai_chat.*` | AI 对话助手配置 |

用户可在「设置」页面配置个人的 LLM API Key 和 Git Token，按用户维度加密存储。

## 飞书集成

CodeMaster 深度集成飞书，提供 OAuth 登录、机器人交互、消息通知和文档抓取能力。

### 创建飞书应用

1. 登录 [飞书开放平台](https://open.feishu.cn/)，创建企业自建应用
2. 获取 `App ID` 和 `App Secret`
3. 添加以下能力与配置

### 所需能力

| 能力 | 用途 |
|------|------|
| **网页应用** | 配置 OAuth 重定向 URL，实现飞书账号登录 CodeMaster |
| **机器人** | 在飞书中与 CodeMaster Bot 交互，接收通知推送 |

### 所需权限

| 权限标识 | 说明 | 用途 |
|----------|------|------|
| `im:message` | 获取与发送单聊、群组消息 | Bot 发送通知卡片 |
| `im:message:send_as_bot` | 以应用的身份发送消息 | Bot 主动推送通知 |
| `im:message.receive_v1` | 接收消息事件 | Bot 接收用户命令和聊天消息 |
| `contact:user.id:readonly` | 获取用户 UserID | OAuth 登录时获取用户标识 |

### 事件订阅

- 订阅 `im.message.receive_v1`（接收消息）事件
- 连接模式选择 **长连接（WebSocket）**，无需公网回调地址，适合内网部署

### OAuth 登录

用户通过飞书 OAuth 2.0 登录 CodeMaster，流程：

```
浏览器 → /auth/feishu/login → 飞书授权页 → 用户授权
    → /auth/feishu/callback → 获取 user_access_token → 获取用户信息
    → 创建/更新用户 → 签发 JWT → 跳转前端
```

首次登录自动创建用户，后续登录更新用户名和头像。用户的 `feishu_uid`（open_id）用于关联 Bot 消息推送。

### 机器人能力

**Bot 命令：**

| 命令 | 说明 |
|------|------|
| `/help` | 显示命令列表 |
| `/projects` | 我的项目列表 |
| `/myreqs` | 我的所有需求 |
| `/reqs <项目ID>` | 指定项目的需求列表 |
| `/status <需求ID>` | 需求详细状态（含代码生成和 Review 信息） |
| `/reviews` | 我的待审查列表 |
| `/codegen <需求ID>` | 触发代码生成 |
| `/clear` | 清除 AI 对话历史 |

**通知推送：**

| 事件 | 触发时机 | 通知对象 | 卡片颜色 |
|------|----------|----------|----------|
| 需求创建 | 创建需求并指派 | 被指派人 | 蓝色 |
| 代码生成完成 | 生成任务成功 | 创建人 + 指派人 | 绿色 |
| 代码生成失败 | 生成任务失败 | 创建人 + 指派人 | 红色 |
| AI Review 完成 | AI 审查完成 | 创建人 + 指派人 | 按评分着色 |
| 人工审查提交 | 人工提交审查 | 创建人 + 指派人 | 按结果着色 |

**AI 聊天：** 发送非 `/` 开头的文本，Bot 转发到 OpenAI 兼容接口，支持多轮对话。

### 飞书相关配置

```yaml
feishu:
  app_id: "cli_xxx"                                          # 飞书应用 App ID
  app_secret: "xxx"                                          # 飞书应用 App Secret
  redirect_uri: "https://your-domain/api/v1/auth/feishu/callback"  # OAuth 回调地址
  bot:
    enabled: true              # 启用机器人（关闭则仅 OAuth 登录可用）
    encrypt_key: ""            # 事件加密 Key（可选）
    verification_token: ""     # 事件验证 Token（可选）

ai_chat:                       # Bot AI 聊天（可选）
  base_url: "https://api.openai.com/v1"
  api_key: "sk-xxx"
  model: "gpt-4"
  max_history: 20
```

> 详细文档参见 [`docs/feishu-bot.md`](docs/feishu-bot.md)。

## Session Resume (会话恢复)

每次代码生成时，系统会从 Claude Code 的 `system/init` 事件中捕获 `session_id` 并持久化到数据库。用户在二次生成时，可以勾选「恢复上次会话」，选择一个历史 session，Claude 将通过 `--resume <session_id>` 继续上一次的上下文。

**关键设计：**

- **稳定工作目录** — 每个需求的工作目录固定为 `codegen/req-<requirement_id>`，确保 Claude session 的 project path 编码一致，`--resume` 能找到历史 session
- **HOME 隔离** — 仅对 Claude 子进程设置 `HOME=session_dir`，session 文件存储在 `<session_dir>/.claude/` 下，不影响主程序
- **K8s 持久化** — 通过 PVC 挂载 `session_dir`，Pod 重启后 session 数据不丢失

```
/data/work/                         (PVC 挂载)
├── codegen/req-123/                ← 需求 123 的代码仓库 (work_dir)
├── codegen/req-456/                ← 需求 456 的代码仓库
└── claude-home/.claude/            ← Claude session 文件 (session_dir)
```

本地开发时 `session_dir` 留空即可，session 存储在用户的 `~/.claude/` 目录。

## API 文档

详见 [`docs/api-specification.md`](docs/api-specification.md)。
