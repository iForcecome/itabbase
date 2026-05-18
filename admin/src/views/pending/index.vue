<template>
  <div class="pending-screen">
    <a-result
      status="info"
      title="申请已提交"
      sub-title="你的访问申请正在等待管理员审批。审批通过后再次登录即可进入系统。如需加急，请联系管理员。"
    >
      <template #extra>
        <a-button type="primary" @click="dismissPending">返回登录</a-button>
      </template>
    </a-result>
  </div>
</template>

<script setup lang="ts">
const router = useRouter();

function dismissPending() {
  router.replace("/login");
}
</script>

<style scoped>
.pending-screen {
  display: flex;
  align-items: center;
  justify-content: center;
  min-height: 100vh;
  background: #f0f2f5;
}
</style>
