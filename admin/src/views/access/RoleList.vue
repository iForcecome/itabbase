<template>
  <div>
    <a-table
      :columns="columns"
      :data-source="roles"
      :loading="loading"
      :pagination="false"
      row-key="id"
    >
      <template #bodyCell="{ column }">
        <template v-if="column.key === 'actions'">
          <a-button size="small" disabled>配置权限</a-button>
        </template>
      </template>
    </a-table>
    <a-typography-text type="secondary" style="display: block; margin-top: 12px; font-size: 12px">
      权限矩阵配置将在后续版本中支持
    </a-typography-text>
  </div>
</template>

<script setup lang="ts">
import { api } from "@/api";
import { message } from "ant-design-vue";

const loading = ref(false);
const roles = ref<any[]>([]);

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 60 },
  { title: "角色名", dataIndex: "name", key: "name" },
  { title: "显示名", dataIndex: "display", key: "display" },
  { title: "操作", key: "actions", width: 120 },
];

async function loadRoles() {
  loading.value = true;
  try {
    const res = await api.list("roles", 1, 100);
    roles.value = res.data;
  } catch { message.error("加载角色列表失败"); }
  finally { loading.value = false; }
}

onMounted(loadRoles);
</script>
