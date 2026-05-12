# itab 基座管理后台扩展方案(itabbase)

> 给 `itab-xxxxxx` 脚手架引入 NocoBase 风格的数据驱动后端 + 开箱即用的管理后台,保留 vibe-coding 前台不变。
>
> 状态:**设计提案**,待决策项见 §17。

---

## 1. 背景

`itab-xxxxxx` 是 WPS 内部为 vibe-coding 提供的项目脚手架,当前能力:

- Vue3 前端空壳,用户自由开发
- Go(GoFrame)后端 + WPS SSO 登录
- 本地 SQLite / 线上 Postgres(经 GoFrame DB 抽象)
- task-driven 文档化开发流程(见 `CLAUDE.md`)

**现状痛点**:每个新项目都从 0 写 CRUD、权限、迁移、用户管理。基础能力沉淀不下来,vibe-coding 的杠杆没拉满。

**目标**:借鉴 NocoBase 的 collection-driven 思想,提供一个**开箱即用的管理后台**,让用户专注 vibe-code 业务前台,基础数据能力交给基座统一沉淀。

---

## 2. 核心思路

> kernel 与 admin 作为**依赖**,不是模板代码。

不要把基座代码复制进每个用户项目(fork 模式),否则用户开发几周后,基座升级时会陷入 git merge 冲突地狱。改成依赖式分层:

| 层 | 形态 | 升级方式 | 维护方 |
|---|---|---|---|
| **scaffold(空壳)** | 当前 itab-xxxxxx 仓库 | 一次性克隆,不再升级 | 基座团队提供初始版本 |
| **itabbase** | 独立 Go module | `go get -u` | 基座团队 |
| **admin SPA** | 编译进 kernel(`go:embed`) | 跟随 kernel 升级 | 基座团队 |
| **业务代码** | 用户写在 `modules/` 和 `views/` | 用户自己 | 用户 |

**升级动作**:用户跑一条 `go get -u github.com/wps/itabbase`,业务代码零修改即可拿到新 admin 能力。

---

## 3. 整体架构

```
[用户浏览器]
   │
   ├── http://host/         ──▶ Vue 业务前台(用户 vibe-code)
   │                              └── 调 /api/kernel/* + /api/biz/*
   │
   └── http://host/admin/   ──▶ admin SPA(kernel 自带,go:embed)
                                  └── 调 /api/admin/meta/* + /api/kernel/*

[Go 后端进程]
├── /api/kernel/*       ── kernel 自动 CRUD(基于 collection 声明)
├── /api/admin/*        ── kernel 元数据 API(供 admin SPA 使用)
├── /admin/*            ── kernel 吐 admin SPA 静态文件(go:embed)
├── /api/auth/*         ── 现有 WPS SSO(kernel 复用)
└── /api/biz/*          ── 用户自己的业务路由
```

---

## 4. 包结构与边界

### kernel 仓库结构

```
itabbase/
├── go.mod                      module github.com/wps/itabbase
├── kernel.go                   入口:New / Mount
├── collection/                 collection 模型 + 注册器
├── crud/                       CRUD handler 自动生成
├── acl/                        权限引擎
├── meta/                       admin meta API
├── migration/                  系统表 + 用户 collection 同步
├── hook/                       生命周期钩子
├── admin/                      Vue admin 源码(基座维护)
│   ├── src/
│   ├── package.json
│   └── dist/                   CI 产出,被 go:embed 引用
└── admin_embed.go              //go:embed admin/dist
```

### scaffold(用户项目)需要改动的文件

只有两处:

```
server/main.go    增 4 行:import + New + RegisterCollection + Mount
server/go.mod     增一行依赖
```

其他全部不变。

---

## 5. 后端集成示例

### 5.1 升级前

```go
// server/main.go
func main() {
    boot.ApplyGFEnv()
    cmd.Main.Run(gctx.New())
}
```

### 5.2 升级后

```go
import (
    itab "github.com/wps/itabbase"
    "it-ai-base/server/internal/modules/okr"
)

func main() {
    boot.ApplyGFEnv()

    k := itab.New(
        itab.WithDB(g.DB()),                  // 复用 GoFrame DB
        itab.WithAuth(authAdapter{}),         // 接入现有 SSO(见 §6.1)
        itab.WithAdminMount("/admin"),        // admin SPA 路径
    )
    k.RegisterCollection(okr.Objective)
    k.RegisterCollection(okr.KeyResult)
    k.Mount(g.Server())                       // 一次挂载所有路由

    cmd.Main.Run(gctx.New())
}
```

### 5.3 `k.Mount(s)` 内部行为

```go
func (k *Kernel) Mount(s *ghttp.Server) {
    // 1. 跑 kernel 自带系统表 migration(roles, permissions, workflows...)
    k.runSystemMigrations()

    // 2. 同步用户声明的 collection 到 DB(只增不减,见 §11)
    k.syncUserCollections()

    // 3. 挂 CRUD 路由
    s.Group("/api/kernel", func(g *ghttp.RouterGroup) {
        g.Middleware(k.aclMiddleware)
        for _, c := range k.collections {
            g.ALL(c.Name+"/*", k.crudHandler(c))
        }
    })

    // 4. 挂 admin 元数据 API
    s.Group("/api/admin", func(g *ghttp.RouterGroup) {
        g.GET("/meta/collections", k.listCollections)
        g.GET("/meta/acl", k.aclConfig)
    })

    // 5. 挂 admin SPA 静态文件
    s.BindHandler("/admin/*", k.serveAdminSPA)
}
```

---

## 6. 关键扩展点

### 6.1 认证接入契约(AuthAdapter)

kernel 不绑定具体 SSO,通过窄接口接入:

```go
type AuthAdapter interface {
    CurrentUser(r *ghttp.Request) (User, error)  // 解出当前用户
    RolesOf(u User) []string                     // 返回用户角色列表
}
```

scaffold 里的实现(复用现有 `internal/service/auth`):

```go
type authAdapter struct{}

func (authAdapter) CurrentUser(r *ghttp.Request) (itab.User, error) {
    sess := authsvc.GetSession(r)
    if sess == nil {
        return itab.User{}, itab.ErrUnauthenticated
    }
    return itab.User{ID: sess.UserID, Name: sess.UserName}, nil
}

func (authAdapter) RolesOf(u itab.User) []string {
    return userRoleRepo.RolesOf(u.ID)  // 从 DB 查
}
```

**好处**:kernel 永远不知道 WPS SSO 长什么样,以后换登录方式 kernel 不用动。

### 6.2 Collection 钩子(业务自定义逻辑)

```go
var Objective = itab.Collection{
    Name: "objectives",
    Fields: []itab.Field{...},
    Hooks: itab.Hooks{
        BeforeCreate: func(ctx context.Context, rec *itab.Record) error {
            if rec.Get("title") == "" {
                return errors.New("title required")
            }
            rec.Set("created_by", itab.UserFromCtx(ctx).ID)
            return nil
        },
        AfterUpdate: func(ctx context.Context, rec *itab.Record) error {
            // 触发业务事件(发通知 / 推 MQ / 写日志)
            return nil
        },
    },
}
```

钩子覆盖完整生命周期:`BeforeCreate / AfterCreate / BeforeUpdate / AfterUpdate / BeforeDelete / AfterDelete`。

### 6.3 自定义路由(非 CRUD 业务)

```go
k.RegisterCustomRoute(itab.Route{
    Method:  "POST",
    Path:    "/objectives/:id/lock",
    Handler: lockObjective,
    ACL:     itab.RequireRole("owner"),
})
```

挂载后变成 `POST /api/kernel/objectives/:id/lock`,自动走 ACL 中间件,享受统一的鉴权与日志。

---

## 7. Admin SPA 嵌入方式

### 7.1 构建与发版

kernel CI 流水线:

1. `cd admin && pnpm install && pnpm build` → 产出 `admin/dist/`
2. `go build` → `go:embed` 把 dist 包进二进制
3. 打 tag `v1.x.x` → 推内网 Go 仓库

### 7.2 嵌入代码

```go
// admin_embed.go
package itab

import "embed"

//go:embed all:admin/dist
var adminFS embed.FS
```

### 7.3 SPA 服务(支持 history 路由)

```go
func (k *Kernel) serveAdminSPA(r *ghttp.Request) {
    path := strings.TrimPrefix(r.URL.Path, k.adminMount)
    if path == "" || path == "/" {
        path = "/index.html"
    }
    data, err := adminFS.ReadFile("admin/dist" + path)
    if err != nil {
        // SPA history 路由 fallback,所有未命中静态文件的路径都返回 index.html
        data, _ = adminFS.ReadFile("admin/dist/index.html")
    }
    r.Response.Write(data)
}
```

---

## 8. Collection 声明形式

### 8.1 推荐形态:Go struct + 字段定义

```go
var Objective = itab.Collection{
    Name:    "objectives",
    Display: "目标",
    Fields: []itab.Field{
        {Name: "title",       Type: itab.TString,    Required: true, MaxLen: 200},
        {Name: "owner_id",    Type: itab.TBelongsTo, Target: "users"},
        {Name: "quarter",     Type: itab.TString,    Enum: []string{"Q1","Q2","Q3","Q4"}},
        {Name: "progress",    Type: itab.TFloat,     Default: 0},
        {Name: "key_results", Type: itab.THasMany,   Target: "key_results"},
    },
    ACL: itab.ACL{
        "viewer": {"list", "get"},
        "owner":  {"*"},
    },
    Indexes: []itab.Index{
        {Fields: []string{"owner_id", "quarter"}},
    },
}
```

### 8.2 自动产出能力

声明完一个 collection,以下能力立刻具备:

- **数据层**:DB 表 `objectives` + 字段 + 索引(自动同步)
- **REST API**:`/api/kernel/objectives` 的 list/get/create/update/delete
- **关联查询**:`?include=key_results` 自动 JOIN
- **筛选/排序/分页**:`?filter[quarter]=Q2&sort=-created_at&page=2&size=20`
- **权限**:ACL 中间件自动检查每个 action
- **管理后台**:admin 自动出现该集合,可看数据、配权限、批量操作

---

## 9. 运行时全景

### 9.1 用户访问 admin

1. `GET /admin/` → Go 命中 SPA 路由,吐 embed 的 `index.html`
2. SPA 启动 → `GET /api/admin/meta/collections` 拿到所有 collection 定义
3. 用户点 "objectives" → SPA 调 `GET /api/kernel/objectives?page=1&size=20`
4. kernel ACL 中间件检查当前用户是否有 `list` 权限
5. CRUD handler 查 DB 返回数据

### 9.2 用户访问业务前台

1. 用户的 Vue 页面调 `GET /api/kernel/objectives` → 跟 admin 走完全相同的链路
2. SSO cookie 同源共享,无需额外接 token

---

## 10. 升级与版本管理

### 10.1 升级流程

```bash
cd server
go get -u github.com/wps/itabbase
go run .
# kernel 启动时自动跑新版 migration
# 访问 /admin 看到新功能
```

### 10.2 兼容性纪律(基座团队约束)

- **严格 semver**:major 才能改公共 API,minor 增能力,patch 修 bug
- **migration 只增不减**:加字段加表加索引可以,改类型/删字段需走显式工具(见 §11)
- **collection 声明 API 要稳定**,因为它会扩散到所有用户项目
- **admin meta API 是契约**,不能随意改格式

### 10.3 钉死版本

```go
require github.com/wps/itabbase v1.3.2  // 不想升就锁定
```

---

## 11. Migration 安全策略

### 11.1 自动允许(无破坏性)

- 新增 collection → 自动 `CREATE TABLE`
- 新增 field → 自动 `ALTER TABLE ADD COLUMN`
- 新增 index → 自动 `CREATE INDEX`

### 11.2 需要显式确认(破坏性)

以下变更**拒绝自动执行**,启动时打印警告:

- 删除 field
- 改字段类型(如 `string → int`)
- 重命名 field
- 删除 index

用户需手写一次性 migration 文件:`server/migrations/2026xxxx_rename_field.go`,kernel 检测到后跑一次。

### 11.3 Dev 环境例外

`GF_ENV=dev` 时可开启 `itab.WithAutoDestructive(true)` 允许自动删改(方便迭代)。**Test/prod 永远禁止**。

---

## 12. 前端调 kernel API 的体验

### 12.1 选项 A:裸 axios(简单但无类型)

```ts
const list = await axios.get('/api/kernel/objectives?filter[quarter]=Q2')
```

### 12.2 选项 B:kernel 提供 TS client 生成器(推荐)

```bash
npx itab-gen-client --out client/src/api/kernel.ts
```

读 `/api/admin/meta/collections` 自动生成:

```ts
// 自动生成,勿改
export const objectives = {
  list:   (q?: ListQuery) => http.get<Objective[]>('/api/kernel/objectives', q),
  get:    (id: string)    => http.get<Objective>(`/api/kernel/objectives/${id}`),
  create: (data: Partial<Objective>) => http.post('/api/kernel/objectives', data),
  update: (id, data) => http.patch(`/api/kernel/objectives/${id}`, data),
  delete: (id)       => http.delete(`/api/kernel/objectives/${id}`),
}
```

vibe-coding 时 AI 看到完整类型,不再瞎猜字段名。**这是 vibe-coding 杠杆放大的关键。**

---

## 13. 非目标(明确不做)

为避免滑回 NocoBase 复刻,以下能力**主动放弃**:

- ❌ 拖拽式 UI 编辑器
- ❌ schema-driven 运行时 UI 渲染(@formily 那一套)
- ❌ admin 里"配置出业务前台页面"的能力
- ❌ 多租户 / 多应用管理(每个 itab 项目独立部署)
- ❌ 插件市场 / 在线安装

> 如果用户需要这些能力,改用 NocoBase 本体。**itabbase 的定位是轻量数据驱动后端 + 朴素管理界面,不替代 NocoBase。**

---

## 14. 风险与注意点

| 风险 | 影响 | 对策 |
|---|---|---|
| go:embed 后二进制变大 | kernel 二进制 + 几 MB | admin SPA 严格控制依赖,vite 分块,目标 < 5MB gzip |
| collection API 一旦扩散就难改 | breaking change 影响所有用户 | v0.x 阶段允许小破坏,v1.0 后冻结 |
| 用户绕过 kernel 直接写 SQL | ACL 失效 | 不强制,文档建议 + 提供逃生舱 `itab.Raw()` |
| GoFrame 版本绑定 | kernel 升级可能要求新 GF | README 标注兼容矩阵 |
| 用户改了 boot/auth 跟 kernel 假设冲突 | Mount 失败 | AuthAdapter 接口足够窄,基本不会冲突 |
| admin SPA 与 scaffold 技术栈不一致 | 维护两套 | admin 用 Vue3 跟 scaffold 一致,不用 React |
| 用户已经有 task-driven 文档流程 | 多套约定打架 | kernel 文档就走 task-driven,不另起一套 |

---

## 15. 实施分期

### v0.1(MVP,2-3 周)

- Collection 声明 + 自动 CRUD(基础类型:string/int/float/bool/datetime/text)
- 基础 ACL(role × resource × action)
- admin SPA:登录 / 集合列表 / 数据浏览 / 简单筛选
- migration:只增不减
- AuthAdapter 接口 + 接入 scaffold 现有 SSO

**验收标准**:在 itab-xxxxxx 上声明一个 OKR collection,curl 能 CRUD,admin 能看见数据。

### v0.2

- 关联字段(belongs_to / has_many)
- 钩子(BeforeCreate 等)
- 自定义路由
- TS client 生成器
- admin:数据编辑 / 关联展示 / 权限配置 UI

### v1.0

- 工作流引擎(triggers + nodes)
- 审计日志
- 字段级 ACL
- 文件字段(对接 OSS)
- admin:工作流编辑 / 审计 / 用户管理

### v1.x+(按需)

- 通知中心、定时任务、外部数据源接入

---

## 16. 与同类方案对比

| 方案 | 后端语言 | 前端能力 | 与 itabbase 关系 |
|---|---|---|---|
| **NocoBase** | Node.js | 拖拽 + schema runtime | 思想来源,但 itabbase 不做 UI 配置 |
| **Directus** | Node.js | 后台管理 + 自由前台 | 最接近的参考,但 JS 栈 |
| **Strapi** | Node.js | Headless CMS + admin | 类似,但更偏内容 |
| **PocketBase** | Go | 单二进制 + admin UI | **最像 itabbase,可重点参考实现** |
| **Supabase** | PG + Deno | 数据库即 API + admin | 重 PG 特性,不适合 SQLite |

> **PocketBase 是最值得参考的实现**(同样 Go 单二进制 + go:embed admin)。建议研究其源码再动手,可少走很多弯路。

---

## 17. 待决策点(需要拍板)

| # | 决策项 | 选项 | 推荐 |
|---|---|---|---|
| 1 | kernel 仓库放哪 | 内网 GitLab / 公开 GitHub | 内网 GitLab(影响 module path) |
| 2 | kernel 包名 | itabbase / itab-core / itab-admin | itabbase |
| 3 | admin SPA 技术栈 | Vue3 / React | Vue3(与 scaffold 一致) |
| 4 | collection 声明形态 | Go struct / YAML | Go struct(类型安全,IDE 友好) |
| 5 | 第一批字段类型 | 见 §15 v0.1 列表 | 同左,关联字段 v0.2 加 |
| 6 | admin 国际化策略 | 单语言中文 / 双语 | 单语言中文(降本) |
| 7 | scaffold 现有 task 流程 | 沿用 / 另起 | 沿用 task-driven workflow |

---

## 18. 后续动作

1. 与 scaffold 维护团队对齐本方案
2. 拍板 §17 待决策项
3. 起 itabbase 仓库 + 写 v0.1 任务清单(走 task-driven)
4. 在 itab-xxxxxx 项目内做集成 demo(以 OKR 为示例业务)
5. 一两个真实业务试点,收集反馈后再 v1.0

---

## 19. 契约 vs 实现:跨栈可移植性

### 19.1 背景

NocoBase 跑在 Node.js (Koa + Toposort) 上,我们选 Go (GoFrame)。如果未来后端切到 NestJS 或其他栈,kernel 的设计能不能跟着走?**能,前提是从 day 1 就把契约和实现分开。**

### 19.2 分层模型

```
┌─────────────────────────────────────────┐
│   契约层(框架中立,长期稳定)              │
│  ・REST URL 规范                         │
│  ・admin meta API 的 JSON shape         │
│  ・ACL 模型(role × resource × action)   │
│  ・collection 声明形态(字段类型枚举)     │
│  ・migration 行为约定                    │
│  ・错误码格式(HTTP status + string code)│
└─────────────────────────────────────────┘
            ▲                    ▲
            │                    │
   ┌────────┴────────┐  ┌────────┴────────┐
   │ Go 实现          │  │ NestJS 实现      │
   │ itabbase     │  │ @itab/nestjs-   │
   │ (现在做)         │  │  kernel(以后)   │
   └─────────────────┘  └─────────────────┘
              ▲
              │
        ┌─────┴──────┐
        │ admin SPA  │  只认契约,Go / Node 后端都能用
        └────────────┘
```

### 19.3 现在该守的零成本纪律

不要为"未来可能切 NestJS"过度抽象——典型 YAGNI。但以下约束**几乎零成本**,从 day 1 就要守住,否则以后切栈时会变成大手术:

| 约束 | 说明 |
|---|---|
| 字段类型枚举值不绑 Go 概念 | `TString` / `TBelongsTo` 等要能映射到通用 ORM 类型语义,不用 GoFrame 内部类型 |
| REST URL 不带框架特有参数 | 不要出现 `?gf_xxx=` 这种 GoFrame idiom |
| ACL 模型保持纯描述 | role × resource × action 三元组,不绑 GoFrame 中间件签名 |
| admin meta API JSON 中立 | 不引用 Go struct tag、GoFrame 错误结构 |
| 错误格式用通用形式 | HTTP status + string code + message,不用 `gerror` 特有结构 |

### 19.4 演进路径

1. **v0.1 - v1.0**:专心用 Go 实现,但对外 API 按"框架无关"原则设计。**不要现在就写 OpenAPI**,太早,会被实现细节带跑。
2. **v1.0 后某个时间点**:把契约形式化为 OpenAPI 文档,放 kernel 仓库 `spec/` 下,作为单一事实来源(SSOT)。Go 实现的 API 行为以 spec 为准,反向校验。
3. **真要切 NestJS 时**:写 `@itab/nestjs-kernel`,遵循同一份 spec。**admin SPA 一行不改,因为它只看契约**。

### 19.5 关于 Koa / Toposort / NestJS 的具体澄清

| 问题 | 回答 |
|---|---|
| Koa 和 GoFrame 怎么选? | 对 kernel 设计无影响,scaffold 已选 Go,坚持即可。两者都是"路由 + 中间件"的标配能力,kernel 跟 HTTP 框架弱耦合。 |
| 要不要引入 Toposort? | NocoBase 用它做**插件依赖排序**。itabbase v0.1 - v1.0 **不需要**(只有 collection,无插件依赖图)。v1.x+ 加插件系统时再加,用 `gonum/graph/topo` 或自写 30 行,不是选型决策。 |
| 切 NestJS 时怎么处理依赖加载? | NestJS 通过 module 的 import 关系,DI 容器自动解析依赖图,**不需要** Toposort。语义等价,实现完全不同——这正是契约与实现解耦的好处。 |

### 19.6 逃生舱:何时允许打破中立性

如果某能力**只能借助 GoFrame 特性高效实现**(如特定中间件链、内置 cache 等),允许打破中立,但要:

1. 在该能力的 API 注释里标注 `// non-portable: <reason>`
2. v1.0 形式化 spec 时,这些能力**显式列为 Go-only 扩展**,不进核心契约
3. NestJS 实现可选择实现等价能力,或显式标记不支持

这条逃生舱很重要——避免"为了可移植牺牲单栈最优性"的反噬。

---

## 变更记录

| 日期 | 变更 | 说明 |
|---|---|---|
| 2026-05-09 | 初版 | 基于与 AI 助手的架构讨论沉淀 |
| 2026-05-09 | 增 §19 | 补充跨栈可移植性(契约 vs 实现分层),澄清 Koa / Toposort / NestJS 选型 |
