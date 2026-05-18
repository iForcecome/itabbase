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
