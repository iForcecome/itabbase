<template>
  <div class="p-8">
    <h1 class="text-2xl font-semibold">{{ title }}</h1>
    <p class="mt-2 text-gray-500">欢迎使用管理后台</p>
  </div>
</template>

<script setup lang="ts">
const title = import.meta.env.VITE_APP_TITLE || "Admin";
</script>
