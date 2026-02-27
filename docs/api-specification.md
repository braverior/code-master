# CodeMaster API 接口文档

> Base URL: `/api/v1`
> 认证方式: Bearer Token (JWT)
> 内容类型: application/json
> 时间格式: ISO 8601 (UTC)

---

## 通用说明

### 统一响应格式

```json
// 成功
{
  "code": 0,
  "message": "success",
  "data": { ... }
}

// 分页
{
  "code": 0,
  "message": "success",
  "data": {
    "list": [...],
    "total": 100,
    "page": 1,
    "page_size": 20
  }
}

// 错误
{
  "code": 40001,
  "message": "参数校验失败: title 不能为空",
  "data": null
}
```

### 通用分页参数

所有列表接口均支持以下分页与排序参数:

| 参数 | 类型 | 必填 | 默认值 | 说明 |
|------|------|------|--------|------|
| page | int | 否 | 1 | 页码，从 1 开始 |
| page_size | int | 否 | 20 | 每页数量，最大 100 |
| sort_by | string | 否 | created_at | 排序字段，每个接口可用字段不同 |
| order | string | 否 | desc | 排序方向: `asc` / `desc` |

### 错误码规范

| 错误码 | 说明 | 典型场景 |
|--------|------|----------|
| 0 | 成功 | |
| 40001 | 参数校验失败 | 必填字段缺失、格式不合法 |
| 40002 | 参数值无效 | 枚举值不在范围内 |
| 40003 | 状态不允许操作 | 需求非 draft 状态下编辑 |
| 40004 | 前置条件不满足 | 触发生成但未关联仓库 |
| 40005 | 资源冲突 | 已存在同名项目、重复添加成员 |
| 40006 | 操作频率限制 | 短时间内重复触发生成 |
| 40101 | Token 缺失 | 未携带 Authorization header |
| 40102 | Token 过期 | JWT 已过期 |
| 40103 | Token 无效 | JWT 签名验证失败 |
| 40104 | 用户已禁用 | 账号被 admin 禁用 |
| 40301 | 角色权限不足 | RD 尝试创建项目 |
| 40302 | 非项目成员 | 访问未加入的项目 |
| 40303 | 非资源所有者 | 非 owner 尝试编辑项目 |
| 40401 | 用户不存在 | |
| 40402 | 项目不存在 | |
| 40403 | 仓库不存在 | |
| 40404 | 需求不存在 | |
| 40405 | 生成任务不存在 | |
| 40406 | Review 记录不存在 | |
| 50001 | 服务端内部错误 | 未预期的 panic |
| 50002 | 数据库错误 | DB 连接失败 |
| 50101 | Git 操作失败 | clone/push 失败 |
| 50102 | Git Token 无效 | 仓库 access_token 鉴权失败 |
| 50103 | Claude Code 执行失败 | 子进程崩溃 |
| 50104 | Claude Code 超时 | 生成任务超过时间限制 |
| 50105 | 飞书 API 错误 | OAuth/文档接口调用失败 |

### 认证 Header

```
Authorization: Bearer <jwt-token>
```

除 `/auth/feishu/login` 和 `/auth/feishu/callback` 外，所有接口均需携带。

---

## 1. 认证模块 (Auth)

### 1.1 飞书 OAuth 登录

**GET** `/auth/feishu/login`

重定向到飞书 OAuth2 授权页面。前端直接 `window.location.href` 跳转。

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| redirect_uri | string | 否 | 登录成功后的前端回调地址，默认 `/` |

**响应:** 302 重定向到飞书授权页

---

### 1.2 飞书 OAuth 回调

**GET** `/auth/feishu/callback`

飞书授权完成后回调此接口，后端用 code 换取用户信息，签发 JWT。

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| code | string | 是 | 飞书授权码 |
| state | string | 是 | CSRF state |

**响应:** 302 重定向到前端页面，URL 中携带 token

```
302 -> {redirect_uri}?token=<jwt>&is_new_user=true
```

`is_new_user=true` 表示首次登录，前端应引导用户选择角色。

**错误响应:**
```json
// 飞书授权码无效
{ "code": 50105, "message": "飞书授权失败: invalid code" }
```

---

### 1.3 获取当前用户信息

**GET** `/auth/me`

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "张三",
    "avatar": "https://avatar.feishu.cn/xxx",
    "email": "zhangsan@company.com",
    "role": "pm",
    "is_admin": true,
    "status": 1,
    "is_new_user": false,
    "last_login_at": "2026-02-12T08:00:00Z",
    "created_at": "2026-01-01T00:00:00Z"
  }
}
```

> **说明:** `role` 为业务角色 (`pm` / `rd`)，`is_admin` 为独立的管理员标记，两者可共存。例如一个用户可以同时是 PM 和管理员。

---

### 1.4 选择/修改角色

**PUT** `/auth/role`

首次登录时用户选择业务角色，admin 可修改他人角色。

**请求:**
```json
{
  "user_id": 2,
  "role": "pm"
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| user_id | int | 否 | >0 | admin 修改他人时传，不传则修改自己 |
| role | string | 是 | pm / rd | 业务角色 |

**权限规则:**
- 不传 `user_id`: 仅首次登录 (`is_new_user=true`) 允许
- 传 `user_id`: 需要 `is_admin=true`

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 2,
    "name": "李四",
    "role": "pm",
    "updated_at": "2026-02-12T10:00:00Z"
  }
}
```

**错误响应:**
```json
// 非首次登录尝试修改自己角色
{ "code": 40003, "message": "角色已选择，如需修改请联系管理员" }

// 非 admin 尝试修改他人
{ "code": 40301, "message": "权限不足，仅管理员可修改他人角色" }
```

---

### 1.5 刷新 Token

**POST** `/auth/refresh`

在 token 即将过期时刷新，返回新 token。要求当前 token 仍在有效期内。

**响应:**
```json
{
  "code": 0,
  "data": {
    "token": "new-jwt-token",
    "expire_at": "2026-02-15T10:00:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 40102, "message": "Token 已过期，请重新登录" }
```

---

## 2. 用户管理 (Admin)

> 所有接口要求: `is_admin=true`

### 2.1 用户列表

**GET** `/admin/users`

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20，最大 100 |
| keyword | string | 否 | 搜索关键词 (姓名/邮箱模糊匹配) |
| role | string | 否 | 按业务角色筛选: pm / rd |
| is_admin | string | 否 | 按管理员身份筛选: true / false |
| status | int | 否 | 按状态筛选: 1=正常 0=禁用 |
| sort_by | string | 否 | 排序字段: created_at / last_login_at / name，默认 created_at |
| order | string | 否 | asc / desc，默认 desc |

**响应:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "name": "张三",
        "avatar": "https://avatar.feishu.cn/xxx",
        "email": "zhangsan@company.com",
        "role": "rd",
        "is_admin": true,
        "status": 1,
        "last_login_at": "2026-02-12T08:00:00Z",
        "created_at": "2026-01-01T00:00:00Z"
      }
    ],
    "total": 50,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 2.2 修改用户角色

**PUT** `/admin/users/:id/role`

**请求:**
```json
{
  "role": "pm"
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| role | string | 是 | pm / rd | 业务角色 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 2,
    "name": "李四",
    "role": "pm",
    "is_admin": false,
    "updated_at": "2026-02-12T10:00:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 40401, "message": "用户不存在" }
```

---

### 2.3 设置/取消管理员

**PUT** `/admin/users/:id/admin`

**请求:**
```json
{
  "is_admin": true
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| is_admin | bool | 是 | true / false | 是否设为管理员 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 2,
    "name": "李四",
    "role": "rd",
    "is_admin": true,
    "updated_at": "2026-02-12T10:00:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 40401, "message": "用户不存在" }
```

---

### 2.4 禁用/启用用户

**PUT** `/admin/users/:id/status`

**请求:**
```json
{
  "status": 0
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| status | int | 是 | 0 或 1 | 0=禁用 1=启用 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 2,
    "name": "李四",
    "status": 0,
    "updated_at": "2026-02-12T10:00:00Z"
  }
}
```

**错误响应:**
```json
// 不能禁用自己
{ "code": 40003, "message": "不能禁用当前登录账号" }
```

---

### 2.5 操作日志查询

**GET** `/admin/operation-logs`

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |
| user_id | int | 否 | 按操作用户筛选 |
| action | string | 否 | 按操作类型: create_project / generate_code / review_approve / ... |
| resource_type | string | 否 | 资源类型: project / requirement / codegen / review |
| start_time | string | 否 | 起始时间 (ISO 8601) |
| end_time | string | 否 | 截止时间 (ISO 8601) |

**响应:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1001,
        "user": { "id": 1, "name": "张三" },
        "action": "generate_code",
        "resource_type": "codegen",
        "resource_id": 42,
        "detail": {
          "requirement_id": 15,
          "requirement_title": "新增用户注册功能",
          "repository": "user-service"
        },
        "ip": "10.0.1.100",
        "created_at": "2026-02-12T11:05:00Z"
      }
    ],
    "total": 500,
    "page": 1,
    "page_size": 20
  }
}
```

---

## 3. 用户搜索 (通用)

### 3.1 搜索用户

**GET** `/users/search`

用于添加项目成员时搜索用户。

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| keyword | string | 是 | 搜索关键词 (姓名/邮箱)，最少 1 个字符 |
| role | string | 否 | 按角色筛选 |
| exclude_project_id | int | 否 | 排除已在该项目中的成员 |
| limit | int | 否 | 返回数量，默认 10，最大 50 |

**响应:**
```json
{
  "code": 0,
  "data": [
    {
      "id": 3,
      "name": "王五",
      "avatar": "https://avatar.feishu.cn/xxx",
      "email": "wangwu@company.com",
      "role": "rd",
      "is_admin": false
    }
  ]
}
```

---

## 4. 项目管理 (Projects)

### 4.1 创建项目

**POST** `/projects`

**权限:** pm, admin

**请求:**
```json
{
  "name": "用户中台",
  "description": "用户中台微服务，包含注册、登录、权限管理等功能",
  "doc_links": [
    { "title": "PRD 文档", "url": "https://xxx.feishu.cn/docs/xxx", "type": "prd" },
    { "title": "技术方案", "url": "https://xxx.feishu.cn/docs/yyy", "type": "tech" }
  ],
  "member_ids": [2, 3, 4]
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| name | string | 是 | 1-128 字符 | 项目名称 |
| description | string | 否 | 最大 5000 字符 | 项目描述 |
| doc_links | array | 否 | 每项需含 title+url | 关联文档列表 |
| doc_links[].title | string | 是 | 1-128 字符 | 文档标题 |
| doc_links[].url | string | 是 | 合法 URL | 文档链接 |
| doc_links[].type | string | 否 | prd / tech / design / other | 文档类型，默认 other |
| member_ids | array[int] | 否 | 有效用户 ID | 初始成员 (自动加为 rd) |

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "用户中台",
    "description": "用户中台微服务，包含注册、登录、权限管理等功能",
    "doc_links": [
      { "title": "PRD 文档", "url": "https://xxx.feishu.cn/docs/xxx", "type": "prd" }
    ],
    "owner": { "id": 1, "name": "张三", "avatar": "..." },
    "members": [
      { "id": 2, "name": "李四", "role": "rd", "avatar": "..." },
      { "id": 3, "name": "王五", "role": "rd", "avatar": "..." }
    ],
    "status": "active",
    "created_at": "2026-02-12T10:00:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 40001, "message": "参数校验失败: name 不能为空" }
{ "code": 40005, "message": "项目名称已存在" }
{ "code": 40301, "message": "权限不足，仅 PM 和管理员可创建项目" }
```

---

### 4.2 项目列表

**GET** `/projects`

返回当前用户有权限查看的项目 (作为 owner 或 member 的项目，admin 看全部)。

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |
| keyword | string | 否 | 按名称模糊搜索 |
| status | string | 否 | active / archived |
| owner_id | int | 否 | 按创建者筛选 |
| sort_by | string | 否 | created_at / updated_at / name，默认 updated_at |
| order | string | 否 | asc / desc，默认 desc |

**响应:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "name": "用户中台",
        "description": "用户中台微服务...",
        "owner": { "id": 1, "name": "张三", "avatar": "..." },
        "member_count": 3,
        "repo_count": 2,
        "requirement_count": 8,
        "open_requirement_count": 3,
        "status": "active",
        "created_at": "2026-02-12T10:00:00Z",
        "updated_at": "2026-02-12T15:00:00Z"
      }
    ],
    "total": 5,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 4.3 项目详情

**GET** `/projects/:id`

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "用户中台",
    "description": "用户中台微服务...",
    "doc_links": [
      { "title": "PRD 文档", "url": "https://xxx.feishu.cn/docs/xxx", "type": "prd" }
    ],
    "owner": { "id": 1, "name": "张三", "avatar": "..." },
    "members": [
      { "id": 1, "name": "张三", "role": "pm", "avatar": "...", "joined_at": "2026-02-12T10:00:00Z" },
      { "id": 2, "name": "李四", "role": "rd", "avatar": "...", "joined_at": "2026-02-12T10:00:00Z" }
    ],
    "repositories": [
      {
        "id": 1,
        "name": "user-service",
        "git_url": "https://gitlab.com/company/user-service.git",
        "platform": "gitlab",
        "default_branch": "develop",
        "analysis_status": "completed",
        "analyzed_at": "2026-02-12T10:05:00Z"
      }
    ],
    "stats": {
      "total_requirements": 8,
      "draft": 2,
      "generating": 0,
      "generated": 1,
      "reviewing": 2,
      "approved": 1,
      "merged": 2
    },
    "status": "active",
    "created_at": "2026-02-12T10:00:00Z",
    "updated_at": "2026-02-12T15:00:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 40402, "message": "项目不存在" }
{ "code": 40302, "message": "非项目成员，无权查看" }
```

---

### 4.4 更新项目

**PUT** `/projects/:id`

**权限:** 项目 owner 或 admin

**请求:**
```json
{
  "name": "用户中台 v2",
  "description": "更新后的描述",
  "doc_links": [
    { "title": "PRD 文档 v2", "url": "https://xxx.feishu.cn/docs/xxx", "type": "prd" }
  ]
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| name | string | 否 | 1-128 字符 | 项目名称 |
| description | string | 否 | 最大 5000 字符 | 项目描述 |
| doc_links | array | 否 | | 关联文档 (全量替换) |

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "用户中台 v2",
    "description": "更新后的描述",
    "doc_links": [...],
    "updated_at": "2026-02-12T16:00:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 40303, "message": "非项目所有者，无权编辑" }
{ "code": 40005, "message": "项目名称已存在" }
```

---

### 4.5 归档项目

**PUT** `/projects/:id/archive`

**权限:** 项目 owner 或 admin

归档后项目变为只读，不可创建需求、触发生成。

**响应:**
```json
{
  "code": 0,
  "data": { "id": 1, "status": "archived", "updated_at": "2026-02-12T16:00:00Z" }
}
```

**错误响应:**
```json
// 有正在运行的生成任务
{ "code": 40003, "message": "项目存在进行中的代码生成任务，无法归档" }
```

---

### 4.6 添加项目成员

**POST** `/projects/:id/members`

**权限:** 项目 owner 或 admin

**请求:**
```json
{
  "user_ids": [5, 6],
  "role": "rd"
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| user_ids | array[int] | 是 | 非空，每项为有效用户 ID | 要添加的用户 |
| role | string | 是 | pm / rd | 成员角色 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "added": [
      { "id": 5, "name": "赵六", "role": "rd" },
      { "id": 6, "name": "钱七", "role": "rd" }
    ],
    "skipped": []
  }
}
```

`skipped` 返回已存在的成员 ID (去重而非报错)。

**错误响应:**
```json
{ "code": 40401, "message": "用户不存在: id=99" }
{ "code": 40303, "message": "非项目所有者，无权添加成员" }
```

---

### 4.7 移除项目成员

**DELETE** `/projects/:id/members/:user_id`

**权限:** 项目 owner 或 admin。不可移除 owner 自己。

**响应:**
```json
{
  "code": 0,
  "data": { "message": "成员已移除" }
}
```

**错误响应:**
```json
{ "code": 40003, "message": "不能移除项目所有者" }
{ "code": 40401, "message": "该用户不是项目成员" }
```

---

## 5. 代码仓库管理 (Repositories)

### 5.1 关联代码仓库

**POST** `/projects/:id/repos`

**权限:** 项目 RD 成员, admin

**请求:**
```json
{
  "name": "user-service",
  "git_url": "https://gitlab.com/company/user-service.git",
  "platform": "gitlab",
  "platform_project_id": "12345",
  "default_branch": "develop"
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| name | string | 是 | 1-128 字符 | 仓库显示名称 |
| git_url | string | 是 | 合法 git URL (https://) | Git clone 地址 |
| platform | string | 是 | gitlab / github | 代码托管平台 |
| platform_project_id | string | 条件必填 | | 平台侧项目 ID (GitLab 必填，用于 API 调用) |
| default_branch | string | 否 | 合法分支名 | 默认分支，默认 develop |

**后端行为:**
1. 创建仓库记录
2. Git Token 从用户的个人设置 (`settings/llm` 中的 `gitlab_token`) 获取，或使用仓库存储的 token
3. 创建成功后返回 (不自动触发分析)

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "user-service",
    "git_url": "https://gitlab.com/company/user-service.git",
    "platform": "gitlab",
    "default_branch": "develop",
    "analysis_status": "pending",
    "created_at": "2026-02-12T10:00:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 50102, "message": "仓库连接失败: access token 无效或无权限" }
{ "code": 50103, "message": "Token 无推送权限: 请确保 Token 拥有 write_repository 权限" }
{ "code": 40005, "message": "该仓库已关联到此项目" }
{ "code": 40001, "message": "GitLab 平台需要提供 platform_project_id" }
```

---

### 5.2 项目仓库列表

**GET** `/projects/:id/repos`

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码，默认 1 |
| page_size | int | 否 | 每页数量，默认 20 |
| analysis_status | string | 否 | 按分析状态筛选 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 1,
        "name": "user-service",
        "git_url": "https://gitlab.com/company/user-service.git",
        "platform": "gitlab",
        "default_branch": "develop",
        "analysis_status": "completed",
        "analysis_result": {
          "modules": [
            { "path": "src/controllers", "description": "HTTP 请求处理层", "files_count": 8 }
          ],
          "tech_stack": ["TypeScript", "Express", "PostgreSQL"],
          "entry_points": ["src/index.ts"],
          "directory_structure": "src/\n  controllers/\n  services/\n  models/\n  routes/",
          "code_style": {
            "naming": "camelCase",
            "error_handling": "try-catch + custom error classes",
            "test_framework": "Jest"
          }
        },
        "analyzed_at": "2026-02-12T10:05:00Z",
        "created_at": "2026-02-12T10:00:00Z"
      }
    ],
    "total": 2,
    "page": 1,
    "page_size": 20
  }
}
```

当 `analysis_status` 为 `failed` 时，响应中会包含 `analysis_error` 字段:

```json
{
  "id": 2,
  "name": "payment-service",
  "git_url": "https://gitlab.com/company/payment-service.git",
  "platform": "gitlab",
  "default_branch": "develop",
  "analysis_status": "failed",
  "analysis_error": "克隆仓库失败: authentication required",
  "analysis_result": null,
  "analyzed_at": null,
  "created_at": "2026-02-12T10:00:00Z"
}
```

> `analysis_result` 在 `analysis_status` 为 `completed` 时包含完整分析数据，其他状态为 `null`。列表和详情接口均返回此字段。

---

### 5.3 仓库详情

**GET** `/repos/:id`

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "user-service",
    "git_url": "https://gitlab.com/company/user-service.git",
    "platform": "gitlab",
    "platform_project_id": "12345",
    "default_branch": "develop",
    "analysis_status": "completed",
    "analysis_result": {
      "modules": [...],
      "tech_stack": [...],
      "entry_points": [...],
      "directory_structure": "...",
      "code_style": { ... }
    },
    "analyzed_at": "2026-02-12T10:05:00Z",
    "project": { "id": 1, "name": "用户中台" },
    "created_at": "2026-02-12T10:00:00Z"
  }
}
```

注意: `access_token` 不会在任何接口中返回明文。

---

### 5.4 更新仓库信息

**PUT** `/repos/:id`

**权限:** 项目 RD 成员, admin

**请求:**
```json
{
  "name": "user-service-v2",
  "default_branch": "main"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| name | string | 否 | 显示名称 |
| default_branch | string | 否 | 默认分支 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "name": "user-service-v2",
    "default_branch": "main",
    "updated_at": "2026-02-12T10:10:00Z"
  }
}
```

---

### 5.5 解除仓库关联

**DELETE** `/repos/:id`

**权限:** 项目 owner, admin

**前置条件:** 该仓库没有正在进行的代码生成任务。

**响应:**
```json
{
  "code": 0,
  "data": { "message": "仓库关联已解除" }
}
```

**错误响应:**
```json
{ "code": 40003, "message": "该仓库存在进行中的生成任务，无法解除关联" }
```

---

### 5.6 测试仓库连通性

**POST** `/repos/:id/test-connection`

**权限:** 项目 RD 成员, admin

不修改数据，验证当前存储的 access_token 读取与推送权限。

**后端行为:**
1. 执行 `git ls-remote` 验证读取权限，获取分支列表
2. 执行 `git push --dry-run` 验证推送权限
3. 返回连接状态与权限详情

**响应:**
```json
// 成功
{
  "code": 0,
  "data": {
    "connected": true,
    "branches": ["main", "develop", "staging"],
    "permissions": {
      "read": true,
      "push": true
    }
  }
}

// 读取成功但无推送权限
{
  "code": 0,
  "data": {
    "connected": true,
    "branches": ["main", "develop", "staging"],
    "permissions": {
      "read": true,
      "push": false
    }
  }
}

// 连接失败
{
  "code": 0,
  "data": {
    "connected": false,
    "error": "authentication failed: token expired or revoked",
    "permissions": {
      "read": false,
      "push": false
    }
  }
}
```

---

### 5.7 触发仓库分析

**POST** `/repos/:id/analyze`

**权限:** 项目 RD 成员, admin

Clone 仓库并使用 Claude Code 分析其结构、技术栈、模块功能。

**前置条件:** 没有正在进行的分析任务。

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 1,
    "analysis_status": "running",
    "message": "分析任务已启动"
  }
}
```

**错误响应:**
```json
{ "code": 40003, "message": "分析任务正在进行中，请稍后" }
{ "code": 50102, "message": "仓库连接失败，请检查 access token" }
```

---

### 5.8 获取分析结果

**GET** `/repos/:id/analysis`

**响应:**
```json
{
  "code": 0,
  "data": {
    "analysis_status": "completed",
    "analyzed_at": "2026-02-12T10:05:00Z",
    "result": {
      "modules": [
        { "path": "internal/handler", "description": "HTTP 接口层，基于 Gin 框架", "files_count": 12 },
        { "path": "internal/service", "description": "业务逻辑层", "files_count": 8 },
        { "path": "internal/model", "description": "数据模型定义", "files_count": 6 },
        { "path": "internal/repository", "description": "数据访问层", "files_count": 6 }
      ],
      "tech_stack": ["Go 1.21", "Gin", "GORM", "MySQL", "Redis"],
      "entry_points": ["cmd/server/main.go"],
      "directory_structure": "标准 Go 项目布局: cmd / internal / pkg",
      "code_style": {
        "naming": "camelCase for variables, PascalCase for exported",
        "error_handling": "github.com/pkg/errors wrap pattern",
        "test_framework": "testify"
      }
    }
  }
}

// 分析中
{
  "code": 0,
  "data": {
    "analysis_status": "running",
    "analyzed_at": null,
    "result": null
  }
}

// 分析失败
{
  "code": 0,
  "data": {
    "analysis_status": "failed",
    "analysis_error": "克隆仓库失败: authentication required",
    "analyzed_at": null,
    "result": null
  }
}
```

---

## 6. 需求管理 (Requirements)

### 6.1 创建需求

**POST** `/projects/:id/requirements`

**权限:** pm, admin

**请求:**
```json
{
  "title": "新增用户注册功能",
  "description": "## 功能描述\n支持手机号+验证码注册\n\n## 验收标准\n1. 用户输入手机号获取验证码\n2. 输入验证码完成注册\n3. 注册成功后自动登录",
  "doc_links": [
    { "title": "注册流程 PRD", "url": "https://xxx.feishu.cn/docs/req-001" }
  ],
  "priority": "p1",
  "deadline": "2026-03-01T00:00:00Z",
  "assignee_id": 3,
  "repository_id": 1
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| title | string | 是 | 1-256 字符 | 需求标题 |
| description | string | 是 | 1-50000 字符 | 需求详细描述 (支持 Markdown) |
| doc_links | array | 否 | | 关联飞书文档链接 |
| doc_links[].title | string | 是 | 1-128 字符 | 文档标题 |
| doc_links[].url | string | 是 | 合法 URL | 文档链接 |
| priority | string | 否 | p0 / p1 / p2 / p3 | 优先级，默认 p1 |
| deadline | string | 否 | ISO 8601 | 期望完成时间 |
| assignee_id | int | 否 | 项目成员 ID | 指派的开发人员 |
| repository_id | int | 否 | 项目关联的仓库 ID | 目标代码仓库 |

**后端行为:** 如果有 `doc_links`，异步通过飞书 API 抓取文档内容存入 `doc_content`。

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 15,
    "title": "新增用户注册功能",
    "description": "...",
    "doc_links": [
      { "title": "注册流程 PRD", "url": "https://xxx.feishu.cn/docs/req-001" }
    ],
    "priority": "p1",
    "deadline": "2026-03-01T00:00:00Z",
    "status": "draft",
    "creator": { "id": 1, "name": "张三", "avatar": "..." },
    "assignee": { "id": 3, "name": "王五", "avatar": "..." },
    "repository": { "id": 1, "name": "user-service" },
    "created_at": "2026-02-12T11:00:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 40001, "message": "参数校验失败: title 不能为空" }
{ "code": 40002, "message": "assignee_id 必须是项目 RD 成员" }
{ "code": 40002, "message": "repository_id 必须是项目关联的仓库" }
```

---

### 6.2 需求列表

**GET** `/projects/:id/requirements`

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |
| status | string | 否 | 按状态筛选: draft / generating / generated / reviewing / approved / merged / rejected |
| priority | string | 否 | 按优先级筛选: p0 / p1 / p2 / p3 |
| assignee_id | int | 否 | 按指派人筛选 |
| creator_id | int | 否 | 按创建者筛选 |
| keyword | string | 否 | 按标题模糊搜索 |
| sort_by | string | 否 | created_at / updated_at / priority，默认 created_at |
| order | string | 否 | asc / desc，默认 desc |

**响应:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 15,
        "title": "新增用户注册功能",
        "priority": "p1",
        "status": "generated",
        "deadline": "2026-03-01T00:00:00Z",
        "creator": { "id": 1, "name": "张三", "avatar": "..." },
        "assignee": { "id": 3, "name": "王五", "avatar": "..." },
        "repository": { "id": 1, "name": "user-service" },
        "latest_codegen": {
          "id": 42,
          "status": "completed",
          "created_at": "2026-02-12T11:05:00Z"
        },
        "latest_review": {
          "id": 10,
          "ai_score": 85,
          "human_status": "pending"
        },
        "created_at": "2026-02-12T11:00:00Z",
        "updated_at": "2026-02-12T11:10:00Z"
      }
    ],
    "total": 8,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 6.3 需求详情

**GET** `/requirements/:id`

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 15,
    "project": { "id": 1, "name": "用户中台" },
    "title": "新增用户注册功能",
    "description": "## 功能描述\n支持手机号+验证码注册...",
    "doc_links": [
      { "title": "注册流程 PRD", "url": "https://xxx.feishu.cn/docs/req-001" }
    ],
    "doc_content_status": "fetched",
    "priority": "p1",
    "deadline": "2026-03-01T00:00:00Z",
    "status": "generated",
    "creator": { "id": 1, "name": "张三", "avatar": "..." },
    "assignee": { "id": 3, "name": "王五", "avatar": "..." },
    "repository": { "id": 1, "name": "user-service", "platform": "gitlab" },
    "codegen_tasks": [
      {
        "id": 42,
        "status": "completed",
        "target_branch": "feature/req-15-user-registration",
        "diff_stat": { "files_changed": 5, "additions": 230, "deletions": 12 },
        "started_at": "2026-02-12T11:05:10Z",
        "completed_at": "2026-02-12T11:08:45Z",
        "created_at": "2026-02-12T11:05:00Z"
      },
      {
        "id": 38,
        "status": "failed",
        "target_branch": "feature/req-15-user-registration",
        "error_message": "Claude Code 执行超时",
        "created_at": "2026-02-12T10:00:00Z"
      }
    ],
    "latest_review": {
      "id": 10,
      "ai_score": 85,
      "ai_status": "passed",
      "human_status": "pending"
    },
    "created_at": "2026-02-12T11:00:00Z",
    "updated_at": "2026-02-12T11:10:00Z"
  }
}
```

---

### 6.4 更新需求

**PUT** `/requirements/:id`

**权限:** 创建者或 admin

**前置条件:** 状态为 draft 或 rejected 时可编辑。

**请求:**
```json
{
  "title": "新增用户注册功能 (含邮箱)",
  "description": "更新后的描述...",
  "doc_links": [...],
  "priority": "p0",
  "deadline": "2026-03-15T00:00:00Z",
  "assignee_id": 5,
  "repository_id": 2
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| title | string | 否 | 1-256 字符 | |
| description | string | 否 | 1-50000 字符 | |
| doc_links | array | 否 | | 全量替换 |
| priority | string | 否 | p0-p3 | |
| deadline | string | 否 | ISO 8601 | 期望完成时间 |
| assignee_id | int | 否 | 项目成员 | |
| repository_id | int | 否 | 项目关联仓库 | |

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 15,
    "title": "新增用户注册功能 (含邮箱)",
    "priority": "p0",
    "status": "draft",
    "updated_at": "2026-02-12T12:00:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 40003, "message": "需求当前状态为 generating，不可编辑" }
{ "code": 40303, "message": "非需求创建者，无权编辑" }
```

---

### 6.5 删除需求

**DELETE** `/requirements/:id`

**权限:** 创建者或 admin

**前置条件:** 状态为 draft 时可删除。其他状态需先确认。

**请求:**
```json
{
  "force": false
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| force | bool | 否 | true 时强制删除 (非 generating 状态)，默认 false |

**响应:**
```json
{
  "code": 0,
  "data": { "message": "需求已删除" }
}
```

**错误响应:**
```json
{ "code": 40003, "message": "需求当前状态为 generating，不可删除" }
{ "code": 40003, "message": "需求非 draft 状态，需要 force=true 确认删除" }
```

---

### 6.6 全局需求列表

**GET** `/requirements`

跨项目获取当前用户有权限查看的需求。

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |
| scope | string | 否 | `all` (默认) / `created` / `assigned` |
| status | string | 否 | 按状态筛选 |
| keyword | string | 否 | 按标题模糊搜索 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 15,
        "title": "新增用户注册功能",
        "priority": "p1",
        "status": "generated",
        "deadline": "2026-03-01T00:00:00Z",
        "project": { "id": 1, "name": "用户中台" },
        "creator": { "id": 1, "name": "张三", "avatar": "..." },
        "assignee": { "id": 3, "name": "王五", "avatar": "..." },
        "repository": { "id": 1, "name": "user-service" },
        "created_at": "2026-02-12T11:00:00Z",
        "updated_at": "2026-02-12T11:10:00Z"
      }
    ],
    "total": 20,
    "page": 1,
    "page_size": 20
  }
}
```

---

## 7. 代码生成 (CodeGen) -- 核心

### 7.1 触发代码生成

**POST** `/requirements/:id/generate`

**权限:** 需求的 assignee (RD), admin

**前置条件:**
- 需求状态为 draft / rejected / generated (可重新生成)
- 需求已关联仓库 (`repository_id` 非空)
- 需求已关联 RD (`assignee_id` 非空)
- 该需求没有正在运行中的生成任务

**请求:**
```json
{
  "extra_context": "参考现有 login handler 的实现风格，使用相同的错误处理模式",
  "source_branch": "develop"
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| extra_context | string | 否 | 最大 5000 字符 | 给 AI 的补充说明 |
| source_branch | string | 否 | 合法分支名 | 基于哪个分支，默认取仓库 default_branch |

**后端行为:**
1. 创建 `codegen_tasks` 记录，status=pending
2. 更新需求 status=generating
3. 将任务推入执行队列
4. 返回任务 ID

**响应:**
```json
{
  "code": 0,
  "data": {
    "task_id": 42,
    "status": "pending",
    "source_branch": "develop",
    "target_branch": "feature/req-15-user-registration",
    "queue_position": 0
  }
}
```

`queue_position`: 队列中的等待位置，0 表示立即执行。

**错误响应:**
```json
{ "code": 40004, "message": "需求未关联代码仓库，请先关联仓库" }
{ "code": 40004, "message": "需求未指派 RD，请先指派开发人员" }
{ "code": 40003, "message": "该需求已有生成任务正在运行中" }
{ "code": 40003, "message": "需求当前状态为 reviewing，不可重新生成" }
{ "code": 50102, "message": "仓库连接失败，请检查 access token" }
```

---

### 7.2 实时获取生成过程 (SSE)

**GET** `/codegen/:id/stream`

**协议:** Server-Sent Events (SSE)

**Headers:**
```
Accept: text/event-stream
Authorization: Bearer <token>
Last-Event-ID: 156        // 可选，断线重连时携带
```

**认证:** 由于浏览器 EventSource API 不支持自定义 Header，也支持通过 query param 传递 token:
```
GET /codegen/:id/stream?token=<jwt-token>
```

**断线重连机制:**
1. 浏览器 EventSource 断线后自动携带 `Last-Event-ID` header
2. 服务端从 Redis 中获取该 ID 之后的事件回放
3. 回放完成后切换到实时推送

**行为:**
1. 如果任务已完成 (completed/failed/cancelled): 从 Redis 回放全部历史事件 → 发送 `event: done` → 关闭连接
2. 如果任务进行中: 回放已有事件 → 切换为实时推送
3. 如果任务待执行: 等待任务开始后推送

**事件类型:**

所有事件按时间顺序存储在 Redis 中，任务完成后可回放完整记录。`event: log` 和 `event: output` 共同组成统一的输出时间线，前端按接收顺序混合展示，呈现完整的 **Git 克隆 → Claude Code 启动 → 代码生成 → 推送** 流程。

#### `event: status` -- 任务状态变更
```
id: 1
event: status
data: {"status":"cloning","message":"正在克隆仓库..."}

id: 15
event: status
data: {"status":"running","message":"Claude Code 已启动","pid":12345}

id: 250
event: status
data: {"status":"completed","files_changed":5,"additions":230,"deletions":12}
```

当 `status` 为 `running` 时，额外携带 `pid` 字段表示 Claude Code 进程 ID。

#### `event: log` -- 系统操作日志

记录 Git 操作、Claude Code 启动/结束、代码推送等系统级过程。所有事件持久化存储，支持历史回放。

`level`: `info` / `warn` / `error`
`phase`: `clone` / `claude` / `push`

```
// ---- 阶段 1: Git 克隆 ----
id: 2
event: log
data: {"level":"info","phase":"clone","message":"开始克隆仓库","detail":{"git_url":"https://gitlab.com/company/user-service.git","branch":"develop","work_dir":"/tmp/codemaster/codegen/42"}}

id: 3
event: log
data: {"level":"info","phase":"clone","message":"仓库克隆完成"}

id: 4
event: log
data: {"level":"info","phase":"clone","message":"已创建工作分支","detail":{"branch":"feature/req-15-user-registration"}}

// ---- 阶段 2: Claude Code 启动 ----
id: 5
event: log
data: {"level":"info","phase":"claude","message":"Claude Code 启动参数","detail":{"command":"claude","args":["-p","...","--output-format","stream-json","--allowedTools","Read,Write,Edit,Glob,Grep,Bash","--max-turns","50"],"work_dir":"/tmp/codemaster/codegen/42","timeout_min":30}}

id: 6
event: log
data: {"level":"info","phase":"claude","message":"Claude Code 进程已启动","detail":{"pid":12345}}

// ---- 阶段 3: Claude Code 输出 (event: output，见下方) ----

// ---- 阶段 4: 完成与推送 ----
id: 200
event: log
data: {"level":"info","phase":"claude","message":"Claude Code 执行完成","detail":{"cost_usd":0.0523}}

id: 201
event: log
data: {"level":"info","phase":"push","message":"正在推送代码到远程仓库","detail":{"branch":"feature/req-15-user-registration"}}

id: 202
event: log
data: {"level":"info","phase":"push","message":"代码推送完成"}

// ---- 错误场景 ----
id: 100
event: log
data: {"level":"error","phase":"clone","message":"克隆仓库失败","detail":{"error":"authentication required"}}
```

#### `event: output` -- Claude Code 实时输出

与 `event: log` 共同组成统一输出时间线。Claude Code 进程的 stdout 流式解析结果。

```
// AI 思考过程
id: 3
event: output
data: {"type":"thinking","content":"我需要先了解现有的 handler 层结构..."}

// AI 文本回复
id: 10
event: output
data: {"type":"text","content":"我将在 internal/handler/ 下新增 register.go 文件..."}

// 工具调用 - 读取文件
id: 15
event: output
data: {"type":"tool_use","tool":"Read","input":{"file_path":"internal/handler/user.go"},"id":"tool_1"}

// 工具调用结果
id: 16
event: output
data: {"type":"tool_result","id":"tool_1","summary":"读取了 user.go (85 行)"}

// 工具调用 - 写入文件
id: 20
event: output
data: {"type":"tool_use","tool":"Write","input":{"file_path":"internal/handler/register.go","content_preview":"package handler\n\nimport ..."},"id":"tool_2"}

// 工具调用 - 编辑文件
id: 30
event: output
data: {"type":"tool_use","tool":"Edit","input":{"file_path":"internal/router/router.go","description":"添加注册路由"},"id":"tool_3"}

// 工具调用 - 执行命令
id: 40
event: output
data: {"type":"tool_use","tool":"Bash","input":{"command":"go build ./..."},"id":"tool_4"}

// 命令执行结果
id: 41
event: output
data: {"type":"tool_result","id":"tool_4","output":"Build succeeded","exit_code":0}
```

#### `event: progress` -- 进度摘要 (每次工具调用后推送)
```
id: 42
event: progress
data: {"files_read":3,"files_written":2,"files_edited":1,"turns_used":12,"max_turns":50,"current_action":"writing internal/handler/register.go"}
```

#### `event: task_error` -- 任务错误
```
id: 100
event: task_error
data: {"message":"Claude Code 进程异常退出","code":"PROCESS_CRASHED"}
```

> **注意:** 使用 `task_error` 而非 `error`，因为 SSE 规范中 `event: error` 会触发 EventSource 的原生 `onerror` 处理器，导致连接关闭和无限重连。

#### `event: done` -- 完成
```
id: 251
event: done
data: {"task_id":42,"status":"completed","review_id":10}
```

`review_id` 非空表示已自动触发 AI Review。

**SSE 心跳:** 每 30 秒发送一次空注释保持连接:
```
: heartbeat
```

---

### 7.3 获取生成任务详情

**GET** `/codegen/:id`

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 42,
    "requirement": {
      "id": 15,
      "title": "新增用户注册功能"
    },
    "repository": { "id": 1, "name": "user-service", "platform": "gitlab", "git_url": "https://gitlab.com/company/user-service.git" },
    "source_branch": "develop",
    "target_branch": "feature/req-15-user-registration",
    "status": "completed",
    "extra_context": "参考现有 login handler 的实现风格",
    "prompt": "你是一个高级软件工程师...(完整 prompt)",
    "diff_stat": {
      "files_changed": 5,
      "additions": 230,
      "deletions": 12,
      "files": [
        { "path": "internal/handler/register.go", "status": "added", "additions": 85, "deletions": 0 },
        { "path": "internal/router/router.go", "status": "modified", "additions": 3, "deletions": 0 },
        { "path": "internal/service/register.go", "status": "added", "additions": 62, "deletions": 0 },
        { "path": "internal/model/register.go", "status": "added", "additions": 22, "deletions": 0 },
        { "path": "internal/handler/register_test.go", "status": "added", "additions": 58, "deletions": 12 }
      ]
    },
    "commit_sha": "a1b2c3d4e5f6",
    "claude_cost_usd": 0.0523,
    "review": {
      "id": 10,
      "ai_score": 85,
      "ai_status": "passed",
      "human_status": "pending"
    },
    "started_at": "2026-02-12T11:05:10Z",
    "completed_at": "2026-02-12T11:08:45Z",
    "created_at": "2026-02-12T11:05:00Z"
  }
}
```

> `extra_context` 为用户在触发生成时提供的补充说明。`commit_sha` 为推送后的 commit hash。`error_message` 在任务失败时返回。

---

### 7.4 需求的生成任务历史

**GET** `/requirements/:id/codegen-tasks`

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 42,
        "status": "completed",
        "target_branch": "feature/req-15-user-registration",
        "diff_stat": { "files_changed": 5, "additions": 230, "deletions": 12 },
        "claude_cost_usd": 0.0523,
        "started_at": "2026-02-12T11:05:10Z",
        "completed_at": "2026-02-12T11:08:45Z",
        "created_at": "2026-02-12T11:05:00Z"
      },
      {
        "id": 38,
        "status": "failed",
        "target_branch": "feature/req-15-user-registration",
        "error_message": "Claude Code 执行超时",
        "started_at": "2026-02-12T10:00:10Z",
        "completed_at": "2026-02-12T10:10:10Z",
        "created_at": "2026-02-12T10:00:00Z"
      }
    ],
    "total": 2,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 7.5 获取代码 Diff

**GET** `/codegen/:id/diff`

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| file | string | 否 | 只返回指定文件的 diff (路径，如 `internal/handler/register.go`) |
| format | string | 否 | `unified` (默认) / `split` |

**前置条件:** 任务状态为 completed。

**响应:**
```json
{
  "code": 0,
  "data": {
    "target_branch": "feature/req-15-user-registration",
    "base_branch": "develop",
    "files": [
      {
        "path": "internal/handler/register.go",
        "status": "added",
        "language": "go",
        "additions": 85,
        "deletions": 0,
        "diff": "--- /dev/null\n+++ b/internal/handler/register.go\n@@ -0,0 +1,85 @@\n+package handler\n+..."
      },
      {
        "path": "internal/router/router.go",
        "status": "modified",
        "language": "go",
        "additions": 3,
        "deletions": 0,
        "diff": "--- a/internal/router/router.go\n+++ b/internal/router/router.go\n@@ -15,6 +15,9 @@..."
      }
    ]
  }
}
```

**错误响应:**
```json
{ "code": 40003, "message": "任务尚未完成，无法获取 diff" }
```

---

### 7.6 获取完整输出日志

**GET** `/codegen/:id/log`

用于任务完成后回看完整的 Claude Code 输出日志 (非 SSE，普通 JSON 响应)。

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| offset | int | 否 | 起始事件偏移量，默认 0 |
| limit | int | 否 | 返回事件数量，默认 500，最大 1000 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "task_id": 42,
    "status": "completed",
    "total_events": 251,
    "events": [
      { "id": 1, "type": "status", "data": {"status": "cloning", "message": "正在克隆仓库..."} },
      { "id": 2, "type": "log", "data": {"level": "info", "phase": "clone", "message": "开始克隆仓库", "detail": {"git_url": "...", "branch": "develop"}} },
      { "id": 3, "type": "log", "data": {"level": "info", "phase": "clone", "message": "仓库克隆完成"} },
      { "id": 4, "type": "log", "data": {"level": "info", "phase": "claude", "message": "Claude Code 启动参数", "detail": {"command": "claude", "args": ["..."]}} },
      { "id": 5, "type": "status", "data": {"status": "running", "message": "Claude Code 已启动", "pid": 12345} },
      { "id": 6, "type": "output", "data": {"type": "thinking", "content": "我需要..."} },
      ...
    ],
    "has_more": false
  }
}
```

---

### 7.7 取消生成任务

**POST** `/codegen/:id/cancel`

**权限:** 任务触发者或 admin

**前置条件:** 任务状态为 pending / cloning / running。

**后端行为:** 向 Claude Code 子进程发送 SIGTERM，更新状态为 cancelled。

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 42,
    "status": "cancelled",
    "cancelled_at": "2026-02-12T11:07:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 40003, "message": "任务已完成，无法取消" }
```

---

### 7.8 手动提交代码

**POST** `/requirements/:id/manual-submit`

用于手动提交已在本地完成的代码 (不通过 Claude Code 自动生成)。

**前置条件:** 需求已关联仓库。

**请求:**
```json
{
  "source_branch": "feature/req-15-user-registration",
  "commit_message": "feat: 完成用户注册功能",
  "commit_url": "https://gitlab.com/company/user-service/-/commit/abc123"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| source_branch | string | 否 | 提交代码所在的分支 |
| commit_message | string | 否 | 提交信息 |
| commit_url | string | 否 | 提交链接 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "task_id": 50,
    "status": "completed",
    "source_branch": "feature/req-15-user-registration",
    "target_branch": "feature/req-15-user-registration"
  }
}
```

**错误响应:**
```json
{ "code": 40404, "message": "需求不存在" }
{ "code": 40004, "message": "需求未关联代码仓库，请先关联仓库" }
```

---

## 8. 代码 Review

### 8.1 触发 AI Review

**POST** `/codegen/:id/review`

通常在代码生成完成后自动触发，也可手动触发重新 Review。

**前置条件:** 任务状态为 completed。

**请求 (可选):**
```json
{
  "reviewer_ids": [3, 5]
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| reviewer_ids | array[int] | 否 | 指定人工 Reviewer 的用户 ID 列表 |

**后端行为:**
1. 获取 target_branch 与 source_branch 之间的 diff
2. 构造 Review prompt 调用 Claude Code (只读模式)
3. 解析输出为结构化 JSON
4. 写入 code_reviews 表

**响应:**
```json
{
  "code": 0,
  "data": {
    "review_id": 10,
    "ai_status": "running",
    "message": "AI Review 已启动"
  }
}
```

**错误响应:**
```json
{ "code": 40003, "message": "生成任务尚未完成，无法 Review" }
{ "code": 40003, "message": "AI Review 正在进行中" }
```

---

### 8.2 获取 Review 结果 (按 CodeGen 任务)

**GET** `/codegen/:id/review`

通过 CodeGen 任务 ID 获取关联的 Review，返回简要信息。

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 10,
    "codegen_task_id": 42,
    "ai_review": {
      "score": 85,
      "summary": "代码整体质量良好，结构清晰。有两处需要关注的问题。",
      "issues": [...],
      "categories": {
        "security": { "status": "passed", "details": "未发现安全问题" },
        "error_handling": { "status": "warning", "details": "部分错误处理可改进" },
        "code_style": { "status": "passed", "details": "符合项目风格" },
        "test_coverage": { "status": "warning", "details": "建议补充单元测试" }
      }
    },
    "ai_score": 85,
    "ai_status": "passed",
    "human_reviewer": null,
    "human_comment": null,
    "human_status": "pending",
    "merge_request_url": null,
    "merge_status": "none",
    "created_at": "2026-02-12T11:09:00Z",
    "updated_at": "2026-02-12T11:09:30Z"
  }
}
```

---

### 8.3 获取 Review 详情 (按 Review ID)

**GET** `/reviews/:id`

通过 Review ID 获取完整 Review 详情，包含关联的需求、仓库、分支等信息。

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 10,
    "codegen_task_id": 42,
    "requirement": { "id": 15, "title": "新增用户注册功能" },
    "repository": { "id": 1, "name": "user-service" },
    "ai_review": {
      "score": 85,
      "summary": "代码整体质量良好，结构清晰。有两处需要关注的问题。",
      "issues": [
        {
          "severity": "warning",
          "file": "internal/handler/register.go",
          "line": 45,
          "code_snippet": "phone := c.PostForm(\"phone\")",
          "message": "缺少手机号格式校验",
          "suggestion": "建议使用正则校验手机号格式"
        }
      ],
      "categories": {
        "security": { "status": "passed", "details": "未发现安全问题" },
        "error_handling": { "status": "warning", "details": "部分错误处理可改进" }
      }
    },
    "ai_score": 85,
    "ai_status": "passed",
    "human_reviewer": { "id": 3, "name": "王五" },
    "human_comment": "整体没问题",
    "human_status": "approved",
    "merge_request_url": "https://gitlab.com/company/user-service/-/merge_requests/789",
    "merge_status": "created",
    "source_branch": "feature/req-15-user-registration",
    "target_branch": "develop",
    "git_url": "https://gitlab.com/company/user-service.git",
    "platform": "gitlab",
    "diff_stat": {
      "files_changed": 5,
      "additions": 230,
      "deletions": 12
    },
    "reviewers": [
      { "id": 3, "name": "王五" },
      { "id": 5, "name": "赵六" }
    ],
    "created_at": "2026-02-12T11:09:00Z",
    "updated_at": "2026-02-12T11:09:30Z"
  }
}
```

**错误响应:**
```json
{ "code": 40406, "message": "Review 记录不存在" }
```

---

### 8.4 我的待审查列表

**GET** `/reviews/pending`

返回当前用户需要进行人工 Review 的列表。

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |
| project_id | int | 否 | 按项目筛选 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "list": [
      {
        "id": 10,
        "codegen_task_id": 42,
        "requirement": { "id": 15, "title": "新增用户注册功能" },
        "project": { "id": 1, "name": "用户中台" },
        "repository": { "id": 1, "name": "user-service" },
        "creator": { "id": 1, "name": "张三", "avatar": "..." },
        "ai_score": 85,
        "ai_status": "passed",
        "ai_summary": "代码整体质量良好，结构清晰。有两处需要关注的问题。",
        "human_reviewer": { "id": 3, "name": "王五" },
        "human_status": "pending",
        "merge_status": "none",
        "target_branch": "feature/req-15-user-registration",
        "diff_stat": { "files_changed": 5, "additions": 230, "deletions": 12 },
        "created_at": "2026-02-12T11:09:00Z"
      }
    ],
    "total": 3,
    "page": 1,
    "page_size": 20
  }
}
```

---

### 8.5 审查列表

**GET** `/reviews/list`

获取审查列表，支持按审查状态筛选。

**Query 参数:**

| 参数 | 类型 | 必填 | 说明 |
|------|------|------|------|
| page | int | 否 | 页码 |
| page_size | int | 否 | 每页数量 |
| human_status | string | 否 | 按人工审查状态筛选: pending / approved / rejected / needs_revision |
| project_id | int | 否 | 按项目筛选 |

**响应:**

与 8.4 待审查列表响应格式相同。

---

### 8.6 人工 Review 提交

**PUT** `/reviews/:id/human`

**权限:** 项目成员, admin

**前置条件:** ai_status 不为 running (AI Review 需先完成)。

**请求:**
```json
{
  "comment": "整体没问题，但 register handler 里建议加上 rate limit",
  "status": "approved"
}
```

| 字段 | 类型 | 必填 | 校验 | 说明 |
|------|------|------|------|------|
| comment | string | 条件必填 | 最大 5000 字符 | 审查意见 (rejected/needs_revision 时必填) |
| status | string | 是 | approved / rejected / needs_revision | 审查结果 |

**响应:**
```json
{
  "code": 0,
  "data": {
    "id": 10,
    "human_reviewer": { "id": 3, "name": "王五" },
    "human_comment": "整体没问题，但 register handler 里建议加上 rate limit",
    "human_status": "approved",
    "updated_at": "2026-02-12T12:00:00Z"
  }
}
```

**错误响应:**
```json
{ "code": 40003, "message": "AI Review 尚未完成，请等待" }
{ "code": 40001, "message": "拒绝时必须填写审查意见" }
```

---

### 8.7 创建合并请求

**POST** `/reviews/:id/merge-request`

**权限:** 人工 Review 的审查者, admin

**前置条件:** human_status = approved

**后端行为:**
1. 通过 GitLab/GitHub API 创建 Merge Request
2. source_branch = feature/req-xxx, target_branch = develop
3. MR 描述中包含需求信息、AI Review 摘要、人工 Review 意见
4. 更新 code_reviews 中 MR 信息

**响应:**
```json
{
  "code": 0,
  "data": {
    "review_id": 10,
    "merge_request_id": "789",
    "merge_request_url": "https://gitlab.com/company/user-service/-/merge_requests/789",
    "merge_status": "created"
  }
}
```

**错误响应:**
```json
{ "code": 40004, "message": "人工审查尚未通过，无法创建合并请求" }
{ "code": 50101, "message": "创建合并请求失败: branch not found" }
{ "code": 40005, "message": "合并请求已创建，请勿重复操作" }
```

---

### 8.8 查看合并请求状态

**GET** `/reviews/:id/merge-request`

**响应:**
```json
{
  "code": 0,
  "data": {
    "merge_request_id": "789",
    "merge_request_url": "https://gitlab.com/company/user-service/-/merge_requests/789",
    "merge_status": "merged",
    "title": "feat(req-15): 新增用户注册功能",
    "ci_status": "passed",
    "merged_at": "2026-02-12T12:00:00Z"
  }
}
```

**未创建 MR 时:**
```json
{
  "code": 0,
  "data": {
    "merge_status": "none",
    "merge_request_url": null
  }
}
```

---

## 9. 个人设置 (Settings)

### 9.1 获取 LLM 设置

**GET** `/settings/llm`

获取当前用户的 LLM 及 Git Token 配置。敏感字段以脱敏形式返回。

**响应:**
```json
{
  "code": 0,
  "data": {
    "base_url": "https://api.anthropic.com",
    "api_key": "sk-****abcd",
    "model": "claude-sonnet-4-20250514",
    "gitlab_token": "****wxyz"
  }
}
```

> `api_key` 和 `gitlab_token` 返回时脱敏处理，仅显示末 4 位。

---

### 9.2 更新 LLM 设置

**PUT** `/settings/llm`

**请求:**
```json
{
  "base_url": "https://api.anthropic.com",
  "api_key": "sk-ant-xxxxxxxxxxxx",
  "model": "claude-sonnet-4-20250514",
  "gitlab_token": "glpat-xxxxxxxxxxxx"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| base_url | string | 否 | LLM API 地址 |
| api_key | string | 否 | LLM API Key (如包含 `****` 则保留原值) |
| model | string | 否 | 模型名称 |
| gitlab_token | string | 否 | GitLab Personal Access Token (如包含 `****` 则保留原值) |

> 前端将脱敏值原样回传时 (如 `sk-****abcd`)，后端自动保留原始密钥不做更新。

**响应:**
```json
{
  "code": 0,
  "data": {
    "base_url": "https://api.anthropic.com",
    "api_key": "sk-****xxxx",
    "model": "claude-sonnet-4-20250514",
    "gitlab_token": "****xxxx"
  }
}
```

---

## 10. 飞书工具 (Feishu)

### 10.1 解析飞书文档

**POST** `/feishu/doc/resolve`

从飞书文档 URL 中提取文档元信息。

**请求:**
```json
{
  "url": "https://xxx.feishu.cn/docs/doccnXXXXX"
}
```

| 字段 | 类型 | 必填 | 说明 |
|------|------|------|------|
| url | string | 是 | 飞书文档 URL |

**响应:**
```json
{
  "code": 0,
  "data": {
    "title": "用户注册功能 PRD",
    "document_id": "doccnXXXXX",
    "url": "https://xxx.feishu.cn/docs/doccnXXXXX"
  }
}
```

**错误响应:**
```json
{ "code": 40001, "message": "无法从 URL 中提取文档 ID" }
{ "code": 40002, "message": "获取文档信息失败: ..." }
```

---

## 11. Dashboard

### 11.1 首页统计数据

**GET** `/dashboard/stats`

**响应:**
```json
{
  "code": 0,
  "data": {
    "my_projects": 3,
    "my_open_requirements": 5,
    "my_pending_reviews": 2,
    "codegen_running": 1,
    "recent_activity": [
      {
        "type": "codegen_completed",
        "requirement": { "id": 15, "title": "新增用户注册功能" },
        "project": { "id": 1, "name": "用户中台" },
        "time": "2026-02-12T11:08:45Z"
      },
      {
        "type": "review_approved",
        "requirement": { "id": 12, "title": "优化登录流程" },
        "project": { "id": 1, "name": "用户中台" },
        "reviewer": { "id": 3, "name": "王五" },
        "time": "2026-02-12T10:30:00Z"
      }
    ]
  }
}
```

---

### 11.2 我的待办

**GET** `/dashboard/my-tasks`

**响应:**
```json
{
  "code": 0,
  "data": {
    "pending_generate": [
      {
        "requirement_id": 20,
        "title": "新增支付功能",
        "project": { "id": 2, "name": "支付中台" },
        "priority": "p0",
        "created_at": "2026-02-12T09:00:00Z"
      }
    ],
    "running_tasks": [
      {
        "task_id": 45,
        "requirement": { "id": 18, "title": "优化搜索性能" },
        "status": "running",
        "started_at": "2026-02-12T11:10:00Z"
      }
    ],
    "pending_reviews": [
      {
        "review_id": 12,
        "requirement": { "id": 16, "title": "新增导出功能" },
        "ai_score": 92,
        "created_at": "2026-02-12T10:00:00Z"
      }
    ]
  }
}
```

---

## 12. 接口权限矩阵

> **权限模型说明:** 系统使用"业务角色 + 管理员"双轨模型:
> - `role`: 业务角色，`pm` (产品经理) 或 `rd` (研发工程师)
> - `is_admin`: 管理员标记 (独立于业务角色)，管理员自动拥有所有业务权限

| 模块 | 接口 | PM | RD | is_admin | 附加条件 |
|------|------|:--:|:--:|:--------:|----------|
| 认证 | 飞书登录/回调 | Y | Y | Y | 无需 token |
| 认证 | 获取用户信息 | Y | Y | Y | |
| 认证 | 选择角色 | Y | Y | Y | 仅首次 / admin 改他人 |
| 认证 | 刷新 Token | Y | Y | Y | |
| 用户 | 搜索用户 | Y | Y | Y | |
| 管理 | 用户列表 | - | - | Y | |
| 管理 | 修改角色 | - | - | Y | |
| 管理 | 设置/取消管理员 | - | - | Y | |
| 管理 | 禁用/启用用户 | - | - | Y | |
| 管理 | 操作日志 | - | - | Y | |
| 项目 | 创建项目 | Y | - | Y | |
| 项目 | 查看项目列表 | Y | Y | Y | 只看自己参与的 |
| 项目 | 查看项目详情 | Y | Y | Y | 需为项目成员 |
| 项目 | 编辑项目 | Owner | - | Y | |
| 项目 | 归档项目 | Owner | - | Y | |
| 项目 | 添加成员 | Owner | - | Y | |
| 项目 | 移除成员 | Owner | - | Y | 不可移除 owner |
| 仓库 | 关联仓库 | Member | Member | Y | |
| 仓库 | 查看仓库 | Member | Member | Y | |
| 仓库 | 修改仓库 | Member | Member | Y | |
| 仓库 | 解除仓库 | Owner | - | Y | 无运行中任务 |
| 仓库 | 测试连通性 | Member | Member | Y | |
| 仓库 | 触发分析 | Member | Member | Y | |
| 仓库 | 查看分析 | Member | Member | Y | |
| 需求 | 创建需求 | Y | - | Y | |
| 需求 | 查看需求 | Member | Member | Y | |
| 需求 | 编辑需求 | Creator | - | Y | draft/rejected 状态 |
| 需求 | 删除需求 | Creator | - | Y | |
| 需求 | 全局需求列表 | Y | Y | Y | |
| 代码生成 | 触发生成 | - | Assignee | Y | 需关联仓库+RD |
| 代码生成 | 手动提交代码 | Y | Y | Y | 需关联仓库 |
| 代码生成 | 查看进度/详情 | Member | Member | Y | |
| 代码生成 | 查看 Diff | Member | Member | Y | completed 状态 |
| 代码生成 | 查看日志 | Member | Member | Y | |
| 代码生成 | 取消生成 | - | Trigger | Y | running 状态 |
| Review | 触发 AI Review | - | Member | Y | completed 状态 |
| Review | 查看 Review | Member | Member | Y | |
| Review | 审查列表 | Y | Y | Y | |
| Review | 待审查列表 | Y | Y | Y | |
| Review | 人工审查 | - | Member | Y | AI Review 完成后 |
| Review | 创建 MR | - | Reviewer | Y | human_status=approved |
| Review | 查看 MR 状态 | Member | Member | Y | |
| 设置 | 获取 LLM 设置 | Y | Y | Y | |
| 设置 | 更新 LLM 设置 | Y | Y | Y | |
| 飞书 | 解析飞书文档 | Y | Y | Y | |
| Dashboard | 统计/待办 | Y | Y | Y | |

**说明:**
- `Owner`: 项目创建者
- `Member`: 项目成员 (含 PM 和 RD)
- `Creator`: 资源创建者
- `Assignee`: 需求指派的 RD
- `Trigger`: 触发该操作的人
- `Reviewer`: 执行人工 Review 的人
