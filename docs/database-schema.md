# CodeMaster 数据库设计文档

> 数据库: MySQL 8.0+
> ORM: GORM
> 字符集: utf8mb4
> 时区: UTC

---

## 1. 用户表 (users)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| feishu_uid | VARCHAR(128) | UNIQUE, NOT NULL | 飞书用户唯一标识 |
| feishu_union_id | VARCHAR(128) | UNIQUE | 飞书 union_id (跨应用) |
| name | VARCHAR(64) | NOT NULL | 用户姓名 |
| avatar | VARCHAR(512) | | 头像 URL |
| email | VARCHAR(128) | | 邮箱 |
| role | ENUM('pm','rd','admin') | NOT NULL, DEFAULT 'rd' | 用户角色 |
| status | TINYINT | DEFAULT 1 | 1=正常 0=禁用 |
| last_login_at | TIMESTAMP | NULL | 最后登录时间 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 创建时间 |
| updated_at | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | 更新时间 |

**索引:**
- `idx_feishu_uid` (feishu_uid) UNIQUE
- `idx_role` (role)

---

## 2. 项目表 (projects)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| name | VARCHAR(128) | NOT NULL | 项目名称 |
| description | TEXT | | 项目描述 |
| owner_id | BIGINT | FK -> users.id, NOT NULL | 创建者 (PM) |
| doc_links | JSON | | 关联飞书文档链接列表 |
| status | ENUM('active','archived') | DEFAULT 'active' | 项目状态 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | |
| updated_at | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | |

**索引:**
- `idx_owner_id` (owner_id)
- `idx_status` (status)

**doc_links JSON 结构:**
```json
[
  {
    "title": "PRD - 用户中台",
    "url": "https://xxx.feishu.cn/docs/xxx",
    "type": "prd"
  }
]
```

---

## 3. 项目成员表 (project_members)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| project_id | BIGINT | FK -> projects.id, NOT NULL | 项目 ID |
| user_id | BIGINT | FK -> users.id, NOT NULL | 用户 ID |
| role | ENUM('pm','rd') | NOT NULL | 在项目中的角色 |
| joined_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | 加入时间 |

**索引:**
- `uk_project_user` (project_id, user_id) UNIQUE
- `idx_user_id` (user_id)

---

## 4. 代码仓库表 (repositories)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| project_id | BIGINT | FK -> projects.id, NOT NULL | 所属项目 |
| name | VARCHAR(128) | NOT NULL | 仓库显示名称 |
| git_url | VARCHAR(512) | NOT NULL | Git clone 地址 |
| platform | ENUM('gitlab','github') | NOT NULL | 代码托管平台 |
| platform_project_id | VARCHAR(64) | | 平台侧项目 ID (用于 API 调用) |
| default_branch | VARCHAR(64) | DEFAULT 'develop' | 默认分支 |
| access_token | VARCHAR(512) | | 加密存储的 access token |
| analysis_result | JSON | | 仓库功能分析结果 |
| analysis_status | ENUM('pending','running','completed','failed') | DEFAULT 'pending' | 分析状态 |
| analyzed_at | TIMESTAMP | NULL | 最后分析时间 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | |
| updated_at | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | |

**索引:**
- `idx_project_id` (project_id)

**analysis_result JSON 结构:**
```json
{
  "modules": [
    {
      "path": "internal/handler",
      "description": "HTTP 接口层，基于 Gin 框架",
      "files_count": 12
    },
    {
      "path": "internal/service",
      "description": "业务逻辑层",
      "files_count": 8
    }
  ],
  "tech_stack": ["Go 1.21", "Gin", "GORM", "MySQL", "Redis"],
  "entry_points": ["cmd/server/main.go"],
  "directory_structure": "标准 Go 项目布局，cmd/internal/pkg 三层",
  "code_style": {
    "naming": "camelCase",
    "error_handling": "errors.Wrap",
    "test_framework": "testify"
  }
}
```

---

## 5. 需求表 (requirements)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| project_id | BIGINT | FK -> projects.id, NOT NULL | 所属项目 |
| title | VARCHAR(256) | NOT NULL | 需求标题 |
| description | TEXT | NOT NULL | 需求详细描述 |
| doc_links | JSON | | 关联飞书文档链接 |
| doc_content | LONGTEXT | | 飞书文档抓取的纯文本内容 |
| priority | ENUM('p0','p1','p2','p3') | DEFAULT 'p1' | 优先级 |
| status | ENUM('draft','generating','generated','reviewing','approved','merged','rejected') | DEFAULT 'draft' | 需求状态 |
| creator_id | BIGINT | FK -> users.id, NOT NULL | 创建者 (PM) |
| assignee_id | BIGINT | FK -> users.id | 指派的 RD |
| repository_id | BIGINT | FK -> repositories.id | 目标代码仓库 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | |
| updated_at | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | |

**索引:**
- `idx_project_id` (project_id)
- `idx_status` (status)
- `idx_creator_id` (creator_id)
- `idx_assignee_id` (assignee_id)

**状态流转:**
```
draft -> generating -> generated -> reviewing -> approved -> merged
                   \-> failed                  \-> rejected -> draft (可重新编辑)
```

---

## 6. 代码生成任务表 (codegen_tasks)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| requirement_id | BIGINT | FK -> requirements.id, NOT NULL | 关联需求 |
| repository_id | BIGINT | FK -> repositories.id, NOT NULL | 目标仓库 |
| source_branch | VARCHAR(64) | NOT NULL | 基于哪个分支 (通常 develop) |
| target_branch | VARCHAR(128) | NOT NULL | 生成代码的特性分支 |
| status | ENUM('pending','cloning','running','completed','failed','cancelled') | DEFAULT 'pending' | 任务状态 |
| prompt | TEXT | | 发送给 Claude Code 的完整 prompt |
| output_log | LONGTEXT | | 完整的流式输出日志 |
| diff_stat | JSON | | 代码变更统计 |
| error_message | TEXT | | 失败时的错误信息 |
| claude_cost_usd | DECIMAL(10,4) | | Claude API 消耗费用 |
| started_at | TIMESTAMP | NULL | 开始执行时间 |
| completed_at | TIMESTAMP | NULL | 执行完成时间 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | |

**索引:**
- `idx_requirement_id` (requirement_id)
- `idx_status` (status)

**diff_stat JSON 结构:**
```json
{
  "files_changed": 5,
  "additions": 230,
  "deletions": 12,
  "files": [
    {
      "path": "internal/handler/register.go",
      "status": "added",
      "additions": 85,
      "deletions": 0
    },
    {
      "path": "internal/router/router.go",
      "status": "modified",
      "additions": 3,
      "deletions": 0
    }
  ]
}
```

---

## 7. 代码审查表 (code_reviews)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| codegen_task_id | BIGINT | FK -> codegen_tasks.id, NOT NULL | 关联生成任务 |
| ai_review_result | JSON | | AI Review 详细结果 |
| ai_score | INT | | AI 评分 0-100 |
| ai_status | ENUM('pending','running','passed','warning','failed') | DEFAULT 'pending' | AI 审查状态 |
| human_reviewer_id | BIGINT | FK -> users.id | 人工审查者 |
| human_comment | TEXT | | 人工审查意见 |
| human_status | ENUM('pending','approved','rejected','needs_revision') | DEFAULT 'pending' | 人工审查状态 |
| merge_request_id | VARCHAR(64) | | 平台侧 MR/PR ID |
| merge_request_url | VARCHAR(512) | | MR/PR 链接 |
| merge_status | ENUM('none','created','merged','closed') | DEFAULT 'none' | 合并状态 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | |
| updated_at | TIMESTAMP | ON UPDATE CURRENT_TIMESTAMP | |

**索引:**
- `idx_codegen_task_id` (codegen_task_id)
- `idx_human_reviewer_id` (human_reviewer_id)

**ai_review_result JSON 结构:**
```json
{
  "summary": "代码整体质量良好，结构清晰，有几处建议需要关注",
  "issues": [
    {
      "severity": "error",
      "file": "internal/handler/register.go",
      "line": 45,
      "code_snippet": "phone := c.PostForm(\"phone\")",
      "message": "缺少手机号格式校验",
      "suggestion": "建议使用正则校验手机号格式，或使用 binding:\"required,len=11\" tag"
    },
    {
      "severity": "warning",
      "file": "internal/service/register.go",
      "line": 23,
      "code_snippet": "db.Create(&user)",
      "message": "未处理数据库唯一约束冲突",
      "suggestion": "应捕获 duplicate key error 并返回友好提示"
    }
  ],
  "categories": {
    "security": { "status": "passed", "details": "未发现 SQL 注入、XSS 等安全问题" },
    "error_handling": { "status": "warning", "details": "部分错误未妥善处理" },
    "code_style": { "status": "passed", "details": "符合项目现有代码风格" },
    "test_coverage": { "status": "warning", "details": "建议补充注册接口的单元测试" }
  }
}
```

---

## 8. 操作日志表 (operation_logs)

| 字段 | 类型 | 约束 | 说明 |
|------|------|------|------|
| id | BIGINT | PK, AUTO_INCREMENT | 主键 |
| user_id | BIGINT | FK -> users.id | 操作用户 |
| action | VARCHAR(64) | NOT NULL | 操作类型 |
| resource_type | VARCHAR(32) | NOT NULL | 资源类型 (project/requirement/codegen/review) |
| resource_id | BIGINT | | 资源 ID |
| detail | JSON | | 操作详情 |
| ip | VARCHAR(45) | | IP 地址 |
| created_at | TIMESTAMP | DEFAULT CURRENT_TIMESTAMP | |

**索引:**
- `idx_user_id` (user_id)
- `idx_resource` (resource_type, resource_id)
- `idx_created_at` (created_at)

---

## ER 关系图

```
users 1──N project_members N──1 projects
  │                                │
  │(creator_id)                    │
  │(assignee_id)                   │
  │                                │
  ├──N requirements N──1───────────┘
  │       │                        │
  │       │                    1───┤
  │       │                        │
  │   1───┤                  repositories
  │       │                        │
  │       N                        │
  │  codegen_tasks ────────────────┘
  │       │
  │       1
  │       │
  │       N
  │  code_reviews
  │       │
  └───────┘ (human_reviewer_id)
```
