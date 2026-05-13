---
name: itab-vueframe
description: Scaffold a Vue 3 + TypeScript + Vite project with full infrastructure. Use when the user wants to create or initialize a Vue frontend project, set up a Vue scaffold, or mentions Vue project initialization, 初始化前端, 搭建Vue项目.
---

# 初始化 Vue 前端项目骨架

通用 Vue 3 + TypeScript + Vite 项目骨架，只搭建基础设施，不含任何业务逻辑和布局假设。

## 概览

| 项目     | 说明                                                                                      |
| -------- | ----------------------------------------------------------------------------------------- |
| 产出物   | 路由 + 状态管理 + Axios 封装 + 登录/首页/错误页/demo导航 + ESLint/Prettier 全套规范 |
| 关键设计 | 鉴权 opt-in（默认不拦截），布局无预设                                                     |
| 执行结果 | `pnpm dev` 启动无报错                                                                     |

## 初始化规则

1. 输出目录由用户指定，未指定时默认 `client/`
2. 若目录已存在：保留已有文件，只创建缺失的文件，**不覆盖**
3. 若不存在：创建目录并生成全部文件
4. **前置检查**：Node.js 18+，优先 pnpm（未安装则 `npm i -g pnpm`）
5. 全部 `.vue` 文件使用 `<script setup lang="ts">`
6. **错误处理**：安装或启动失败时输出错误信息并尝试修复，不静默跳过

## 目录结构

```
client/
├── src/
│   ├── api/index.ts
│   ├── assets/
│   ├── components/{common,business}/
│   ├── composables/
│   ├── layouts/DefaultLayout.vue
│   ├── views/{home,login,demo,error}/
│   ├── router/{index.ts,routes.ts}
│   ├── stores/{index.ts,user.ts}
│   ├── utils/request.ts
│   ├── types/{env.d.ts,router.d.ts,api.d.ts}
│   ├── styles/main.css
│   ├── App.vue
│   └── main.ts
├── index.html
├── vite.config.ts
├── tsconfig.json / tsconfig.app.json / tsconfig.node.json
├── eslint.config.js
├── .prettierrc
├── .env / .env.development / .env.production
├── .gitignore
└── package.json
```

## 依赖

**dependencies**: `vue` `vue-router` `pinia` `axios` `nprogress` `@vueuse/core` `@lucide/vue`

**devDependencies**: `typescript` `vite` `@vitejs/plugin-vue` `@tailwindcss/vite` `unplugin-vue-components` `unplugin-auto-import` `vue-tsc` `eslint` `@eslint/js` `typescript-eslint` `eslint-plugin-vue` `eslint-config-prettier` `prettier` `@types/nprogress`

## 核心文件内容

### vite.config.ts

```ts
import { defineConfig } from "vite";
import vue from "@vitejs/plugin-vue";
import tailwindcss from "@tailwindcss/vite";
import Components from "unplugin-vue-components/vite";
import AutoImport from "unplugin-auto-import/vite";
import { resolve } from "path";

export default defineConfig({
  plugins: [
    vue(),
    tailwindcss(),
    Components({ dts: "src/types/components.d.ts" }),
    AutoImport({
      imports: ["vue", "vue-router", "pinia", "@vueuse/core"],
      dts: "src/types/auto-imports.d.ts",
      eslintrc: { enabled: true },
    }),
  ],
  resolve: { alias: { "@": resolve(__dirname, "src") } },
  build: {
    outDir: "dist",
    rollupOptions: {
      output: {
        manualChunks: {
          vue: ["vue", "vue-router", "pinia"],
        },
      },
    },
  },
});
```

### main.ts

```ts
import { createApp } from "vue";
import { createPinia } from "pinia";
import router from "./router";
import App from "./App.vue";
import "./styles/main.css";
import "nprogress/nprogress.css";

const app = createApp(App);
app.use(createPinia());
app.use(router);
app.mount("#app");
```

### router/routes.ts

骨架阶段所有路由**不设置** `requiresAuth`，默认全部可自由访问。

```ts
import type { RouteRecordRaw } from "vue-router";
import DefaultLayout from "@/layouts/DefaultLayout.vue";

const routes: RouteRecordRaw[] = [
  {
    path: "/login",
    name: "Login",
    component: () => import("@/views/login/index.vue"),
    meta: { title: "登录" },
  },
  {
    path: "/",
    component: DefaultLayout,
    children: [
      {
        path: "",
        name: "Home",
        component: () => import("@/views/home/index.vue"),
        meta: { title: "首页" },
      },
      {
        path: "demo",
        name: "Demo",
        component: () => import("@/views/demo/index.vue"),
        meta: { title: "示例" },
      },
    ],
  },
  {
    path: "/403",
    name: "Forbidden",
    component: () => import("@/views/error/403.vue"),
    meta: { title: "无权限" },
  },
  {
    path: "/:pathMatch(.*)*",
    name: "NotFound",
    component: () => import("@/views/error/404.vue"),
    meta: { title: "页面不存在" },
  },
];

export default routes;
```

### router/index.ts

鉴权为 **opt-in**：骨架不假设需要登录，业务方按需给路由加 `meta: { requiresAuth: true }`。

```ts
import { createRouter, createWebHistory } from "vue-router";
import NProgress from "nprogress";
import routes from "./routes";
import { useUserStore } from "@/stores/user";

NProgress.configure({ showSpinner: false });

const router = createRouter({ history: createWebHistory(), routes });

router.beforeEach((to, _from, next) => {
  NProgress.start();
  const userStore = useUserStore();
  if (to.meta.requiresAuth && !userStore.token) {
    next({ path: "/login", query: { redirect: to.fullPath } });
  } else if (userStore.token && to.path === "/login") {
    next("/");
  } else {
    next();
  }
});

router.afterEach((to) => {
  NProgress.done();
  if (to.meta.title)
    document.title = `${to.meta.title} - ${import.meta.env.VITE_APP_TITLE}`;
});

export default router;
```

### stores/user.ts

`stores/index.ts` 仅 barrel 导出：`export { useUserStore } from './user'`

```ts
import { defineStore } from "pinia";
import { ref } from "vue";
import router from "@/router";

export const useUserStore = defineStore("user", () => {
  const token = ref(localStorage.getItem("token") || "");
  const userInfo = ref<Record<string, unknown> | null>(null);

  function setToken(val: string) {
    token.value = val;
    localStorage.setItem("token", val);
  }
  function clearAuth() {
    token.value = "";
    userInfo.value = null;
    localStorage.removeItem("token");
  }
  function logout() {
    clearAuth();
    router.push("/login");
  }
  return { token, userInfo, setToken, clearAuth, logout };
});
```

### utils/request.ts

```ts
import axios from "axios";
import type { AxiosRequestConfig } from "axios";
import { useUserStore } from "@/stores/user";
import router from "@/router";

const request = axios.create({
  baseURL: import.meta.env.VITE_API_BASE_URL,
  timeout: 10000,
});

request.interceptors.request.use((config) => {
  const { token } = useUserStore();
  if (token) config.headers.Authorization = `Bearer ${token}`;
  return config;
});

request.interceptors.response.use(
  (res) => res.data,
  (error) => {
    if (error.response?.status === 401) {
      useUserStore().clearAuth();
      router.push("/login");
    }
    return Promise.reject(error);
  },
);

export default request;

export const get = <T>(url: string, config?: AxiosRequestConfig) =>
  request.get<unknown, T>(url, config);
export const post = <T>(
  url: string,
  data?: unknown,
  config?: AxiosRequestConfig,
) => request.post<unknown, T>(url, data, config);
export const put = <T>(
  url: string,
  data?: unknown,
  config?: AxiosRequestConfig,
) => request.put<unknown, T>(url, data, config);
export const del = <T>(url: string, config?: AxiosRequestConfig) =>
  request.delete<unknown, T>(url, config);
```

### 类型定义

**types/env.d.ts**：

```ts
/// <reference types="vite/client" />
interface ImportMetaEnv {
  readonly VITE_APP_TITLE: string;
  readonly VITE_API_BASE_URL: string;
}
interface ImportMeta {
  readonly env: ImportMetaEnv;
}
```

**types/router.d.ts**：

```ts
import "vue-router";
declare module "vue-router" {
  interface RouteMeta {
    title?: string;
    requiresAuth?: boolean;
  }
}
```

**types/api.d.ts**：

```ts
export interface ApiResponse<T = unknown> {
  code: number;
  message: string;
  data: T;
}
export interface PaginatedData<T> {
  list: T[];
  total: number;
  page: number;
  pageSize: number;
}
export type PaginatedResponse<T> = ApiResponse<PaginatedData<T>>;
```

### views/error/404.vue

```vue
<template>
  <div class="min-h-screen flex flex-col items-center justify-center">
    <h1 class="text-8xl font-bold text-gray-300">404</h1>
    <p class="mt-4 text-xl text-gray-500">页面不存在</p>
    <button
      @click="router.push('/')"
      class="mt-8 px-6 py-2 text-sm text-blue-600 hover:text-blue-800 border border-blue-600 rounded-md hover:bg-blue-50"
    >
      返回首页
    </button>
  </div>
</template>

<script setup lang="ts">
const router = useRouter();
</script>
```

### views/error/403.vue

```vue
<script setup lang="ts">
const router = useRouter();
</script>

<template>
  <div class="min-h-screen flex flex-col items-center justify-center">
    <h1 class="text-8xl font-bold text-gray-300">403</h1>
    <p class="mt-4 text-xl text-gray-500">暂无权限访问此页面</p>
    <button
      @click="router.push('/')"
      class="mt-8 px-6 py-2 text-sm text-blue-600 hover:text-blue-800 border border-blue-600 rounded-md hover:bg-blue-50"
    >
      返回首页
    </button>
  </div>
</template>
```

### views/demo/index.vue

用于验证路由和页面跳转是否正常。

```vue
<template>
  <div class="p-8 max-w-md mx-auto">
    <h1 class="text-2xl font-semibold mb-6">路由跳转示例</h1>
    <ul class="space-y-3">
      <li v-for="link in links" :key="link.path">
        <button
          @click="router.push(link.path)"
          class="w-full text-left px-4 py-2 rounded-md border hover:bg-gray-50"
        >
          {{ link.label }}
          <span class="text-gray-400 text-sm ml-2">{{ link.path }}</span>
        </button>
      </li>
    </ul>
  </div>
</template>

<script setup lang="ts">
const router = useRouter();

const links = [
  { label: "首页", path: "/" },
  { label: "登录页", path: "/login" },
  { label: "403 页", path: "/403" },
  { label: "404 页（随机路径）", path: "/this-page-does-not-exist" },
];
</script>

```

### 其他简单文件

- **App.vue**：仅 `<RouterView />`
- **layouts/DefaultLayout.vue**：`<div class="min-h-screen"><RouterView /></div>`，业务方修改此文件定义具体布局
- **styles/main.css**：`@import "tailwindcss";`
- **views/home/index.vue**：显示 `VITE_APP_TITLE`，占位即可
- **views/login/index.vue**：占位页面，具体 UI 和表单逻辑由业务实现
- **api/index.ts**：空文件，按业务模块拆分后在此 barrel 导出

### 配置文件

**tsconfig.json**：project references，引用 `tsconfig.app.json` + `tsconfig.node.json`。

**tsconfig.app.json**：

```json
{
  "compilerOptions": {
    "target": "ES2020",
    "module": "ESNext",
    "lib": ["ES2020", "DOM", "DOM.Iterable"],
    "strict": true,
    "skipLibCheck": true,
    "moduleResolution": "bundler",
    "allowImportingTsExtensions": true,
    "isolatedModules": true,
    "moduleDetection": "force",
    "noEmit": true,
    "jsx": "preserve",
    "noUnusedLocals": true,
    "noUnusedParameters": true,
    "paths": { "@/*": ["./src/*"] },
    "types": ["vite/client"]
  },
  "include": [
    "src/**/*.ts",
    "src/**/*.vue",
    "src/types/auto-imports.d.ts",
    "src/types/components.d.ts"
  ]
}
```

**tsconfig.node.json**：target ES2022，module ESNext，bundler，strict，include `["vite.config.ts"]`。

**eslint.config.js**：

```js
import js from "@eslint/js";
import tseslint from "typescript-eslint";
import pluginVue from "eslint-plugin-vue";
import prettier from "eslint-config-prettier";
import autoImportGlobals from "./.eslintrc-auto-import.json" with { type: "json" };

export default [
  {
    ignores: [
      "dist/",
      "src/types/auto-imports.d.ts",
      "src/types/components.d.ts",
    ],
  },
  js.configs.recommended,
  ...tseslint.configs.recommended,
  ...pluginVue.configs["flat/recommended"],
  {
    languageOptions: {
      globals: autoImportGlobals.globals,
    },
  },
  {
    files: ["**/*.vue"],
    languageOptions: { parserOptions: { parser: tseslint.parser } },
  },
  prettier,
];
```

**.prettierrc**：`{ "semi": false, "singleQuote": true, "trailingComma": "all", "printWidth": 100 }`

**.env**：`VITE_APP_TITLE=My App` / **.env.development** + **.env.production**：`VITE_API_BASE_URL=/api`

**index.html**：标准 Vite HTML，lang `zh-CN`，`<div id="app">`，`<script type="module" src="/src/main.ts">`

**package.json scripts**：`{ "dev": "vite", "build": "vue-tsc -b && vite build", "preview": "vite preview", "lint": "eslint . --fix", "format": "prettier --write \"src/**/*.{vue,ts,js,css,md}\"" }`

**.gitignore**：`node_modules/`、`dist/`、`*.local`、`src/types/auto-imports.d.ts`、`src/types/components.d.ts`、`.eslintrc-auto-import.json`

## 执行步骤

1. 检查 Node.js ≥ 18，确认 pnpm 可用
2. 创建所有目录和文件（遵守不覆盖规则）
3. `pnpm install`（失败则检查网络和包名拼写）
4. `pnpm dev` → 确认终端输出 `Local: http://localhost:xxxx/` 且无报错
