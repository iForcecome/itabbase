---
name: itab-client
description: >-
  it-ai-base `client/`：Vue + Vite（Vue Router、Pinia、与 Vue 兼容的 UI 库）。与 `server/` 仅经 HTTP(S) 或约定通道；
  鉴权、业务规则与持久化以后端为准；接口与权限配合 `.cursor/skills/itab-server/SKILL.md`。本地开发 8081。
---

# 前端（`client/`）

只改 **`client/`**；与 **`server/`** 仅经 HTTP(S) 或约定通道。鉴权、业务规则、持久化**以后端为准**；接口与权限以服务端技能为准。

## 技术栈

**Vue** 及常见配套：**Vite**、**Vue Router**、**Pinia**、与 Vue 兼容的 UI 组件库。不以 React/Svelte/Angular 作为本仓库面向用户应用的主框架。

## 路径前缀（`base`）

`client/vite.config.ts` 中 **必须** 保留 `base` 配置（当前值见文件实际内容）。该值决定：

1. **静态资源路径**：打包后的 JS/CSS/图片都挂在此前缀下。
2. **`import.meta.env.BASE_URL`**：前端代码通过此变量拼接 API 请求路径，例如：
   ```ts
   const basePath = `${import.meta.env.BASE_URL}`.replace(/\/+$/, "");
   // 请求示例：`${basePath}/api/ledger`
   ```
3. **Vite proxy**：`server.proxy` 的 key 必须以同一前缀开头（如 `"<base>api"`），确保本地开发请求能正确转发到后端。

**禁止**在代码中硬编码路径前缀；**统一**通过 `import.meta.env.BASE_URL` 获取。

## 端口与联调

- **前端开发服务**：**`8081`**（`client/vite.config.ts` 的 `server.port`，`strictPort: true`）。
- **后端**：**`8080`**（见 `.cursor/skills/itab-server/SKILL.md`）。

若在 Vite 中为销售易相关 API 配置 `proxy`，**勿**将整条路径 **`/crm`** 代理到后端（易与 SPA 前端路由冲突）；仅代理后端实际注册的 API 路径（例如历史上可能单独代理 `/crm/oauth`、`/crm/xoql` 等，**以当前 `server/` 路由为准**）。

## 参考索引

| 主题 | 文档 |
|------|------|
| 文件上传 / 下载（业务表只存 `file_object.id`；小文件 multipart、大文件 presign 三段式；下载 302 预签） | 前后端一体技能：[../itab-file-object-storage/SKILL.md](../itab-file-object-storage/SKILL.md)；前端专项：[../itab-file-object-storage/references/frontend-integration.md](../itab-file-object-storage/references/frontend-integration.md) |
| SSO 前端接入（KSO 域名 / Cookie / axios 拦截） | 前后端一体技能：[../itab-kso-sso/SKILL.md](../itab-kso-sso/SKILL.md)；前端专项：[../itab-kso-sso/references/client/integration-guide.md](../itab-kso-sso/references/client/integration-guide.md) |
| 销售易 CRM 授权绑定（前后端一体） | [../itab-xiaoshouyi-crm/SKILL.md](../itab-xiaoshouyi-crm/SKILL.md)；前端专项：[../itab-xiaoshouyi-crm/references/client/integration-guide.md](../itab-xiaoshouyi-crm/references/client/integration-guide.md) |

新增前端主题：本表增一行，若涉及跨前后端契约，在对应的服务端技能 `references/` 下建目录，并在此处回链。
