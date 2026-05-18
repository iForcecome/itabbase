<template>
  <a-layout style="min-height: 100vh">
    <a-layout-sider
      v-model:collapsed="siderCollapsed"
      :width="200"
      :collapsed-width="64"
      collapsible
      theme="light"
      style="border-right: 1px solid #f0f0f0"
    >
      <div class="sider-logo" :class="{ collapsed: siderCollapsed }">
        <span class="logo-icon">⬡</span>
        <span v-if="!siderCollapsed" class="logo-text">ITabBase</span>
      </div>
      <a-menu
        mode="inline"
        :selected-keys="selectedKeys"
        @click="onMenuClick"
      >
        <a-menu-item key="/dashboard">
          <template #icon><DashboardOutlined /></template>
          <span>仪表盘</span>
        </a-menu-item>
        <a-menu-item key="/datasource">
          <template #icon><DatabaseOutlined /></template>
          <span>数据源</span>
        </a-menu-item>
        <a-menu-item key="/access">
          <template #icon><TeamOutlined /></template>
          <span>用户和权限</span>
        </a-menu-item>
        <a-menu-item key="/settings">
          <template #icon><SettingOutlined /></template>
          <span>系统设置</span>
        </a-menu-item>
      </a-menu>
    </a-layout-sider>

    <a-layout>
      <a-layout-header class="app-header">
        <div style="flex: 1" />
        <a-dropdown>
          <a-space style="cursor: pointer; color: rgba(0, 0, 0, 0.65)">
            <a-avatar size="small" style="background-color: #1677ff">
              <template #icon><UserOutlined /></template>
            </a-avatar>
            <span>{{ user?.name }}</span>
          </a-space>
          <template #overlay>
            <a-menu>
              <a-menu-item key="logout" @click="handleLogout">
                <LogoutOutlined /> 登出
              </a-menu-item>
            </a-menu>
          </template>
        </a-dropdown>
      </a-layout-header>

      <a-layout-content class="app-content">
        <RouterView />
      </a-layout-content>
    </a-layout>
  </a-layout>
</template>

<script setup lang="ts">
import { useUserStore } from "@/stores/user";

const route = useRoute();
const router = useRouter();
const userStore = useUserStore();

const user = computed(() => userStore.user);
const siderCollapsed = ref(false);
const selectedKeys = computed(() => {
  const path = route.path;
  if (path.startsWith("/datasource")) return ["/datasource"];
  if (path.startsWith("/access")) return ["/access"];
  if (path.startsWith("/settings")) return ["/settings"];
  return ["/dashboard"];
});

function onMenuClick(info: { key: string | number }) {
  router.push(String(info.key));
}

async function handleLogout() {
  await userStore.logout();
  router.replace("/login");
}
</script>

<style scoped>
.sider-logo {
  height: 48px;
  display: flex;
  align-items: center;
  padding: 0 20px;
  border-bottom: 1px solid #f0f0f0;
  gap: 10px;
  overflow: hidden;
  white-space: nowrap;
}
.sider-logo.collapsed {
  justify-content: center;
  padding: 0;
}
.logo-icon {
  font-size: 22px;
  color: #1677ff;
  flex-shrink: 0;
}
.logo-text {
  font-weight: 700;
  font-size: 16px;
  color: rgba(0, 0, 0, 0.85);
}
.app-header {
  display: flex;
  align-items: center;
  padding: 0 24px;
  background: #fff;
  border-bottom: 1px solid #f0f0f0;
  height: 48px;
  line-height: 48px;
}
.app-content {
  padding: 24px;
  overflow: auto;
  background: #f5f5f5;
  min-height: 0;
}
</style>
