<template>
  <div>
    <a-tabs v-model:activeKey="activeTab" type="card">
      <a-tab-pane key="users" tab="用户管理">
        <UserList />
      </a-tab-pane>
      <a-tab-pane key="roles" tab="角色权限">
        <RoleList />
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import UserList from "./UserList.vue";
import RoleList from "./RoleList.vue";

const activeTab = ref("users");
</script>
