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
