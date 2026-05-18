<template>
  <div>
    <a-page-header title="用户管理" sub-title="管理系统用户和审批" :ghost="false" style="margin-bottom: 16px" />

    <a-card>
      <a-table
        :columns="columns"
        :data-source="users"
        :loading="loading"
        :pagination="pagination"
        row-key="id"
        @change="onTableChange"
      >
        <template #bodyCell="{ column, record }">
          <template v-if="column.key === 'status'">
            <a-tag :color="statusColor(record.status)">{{ statusLabel(record.status) }}</a-tag>
          </template>
          <template v-else-if="column.key === 'actions'">
            <a-space>
              <a-button
                v-if="record.status === 'pending'"
                size="small"
                type="primary"
                @click="handleApprove(record.id)"
              >
                批准
              </a-button>
              <a-button
                v-if="record.status === 'pending'"
                size="small"
                danger
                @click="handleReject(record.id)"
              >
                拒绝
              </a-button>
            </a-space>
          </template>
        </template>
      </a-table>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { api } from "@/api";
import { message } from "ant-design-vue";

const loading = ref(false);
const users = ref<any[]>([]);
const pagination = reactive({ current: 1, pageSize: 20, total: 0 });

const columns = [
  { title: "ID", dataIndex: "id", key: "id", width: 60 },
  { title: "用户名", dataIndex: "username", key: "username" },
  { title: "显示名", dataIndex: "display_name", key: "display_name" },
  { title: "状态", key: "status", width: 100 },
  { title: "首次登录", dataIndex: "first_seen_at", key: "first_seen_at" },
  { title: "操作", key: "actions", width: 180 },
];

async function loadUsers() {
  loading.value = true;
  try {
    const res = await api.list("users", pagination.current, pagination.pageSize);
    users.value = res.data;
    pagination.total = res.total;
  } catch (err) {
    message.error("加载用户列表失败");
  } finally {
    loading.value = false;
  }
}

function onTableChange(pag: any) {
  pagination.current = pag.current;
  pagination.pageSize = pag.pageSize;
  loadUsers();
}

async function handleApprove(id: number) {
  try {
    await api.update("users", id, { status: "active" });
    message.success("已批准");
    loadUsers();
  } catch (err) {
    message.error("操作失败");
  }
}

async function handleReject(id: number) {
  try {
    await api.update("users", id, { status: "rejected" });
    message.success("已拒绝");
    loadUsers();
  } catch (err) {
    message.error("操作失败");
  }
}

function statusColor(s: string) {
  if (s === "active") return "green";
  if (s === "pending") return "orange";
  if (s === "rejected") return "red";
  return "default";
}

function statusLabel(s: string) {
  if (s === "active") return "活跃";
  if (s === "pending") return "待审批";
  if (s === "rejected") return "已拒绝";
  return s;
}

onMounted(loadUsers);
</script>
