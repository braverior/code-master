# CodeMaster 实现路径

---

## 一、技术选型

| 层面 | 技术 | 版本 | 说明 |
|------|------|------|------|
| 前端框架 | React + TypeScript | 18.x | SPA |
| 前端构建 | Vite | 5.x | |
| 前端 UI | Ant Design | 5.x | 企业级组件库 |
| 前端状态 | Zustand | 4.x | 轻量状态管理 |
| 前端路由 | React Router | 6.x | |
| 后端语言 | Go | 1.22+ | |
| Web 框架 | Gin | 1.9+ | |
| ORM | GORM | 2.x | |
| 数据库 | MySQL | 8.0+ | |
| 缓存 | Redis | 7.x | SSE 事件缓存、会话管理 |
| 代码生成 | Claude Code CLI | latest | 子进程调用 |
| Git 操作 | go-git + CLI | | 仓库克隆/分支管理 |
| Git 平台 API | gitlab-go / go-github | | MR 创建 |
| 飞书 SDK | oapi-sdk-go | | OAuth + 文档 API |

---

## 二、项目目录结构

```
code-master/
├── docs/                          # 文档
│   ├── database-schema.md
│   ├── api-specification.md
│   └── implementation-guide.md
│
├── frontend/
│   ├── public/
│   ├── src/
│   │   ├── api/                   # 接口请求层
│   │   │   ├── client.ts          # axios 实例、拦截器
│   │   │   ├── auth.ts
│   │   │   ├── project.ts
│   │   │   ├── repository.ts
│   │   │   ├── requirement.ts
│   │   │   ├── codegen.ts
│   │   │   └── review.ts
│   │   ├── components/            # 通用组件
│   │   │   ├── Layout/            # 整体布局 (侧边栏/顶栏)
│   │   │   ├── PrivateRoute.tsx   # 路由守卫
│   │   │   └── RoleGuard.tsx      # 角色权限守卫
│   │   ├── pages/
│   │   │   ├── Login/             # 飞书 OAuth 登录 + 角色选择
│   │   │   │   ├── index.tsx
│   │   │   │   └── RoleSelect.tsx
│   │   │   ├── Dashboard/         # 首页仪表盘
│   │   │   │   └── index.tsx
│   │   │   ├── Projects/          # 项目管理
│   │   │   │   ├── List.tsx       # 项目列表
│   │   │   │   ├── Detail.tsx     # 项目详情 (含仓库/成员/需求 tab)
│   │   │   │   └── Create.tsx     # 创建项目
│   │   │   ├── Requirements/      # 需求管理
│   │   │   │   ├── List.tsx       # 需求列表
│   │   │   │   ├── Detail.tsx     # 需求详情
│   │   │   │   └── Create.tsx     # 创建需求
│   │   │   ├── CodeGen/           # 代码生成 (核心页面)
│   │   │   │   ├── index.tsx      # 生成面板主页面
│   │   │   │   ├── StreamView.tsx # 实时输出流展示
│   │   │   │   ├── DiffView.tsx   # 代码 Diff 查看
│   │   │   │   ├── ProgressBar.tsx# 阶段进度条
│   │   │   │   └── FileList.tsx   # 变更文件列表
│   │   │   ├── Review/            # 代码 Review
│   │   │   │   ├── AIReview.tsx   # AI Review 结果展示
│   │   │   │   ├── HumanReview.tsx# 人工 Review 表单
│   │   │   │   └── MergeRequest.tsx# MR 状态
│   │   │   └── Admin/             # 管理后台
│   │   │       └── Users.tsx      # 用户管理
│   │   ├── hooks/
│   │   │   ├── useAuth.ts         # 认证 hook
│   │   │   ├── useCodeGenStream.ts# SSE 流式数据 hook (核心)
│   │   │   └── usePermission.ts   # 权限判断
│   │   ├── stores/
│   │   │   ├── authStore.ts       # 用户/认证状态
│   │   │   └── codegenStore.ts    # 代码生成状态
│   │   ├── utils/
│   │   │   ├── token.ts           # JWT 存取
│   │   │   └── constants.ts       # 常量
│   │   ├── App.tsx
│   │   ├── router.tsx             # 路由配置
│   │   └── main.tsx
│   ├── index.html
│   ├── package.json
│   ├── tsconfig.json
│   └── vite.config.ts
│
├── backend/
│   ├── cmd/
│   │   └── server/
│   │       └── main.go            # 入口: 初始化配置、DB、路由、启动服务
│   ├── internal/
│   │   ├── config/
│   │   │   └── config.go          # 配置结构体 + 加载 (Viper)
│   │   ├── handler/               # HTTP Handler 层
│   │   │   ├── auth.go            # 飞书 OAuth 回调、获取用户信息
│   │   │   ├── user.go            # 用户管理 (Admin)
│   │   │   ├── project.go         # 项目 CRUD + 成员管理
│   │   │   ├── repository.go      # 仓库关联 + 分析
│   │   │   ├── requirement.go     # 需求 CRUD
│   │   │   ├── codegen.go         # 代码生成触发 + SSE 流 + 取消
│   │   │   └── review.go          # Review 触发 + 人工评审 + 创建 MR
│   │   ├── middleware/
│   │   │   ├── auth.go            # JWT 解析 + 注入用户上下文
│   │   │   ├── rbac.go            # 角色/权限校验
│   │   │   └── cors.go            # CORS 配置
│   │   ├── model/                 # GORM 模型
│   │   │   ├── user.go
│   │   │   ├── project.go
│   │   │   ├── project_member.go
│   │   │   ├── repository.go
│   │   │   ├── requirement.go
│   │   │   ├── codegen_task.go
│   │   │   ├── code_review.go
│   │   │   └── operation_log.go
│   │   ├── service/               # 业务逻辑层
│   │   │   ├── auth.go            # 飞书 token 换取、用户创建/查找
│   │   │   ├── project.go
│   │   │   ├── repository.go      # 仓库管理 + 触发分析
│   │   │   ├── requirement.go
│   │   │   ├── codegen.go         # 代码生成任务调度 (核心)
│   │   │   └── review.go          # AI Review + 人工 Review 流程
│   │   ├── codegen/               # 代码生成核心引擎 (核心)
│   │   │   ├── executor.go        # Claude Code 子进程管理
│   │   │   ├── prompt.go          # Prompt 构造器
│   │   │   ├── stream_parser.go   # stream-json 逐行解析
│   │   │   ├── analyzer.go        # 仓库功能分析 (也基于 Claude Code)
│   │   │   └── pool.go            # 任务执行池 (goroutine pool)
│   │   ├── review/                # Review 引擎
│   │   │   ├── ai_reviewer.go     # AI Review 调用 Claude Code
│   │   │   └── prompt.go          # Review Prompt 模板
│   │   ├── gitops/                # Git 操作封装
│   │   │   ├── clone.go           # git clone (带 token 鉴权)
│   │   │   ├── branch.go          # 分支创建/切换/推送
│   │   │   ├── diff.go            # diff 获取与解析
│   │   │   └── merge_request.go   # GitLab/GitHub MR API 封装
│   │   ├── sse/                   # SSE 推送引擎
│   │   │   └── hub.go             # 订阅/广播/历史回放
│   │   └── router/
│   │       └── router.go          # 路由注册
│   ├── pkg/
│   │   ├── feishu/                # 飞书 SDK 封装
│   │   │   ├── oauth.go           # OAuth2 授权流程
│   │   │   └── doc.go             # 飞书文档内容获取
│   │   ├── jwt/
│   │   │   └── jwt.go             # JWT 签发与验证
│   │   └── encrypt/
│   │       └── aes.go             # access_token 加解密
│   ├── migrations/                # 数据库迁移文件
│   │   ├── 000001_init_schema.up.sql
│   │   └── 000001_init_schema.down.sql
│   ├── config.yaml                # 配置文件模板
│   ├── go.mod
│   └── go.sum
│
├── docker-compose.yml             # MySQL + Redis + 前后端
├── Makefile                       # 常用命令
└── .gitignore
```

---

## 三、分阶段实现路径

### 阶段一: 基础框架搭建

**目标:** 前后端项目初始化，跑通认证流程。

**后端任务:**

1. 初始化 Go 项目 (`go mod init`)
2. 引入依赖: Gin, GORM, Redis, Viper, jwt-go, oapi-sdk-go
3. 实现配置加载 (`config/config.go`)
4. 数据库连接 + AutoMigrate 所有模型
5. 实现中间件: CORS, JWT Auth, RBAC
6. 实现飞书 OAuth 全流程:
   - `/auth/feishu/login` → 重定向飞书授权页
   - `/auth/feishu/callback` → code 换 token → 查找/创建用户 → 签发 JWT
   - `/auth/me` → 返回当前用户
   - `/auth/role` → 角色选择/修改
7. 实现 Admin 用户管理接口
8. 编写 `docker-compose.yml` (MySQL + Redis)

**前端任务:**

1. Vite + React + TypeScript 初始化
2. 引入 Ant Design, React Router, Zustand, Axios
3. 实现 Layout 组件 (侧边栏导航 + 顶栏用户信息)
4. 实现登录页 + 角色选择页
5. 实现 axios 拦截器 (自动携带 token, 401 跳转登录)
6. 实现路由守卫 (PrivateRoute + RoleGuard)
7. 实现 Admin 用户管理页面

**交付物:** 用户可通过飞书登录，选择角色，Admin 可管理用户。

---

### 阶段二: 项目与需求管理

**目标:** 完成项目 CRUD、成员管理、仓库关联、需求 CRUD。

**后端任务:**

1. 实现项目 CRUD Handler + Service
2. 实现项目成员管理 (添加/移除)
3. 实现仓库关联:
   - 接收 git_url + access_token
   - 验证 token 有效性 (`git ls-remote`)
   - 存储 (access_token 加密)
4. 实现需求 CRUD Handler + Service
5. 实现飞书文档内容抓取 (`pkg/feishu/doc.go`):
   - 调用飞书开放平台文档 API
   - 解析富文本为纯文本
   - 存入 `requirements.doc_content`

**前端任务:**

1. 项目列表页 + 创建项目弹窗/页面
2. 项目详情页 (Tab: 概览 / 成员 / 仓库 / 需求)
3. 仓库关联表单 (填写 git_url, platform, token)
4. 需求列表页 + 创建需求页面 (富文本编辑器 or Markdown)
5. 需求详情页

**交付物:** PM 可创建项目、关联 RD、创建需求；RD 可关联代码仓库。

---

### 阶段三: 仓库分析引擎

**目标:** 实现仓库自动分析功能，为代码生成提供上下文。

**后端任务:**

1. 实现 `gitops/clone.go`:
   - 支持 HTTPS + token 方式 clone
   - clone 到临时目录 `/tmp/codemaster/repos/<repo-id>/`
   - 只做 shallow clone (`--depth 1`) 加速
2. 实现 `codegen/analyzer.go`:
   - clone 仓库到临时目录
   - 调用 `claude -p "<分析prompt>" --output-format json --allowedTools Read,Glob,Grep`
   - 工作目录设为仓库目录
   - 解析 JSON 输出，写入 `repositories.analysis_result`
3. 分析 Prompt 设计:
   ```
   分析这个代码仓库的结构和功能。输出严格 JSON 格式:
   {
     "modules": [{"path": "", "description": "", "files_count": 0}],
     "tech_stack": [],
     "entry_points": [],
     "directory_structure": "",
     "code_style": {"naming": "", "error_handling": "", "test_framework": ""}
   }
   只输出 JSON，不要任何其他内容。
   ```
4. 异步执行: 分析任务通过 goroutine 异步执行，前端轮询 `analysis_status`

**前端任务:**

1. 仓库卡片增加"分析"按钮 + 分析状态展示
2. 分析结果展示面板 (模块列表/技术栈/项目结构)

**交付物:** 关联仓库后可一键分析，查看仓库模块和技术栈。

---

### 阶段四: 代码生成引擎 (核心)

**目标:** 实现一键代码生成 + 实时流式输出展示。

**后端任务:**

1. 实现 `codegen/pool.go` -- 任务执行池:
   ```go
   type Pool struct {
       maxWorkers int
       taskQueue  chan *Task
       wg         sync.WaitGroup
   }
   ```
   - 控制并发数 (防止同时启动过多 Claude 进程)
   - 初期设置 max=3

2. 实现 `codegen/executor.go` -- 核心执行器:
   ```
   完整执行流程:
   a. 状态更新: pending → cloning
   b. Clone 仓库到 /tmp/codemaster/codegen/<task-id>/
   c. 从 source_branch 创建 feature 分支
   d. 状态更新: cloning → running
   e. 构造 prompt (调用 prompt.go)
   f. 启动 claude 子进程:
      claude -p "<prompt>" \
        --output-format stream-json \
        --allowedTools Read,Write,Edit,Glob,Grep,Bash \
        --max-turns 50
      工作目录 = 克隆的仓库目录
   g. 流式读取 stdout (逐行 JSON)
   h. 解析每行，广播到 SSE Hub + 缓存到 Redis
   i. 等待进程结束
   j. 执行 git diff，记录变更统计
   k. git push feature 分支到远端
   l. 状态更新: running → completed
   m. 清理临时目录 (可选延迟清理)
   ```

3. 实现 `codegen/prompt.go` -- Prompt 构造:
   ```
   组成部分:
   ① 系统角色设定 (资深工程师，遵循项目风格)
   ② 项目上下文 (仓库分析结果: 技术栈/目录结构/代码风格)
   ③ 需求描述 (标题 + 详细描述)
   ④ 飞书文档内容 (如有)
   ⑤ 额外上下文 (用户补充说明)
   ⑥ 编码约束 (先读后写、遵循风格、生成测试、编译检查)
   ```

4. 实现 `codegen/stream_parser.go`:
   ```go
   // Claude Code stream-json 事件类型:
   // - type=assistant, subtype=thinking  → AI 思考
   // - type=assistant, subtype=text      → AI 回复
   // - type=tool_use                     → 工具调用 (Read/Write/Edit/Bash...)
   // - type=tool_result                  → 工具返回
   // - type=result                       → 最终结果 (含 cost)
   ```

5. 实现 `sse/hub.go` -- SSE 广播中心:
   ```go
   // 核心接口:
   Subscribe(taskID) → (eventChan, unsubscribeFn)
   Broadcast(taskID, event)
   ReplayHistory(taskID) → []events  // 从 Redis 回放
   ```
   - 每个 task 的事件缓存在 Redis List: `codegen:stream:<task-id>`
   - 设置 TTL = 24h (任务完成后)
   - 支持断线重连: 客户端带 `Last-Event-ID` header

6. 实现 `handler/codegen.go`:
   - `POST /requirements/:id/generate` → 创建任务，推入队列
   - `GET /codegen/:id/stream` → SSE Handler
   - `GET /codegen/:id` → 任务详情
   - `GET /codegen/:id/diff` → 获取 diff
   - `POST /codegen/:id/cancel` → 发送 SIGTERM 取消

**前端任务:**

1. 实现 `hooks/useCodeGenStream.ts`:
   ```typescript
   // 核心 hook:
   // - 建立 EventSource 连接
   // - 分类处理 status/output/progress/error/done 事件
   // - 维护 events 列表、status、filesChanged 状态
   // - 支持断线自动重连
   ```

2. 代码生成主页面 (`CodeGen/index.tsx`):
   - 顶部: 需求标题 + 状态 + 取消按钮
   - 阶段进度条: 克隆 → 分析 → 生成 → 检查 → 完成
   - 左侧: 实时输出流 (StreamView)
   - 右侧: 变更文件列表 (FileList)

3. StreamView 组件:
   - thinking 事件 → 灰色折叠块，可展开
   - text 事件 → 正常文本展示
   - tool_use(Read) → 显示 "读取文件: xxx"
   - tool_use(Write) → 显示 "创建文件: xxx" + 代码高亮
   - tool_use(Edit) → 显示 "编辑文件: xxx" + diff 高亮
   - tool_use(Bash) → 显示命令和输出
   - 自动滚动到底部

4. DiffView 组件:
   - 生成完成后展示完整 diff
   - 使用 react-diff-viewer 或类似库
   - 支持文件级切换

**交付物:** 用户可以一键生成代码，实时看到 Claude Code 的思考和操作过程。

---

### 阶段五: 代码 Review 与合并

**目标:** AI 自动 Review + 人工 Review + 自动创建 MR。

**后端任务:**

1. 实现 `review/ai_reviewer.go`:
   - 代码生成完成后自动触发
   - 获取 feature 分支与 develop 的 diff
   - 构造 Review Prompt:
     ```
     你是资深代码审查专家。请 Review 以下代码变更，评估:
     1. 代码质量 (可读性、可维护性)
     2. 安全性 (注入、XSS 等)
     3. 错误处理 (是否妥善处理异常)
     4. 代码风格 (是否符合项目规范)
     5. 测试覆盖 (是否有足够测试)

     输出严格 JSON 格式...
     ```
   - 调用 `claude -p "<prompt>" --output-format json --allowedTools Read,Glob,Grep`
   - 工作目录设为仓库目录 (只读工具，安全)
   - 解析结果写入 `code_reviews`

2. 实现人工 Review 接口:
   - 获取 AI Review 结果 + diff
   - 提交审查意见 (approved / rejected / needs_revision)

3. 实现 `gitops/merge_request.go`:
   - GitLab: 调用 `POST /api/v4/projects/:id/merge_requests`
   - GitHub: 调用 `POST /repos/:owner/:repo/pulls`
   - MR 描述模板:
     ```markdown
     ## 需求
     [需求标题](需求链接)

     ## 变更说明
     (由 AI 生成的代码变更摘要)

     ## AI Review
     - 评分: 85/100
     - 安全检查: 通过
     - 问题: 2 个 warning

     ## 人工 Review
     - 审查人: 王五
     - 意见: 整体通过，注意 xxx

     ---
     *由 CodeMaster 自动生成*
     ```
   - 人工审批通过后自动创建 MR

**前端任务:**

1. AI Review 结果页:
   - 评分展示 (分数 + 颜色)
   - 问题列表 (按 severity 排序)
   - 每个 issue 展示文件、行号、代码片段、问题描述、建议
   - 分类评估 (安全/错误处理/风格/测试)

2. 人工 Review 页面:
   - 左侧: Diff 查看 (含 AI 标注的问题位置)
   - 右侧: 审查表单 (意见 + 通过/拒绝)
   - 通过后显示 "创建合并请求" 按钮

3. MR 状态卡片:
   - MR 链接
   - 合并状态 (created → merged)

**交付物:** 代码生成后自动 AI Review，人工通过后一键创建 MR 合并到 develop。

---

### 阶段六: 完善与优化

**目标:** 打磨体验，提升稳定性。

1. **Dashboard 首页:**
   - 我的项目 / 我的需求 / 待 Review
   - 最近的代码生成任务
   - 统计数据 (生成次数、通过率等)

2. **操作日志:**
   - 记录关键操作
   - Admin 可查看全部日志

3. **通知集成:**
   - 代码生成完成 → 飞书通知 RD
   - AI Review 完成 → 飞书通知 RD
   - 人工 Review 通过/拒绝 → 飞书通知

4. **稳定性:**
   - Claude Code 进程超时处理 (默认 10 分钟)
   - 失败自动重试 (最多 1 次)
   - 临时文件定期清理
   - 任务执行池限流

5. **安全:**
   - access_token AES 加密存储
   - API 限流
   - 操作审计

---

## 四、代码生成引擎详细实现

这是整个系统的核心，单独展开说明。

### 4.1 整体架构

```
                  HTTP Request (触发生成)
                         │
                         ▼
                 ┌───────────────┐
                 │  codegen.go   │  Handler: 参数校验，创建任务记录
                 │  (handler)    │
                 └───────┬───────┘
                         │ 推入队列
                         ▼
                 ┌───────────────┐
                 │   pool.go     │  任务池: 控制并发，取出待执行任务
                 │  (worker pool)│  max_workers = 3
                 └───────┬───────┘
                         │ 分配 worker
                         ▼
                 ┌───────────────┐
                 │ executor.go   │  执行器: 完整生命周期管理
                 └───┬───┬───┬──┘
                     │   │   │
          ┌──────────┘   │   └──────────┐
          ▼              ▼              ▼
   ┌────────────┐ ┌────────────┐ ┌────────────┐
   │ gitops/    │ │ prompt.go  │ │ stream_    │
   │ clone.go   │ │ 构造Prompt │ │ parser.go  │
   │ branch.go  │ │            │ │ 解析输出   │
   │ diff.go    │ │            │ │            │
   └────────────┘ └────────────┘ └─────┬──────┘
                                       │ 广播事件
                                       ▼
                                ┌────────────┐
                                │  sse/hub   │ → SSE → 前端
                                │  + Redis   │ → 持久化日志
                                └────────────┘
```

### 4.2 Executor 完整流程伪代码

```go
func (e *Executor) Run(ctx context.Context) error {
    // ── Phase 1: 准备 ─────────────────────────────────
    e.updateStatus("cloning")
    e.broadcast(StatusEvent{Status: "cloning", Message: "正在克隆仓库..."})

    workDir := filepath.Join(os.TempDir(), "codemaster", "codegen", strconv.FormatInt(e.taskID, 10))
    if err := e.gitClone(ctx, workDir); err != nil {
        return e.fail("clone 失败: " + err.Error())
    }

    branch := fmt.Sprintf("feature/req-%d", e.requirementID)
    if err := e.gitCheckoutNewBranch(ctx, workDir, branch); err != nil {
        return e.fail("创建分支失败: " + err.Error())
    }

    // ── Phase 2: 构造 Prompt ──────────────────────────
    prompt := e.promptBuilder.Build(PromptInput{
        RepoAnalysis: e.repo.AnalysisResult,
        Requirement:  e.requirement,
        ExtraContext: e.extraContext,
    })

    // 保存 prompt 到数据库 (方便调试)
    e.savePrompt(prompt)

    // ── Phase 3: 执行 Claude Code ─────────────────────
    e.updateStatus("running")
    e.broadcast(StatusEvent{Status: "running", Message: "Claude Code 已启动"})

    cmd := exec.CommandContext(ctx, "claude",
        "-p", prompt,
        "--output-format", "stream-json",
        "--allowedTools", "Read,Write,Edit,Glob,Grep,Bash",
        "--max-turns", "50",
    )
    cmd.Dir = workDir
    cmd.Env = append(os.Environ(),
        "CLAUDE_CODE_MAX_TIMEOUT=600000", // 10 分钟超时
    )

    stdout, _ := cmd.StdoutPipe()
    stderr, _ := cmd.StderrPipe()

    if err := cmd.Start(); err != nil {
        return e.fail("启动 Claude Code 失败: " + err.Error())
    }

    // 记录 PID (用于取消)
    e.savePID(cmd.Process.Pid)

    // ── Phase 4: 流式读取 ─────────────────────────────
    scanner := bufio.NewScanner(stdout)
    scanner.Buffer(make([]byte, 1024*1024), 1024*1024) // 1MB buffer

    var filesChanged []string

    for scanner.Scan() {
        line := scanner.Text()

        event, err := ParseStreamJSON(line)
        if err != nil {
            continue // 跳过无法解析的行
        }

        // 追踪文件变更
        if event.Type == "tool_use" && (event.ToolName == "Write" || event.ToolName == "Edit") {
            filesChanged = appendUnique(filesChanged, event.FilePath())
        }

        // 广播: SSE → 前端
        e.sseHub.Broadcast(e.taskID, event)

        // 缓存: Redis (断线重连用)
        e.redis.RPush(ctx, fmt.Sprintf("codegen:stream:%d", e.taskID), line)

        // 进度事件
        if event.Type == "tool_use" {
            e.broadcast(ProgressEvent{
                FilesChanged:  len(filesChanged),
                CurrentAction: event.Summary(),
            })
        }
    }

    // 读取 stderr (错误信息)
    stderrBytes, _ := io.ReadAll(stderr)

    if err := cmd.Wait(); err != nil {
        errMsg := string(stderrBytes)
        return e.fail("Claude Code 执行失败: " + errMsg)
    }

    // ── Phase 5: 收尾 ─────────────────────────────────
    // 获取 diff 统计
    diffStat, _ := e.gitDiffStat(ctx, workDir, "develop", branch)

    // Push feature 分支到远端
    if err := e.gitPush(ctx, workDir, branch); err != nil {
        return e.fail("push 失败: " + err.Error())
    }

    // 更新任务记录
    e.updateCompleted(diffStat)

    // 更新需求状态
    e.updateRequirementStatus("generated")

    // 广播完成事件
    e.broadcast(StatusEvent{
        Status:       "completed",
        FilesChanged: diffStat.FilesChanged,
        Additions:    diffStat.Additions,
        Deletions:    diffStat.Deletions,
    })

    // 设置 Redis 缓存 TTL
    e.redis.Expire(ctx, fmt.Sprintf("codegen:stream:%d", e.taskID), 24*time.Hour)

    // ── Phase 6: 自动触发 AI Review ───────────────────
    go e.reviewService.TriggerAIReview(context.Background(), e.taskID)

    return nil
}
```

### 4.3 任务取消机制

```go
func (e *Executor) Cancel() error {
    pid := e.getPID()
    if pid == 0 {
        return errors.New("task not running")
    }

    process, err := os.FindProcess(pid)
    if err != nil {
        return err
    }

    // 先发 SIGTERM 优雅退出
    process.Signal(syscall.SIGTERM)

    // 3 秒后如果还在运行，发 SIGKILL
    time.AfterFunc(3*time.Second, func() {
        process.Signal(syscall.SIGKILL)
    })

    e.updateStatus("cancelled")
    e.broadcast(StatusEvent{Status: "cancelled"})

    return nil
}
```

### 4.4 SSE 断线重连

```go
// handler/codegen.go - SSE Handler
func (h *Handler) StreamCodeGen(c *gin.Context) {
    taskID := parseID(c.Param("id"))

    // 设置 SSE headers
    c.Writer.Header().Set("Content-Type", "text/event-stream")
    c.Writer.Header().Set("Cache-Control", "no-cache")
    c.Writer.Header().Set("Connection", "keep-alive")
    c.Writer.Header().Set("X-Accel-Buffering", "no") // nginx 不缓冲
    flusher := c.Writer.(http.Flusher)

    // 1. 回放历史事件 (从 Redis)
    lastEventID := c.GetHeader("Last-Event-ID")
    startIndex := int64(0)
    if lastEventID != "" {
        startIndex, _ = strconv.ParseInt(lastEventID, 10, 64)
        startIndex++ // 从下一条开始
    }

    history := h.redis.LRange(c, fmt.Sprintf("codegen:stream:%d", taskID), startIndex, -1).Val()
    eventID := startIndex
    for _, line := range history {
        fmt.Fprintf(c.Writer, "id: %d\nevent: output\ndata: %s\n\n", eventID, line)
        eventID++
        flusher.Flush()
    }

    // 2. 检查任务是否已完成
    task := h.service.GetTask(taskID)
    if task.Status == "completed" || task.Status == "failed" || task.Status == "cancelled" {
        fmt.Fprintf(c.Writer, "event: done\ndata: {\"status\":\"%s\"}\n\n", task.Status)
        flusher.Flush()
        return
    }

    // 3. 订阅实时事件
    ch, unsub := h.sseHub.Subscribe(taskID)
    defer unsub()

    for {
        select {
        case event := <-ch:
            data, _ := json.Marshal(event)
            fmt.Fprintf(c.Writer, "id: %d\nevent: %s\ndata: %s\n\n", eventID, event.Type, data)
            eventID++
            flusher.Flush()

            if event.Type == "done" {
                return
            }
        case <-c.Request.Context().Done():
            return // 客户端断开
        }
    }
}
```

---

## 五、配置文件模板

```yaml
# backend/config.yaml

server:
  port: 8080
  mode: debug  # debug / release

database:
  host: 127.0.0.1
  port: 3306
  user: root
  password: ""
  dbname: codemaster
  charset: utf8mb4

redis:
  addr: 127.0.0.1:6379
  password: ""
  db: 0

feishu:
  app_id: "cli_xxxxx"
  app_secret: "xxxxx"
  redirect_uri: "http://localhost:8080/api/v1/auth/feishu/callback"

jwt:
  secret: "your-jwt-secret-key"
  expire_hours: 72

codegen:
  max_workers: 3             # 最大并发生成任务数
  max_turns: 50              # Claude Code 最大交互轮次
  timeout_minutes: 10        # 单个任务超时时间
  work_dir: "/tmp/codemaster" # 临时工作目录

encrypt:
  aes_key: "32-byte-aes-key-for-token-encrypt"
```

---

## 六、Docker Compose

```yaml
# docker-compose.yml

version: "3.8"

services:
  mysql:
    image: mysql:8.0
    environment:
      MYSQL_ROOT_PASSWORD: root
      MYSQL_DATABASE: codemaster
    ports:
      - "3306:3306"
    volumes:
      - mysql_data:/var/lib/mysql

  redis:
    image: redis:7-alpine
    ports:
      - "6379:6379"

  backend:
    build:
      context: ./backend
      dockerfile: Dockerfile
    ports:
      - "8080:8080"
    depends_on:
      - mysql
      - redis
    volumes:
      - /tmp/codemaster:/tmp/codemaster  # 共享临时目录
    environment:
      - CONFIG_PATH=/app/config.yaml

  frontend:
    build:
      context: ./frontend
      dockerfile: Dockerfile
    ports:
      - "3000:80"
    depends_on:
      - backend

volumes:
  mysql_data:
```

---

## 七、关键风险与应对

| 风险 | 影响 | 应对策略 |
|------|------|----------|
| Claude Code 生成代码质量不稳定 | 生成的代码可能无法编译或逻辑错误 | Prompt 中要求编译检查；AI Review 层兜底；人工 Review 必须通过 |
| 单次生成耗时过长 | 用户等待时间长，资源占用 | 设置 max_turns=50 和 10 分钟超时；进度提示管理预期 |
| 并发生成任务过多 | 服务器资源不足，Claude API 限流 | worker pool 限制并发数=3；任务队列排队 |
| 仓库 clone 时间长 | 大仓库首次 clone 慢 | shallow clone (--depth 1)；可考虑仓库镜像缓存 |
| SSE 连接断开 | 用户丢失输出内容 | Redis 缓存全量事件；Last-Event-ID 断线重连 |
| access_token 安全 | 仓库 token 泄露风险 | AES 加密存储；不在日志中打印；API 不返回明文 |
