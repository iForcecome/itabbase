<template>
  <div>
    <a-tabs v-model:activeKey="activeTab" type="card">
      <a-tab-pane key="general" tab="基本设置">
        <a-card :loading="loading" :bordered="false">
          <a-form :model="form" layout="vertical" style="max-width: 480px">
            <a-form-item label="新用户审批">
              <a-switch
                v-model:checked="form.require_approval"
                checked-children="开"
                un-checked-children="关"
              />
              <div style="margin-top: 4px; color: #999; font-size: 12px">
                开启后，新注册/SSO 首次登录的用户需要管理员审批后才能使用
              </div>
            </a-form-item>
            <a-form-item>
              <a-button type="primary" @click="handleSave" :loading="saving">保存</a-button>
            </a-form-item>
          </a-form>
        </a-card>
      </a-tab-pane>
      <a-tab-pane key="about" tab="关于">
        <a-card :bordered="false">
          <a-descriptions :column="1" bordered size="small">
            <a-descriptions-item label="产品名称">ITabBase</a-descriptions-item>
            <a-descriptions-item label="版本">v0.1.0</a-descriptions-item>
            <a-descriptions-item label="内核">itab kernel</a-descriptions-item>
            <a-descriptions-item label="前端框架">Vue 3 + Ant Design Vue</a-descriptions-item>
            <a-descriptions-item label="后端框架">GoFrame v2</a-descriptions-item>
          </a-descriptions>
        </a-card>
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { api } from "@/api";
import { message } from "ant-design-vue";

const activeTab = ref("general");
const loading = ref(false);
const saving = ref(false);
const form = reactive({ require_approval: true });

async function loadSettings() {
  loading.value = true;
  try {
    const res = await api.list("system_settings", 1, 100);
    for (const row of res.data) {
      if (row.key === "require_approval") form.require_approval = row.value === "true";
    }
  } catch { /* ignore */ }
  finally { loading.value = false; }
}

async function handleSave() {
  saving.value = true;
  try {
    const res = await api.list("system_settings", 1, 100, "filter[key]=require_approval");
    if (res.data.length > 0) {
      await api.update("system_settings", res.data[0].id as number, {
        value: form.require_approval ? "true" : "false",
      });
    }
    message.success("设置已保存");
  } catch { message.error("保存失败"); }
  finally { saving.value = false; }
}

onMounted(loadSettings);
</script>
