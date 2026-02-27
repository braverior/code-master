# CodeMaster 飞书机器人

## 功能概述

CodeMaster 飞书机器人提供三大核心能力：

1. **通知推送** — 需求创建、代码生成完成/失败、AI Review 完成、人工审查提交时，自动推送飞书卡片消息给相关人员
2. **Bot 命令** — 用户在飞书与机器人对话，通过 `/` 命令查询项目、需求、审查状态
3. **AI 聊天** — 非命令文本转发到 OpenAI 兼容接口，支持上下文对话

机器人采用 **WebSocket 长连接** 模式，无需公网回调地址，适合内网部署。

## 架构

```
┌─────────────────────────────────────────────────────┐
│                    飞书服务端                         │
│         im.message.receive_v1 (WebSocket)            │
└──────────────────────┬──────────────────────────────┘
                       │ WebSocket 长连接
                       ▼
┌──────────────────────────────────────────────────────┐
│  internal/bot/bot.go                                  │
│  ┌─────────────────────────────────────────────────┐ │
│  │ dispatcher.EventDispatcher                       │ │
│  │   └─ OnP2MessageReceiveV1()                      │ │
│  └──────────────────┬──────────────────────────────┘ │
│                     ▼                                 │
│  ┌──────────────────────────────────────────────────┐│
│  │ handler.go: MessageHandler                       ││
│  │   ├─ /command → commands.go (CommandHandler)     ││
│  │   └─ free text → aichat.go (AIChatClient)       ││
│  └──────────────────────────────────────────────────┘│
└──────────────────────────────────────────────────────┘

┌──────────────────────────────────────────────────────┐
│  internal/notify/notifier.go                          │
│  ┌──────────────────────────────────────────────────┐│
│  │ FeishuNotifier                                   ││
│  │   └─ pkg/feishu/bot_client.go (BotClient)        ││
│  └──────────────────────────────────────────────────┘│
│  触发点：                                             │
│  ├─ handler/requirement.go  → NotifyRequirementCreated│
│  ├─ service/codegen.go      → NotifyCodegenCompleted  │
│  │                          → NotifyCodegenFailed     │
│  ├─ service/review.go       → NotifyAIReviewCompleted │
│  └─ service/review.go       → NotifyHumanReviewSubmitted│
└──────────────────────────────────────────────────────┘
```

## 飞书应用配置

### 1. 开启机器人能力

在飞书开放平台 → 应用详情 → 添加能力 → 机器人。

### 2. 所需权限

| 权限 | 说明 |
|------|------|
| `im:message` | 发送消息 |
| `im:message:send_as_bot` | 以机器人身份发送消息 |
| `im:message.receive_v1` | 接收消息事件 |
| `contact:user.id:readonly` | 读取用户 ID |

### 3. 事件订阅

订阅事件 `im.message.receive_v1`（接收消息）。

### 4. 长连接模式

在飞书开放平台 → 事件订阅 → 选择 **长连接** 模式（WebSocket），无需配置回调 URL。

## 配置项

在 `config.yaml` 中添加：

```yaml
feishu:
  app_id: "cli_xxx"
  app_secret: "xxx"
  redirect_uri: "http://localhost:30003/api/v1/auth/feishu/callback"
  bot:
    enabled: true                    # 是否启用机器人
    encrypt_key: ""                  # 事件加密 Key（可选）
    verification_token: ""           # 事件验证 Token（可选）

ai_chat:
  base_url: "https://api.openai.com/v1"  # OpenAI 兼容接口地址
  api_key: "sk-xxx"                       # API Key
  model: "gpt-4"                          # 模型名称
  max_history: 20                         # 每用户最大对话轮次
```

| 配置项 | 说明 | 必填 |
|--------|------|------|
| `feishu.bot.enabled` | 启用飞书机器人 | 否，默认 false |
| `feishu.bot.encrypt_key` | 飞书事件加密 Key | 否 |
| `feishu.bot.verification_token` | 飞书事件验证 Token | 否 |
| `ai_chat.base_url` | AI 接口地址 | 使用 AI 聊天时必填 |
| `ai_chat.api_key` | AI 接口密钥 | 使用 AI 聊天时必填 |
| `ai_chat.model` | AI 模型名称 | 使用 AI 聊天时必填 |
| `ai_chat.max_history` | 对话历史保留条数 | 否，默认 20 |

## 通知事件

| 事件 | 触发时机 | 通知对象 | 卡片颜色 |
|------|----------|----------|----------|
| 需求创建 | 创建需求并指派 assignee | 被指派人 | 蓝色 |
| 代码生成完成 | 代码生成任务成功 | 创建人 + 指派人 | 绿色 |
| 代码生成失败 | 代码生成任务失败 | 创建人 + 指派人 | 红色 |
| AI Review 完成 | AI 代码审查完成 | 创建人 + 指派人 | 绿/黄/红 |
| 人工审查提交 | 人工提交审查结果 | 创建人 + 指派人 | 绿/红 |

通知前提：用户的 `feishu_uid`（open_id）非空（即用户已通过飞书 OAuth 登录过 CodeMaster）。

所有通知均为异步发送（`go func()`），不阻塞业务流程。

## Bot 命令

| 命令 | 说明 | 示例 |
|------|------|------|
| `/help` | 显示命令列表 | `/help` |
| `/projects` | 我的项目列表 | `/projects` |
| `/reqs <项目ID>` | 指定项目的需求列表 | `/reqs 1` |
| `/status <需求ID>` | 需求详细状态（含代码生成和 Review） | `/status 42` |
| `/reviews` | 我的待审查列表 | `/reviews` |
| `/clear` | 清除 AI 对话历史 | `/clear` |

未识别的 `/xxx` 命令会提示"未知命令"。

命令功能需要用户已通过 CodeMaster Web 端完成飞书登录（系统根据 open_id 匹配用户）。

## AI 聊天

发送任意非 `/` 开头的文本，机器人会将消息转发到配置的 OpenAI 兼容接口并返回回复。

- 支持多轮对话，每用户独立维护对话历史
- 对话历史存储在内存中，服务重启后清空
- 使用 `/clear` 命令可手动清除对话历史
- 超过 `max_history` 限制会自动裁剪最早的消息
- System Prompt 设定为 CodeMaster Bot 编程助手角色，中文回复

### 兼容接口

AI 聊天使用标准 OpenAI Chat Completions API 格式：
- `POST {base_url}/chat/completions`
- 支持任何兼容 OpenAI API 的服务（如 Azure OpenAI、本地部署的 LLM 等）
