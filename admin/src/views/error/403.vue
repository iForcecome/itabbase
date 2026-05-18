<script setup lang="ts">
import { useUserStore } from "@/stores/user";

const router = useRouter();
const userStore = useUserStore();

async function handleLogout() {
  await userStore.logout();
  router.replace("/login");
}
</script>

<template>
  <div style="min-height: 100vh; display: flex; flex-direction: column; align-items: center; justify-content: center; background: #f5f5f5">
    <a-result
      status="403"
      title="无权访问"
      sub-title="管理后台仅限管理员登录，请联系管理员授权"
    >
      <template #extra>
        <a-space>
          <a-button @click="handleLogout">切换账号</a-button>
          <a-button type="primary" @click="router.push('/login')">返回登录</a-button>
        </a-space>
      </template>
    </a-result>
  </div>
</template>
