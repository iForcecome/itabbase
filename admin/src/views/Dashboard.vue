<template>
  <div>
    <a-page-header title="仪表盘" :ghost="false" style="margin-bottom: 16px" />

    <a-row :gutter="16">
      <a-col :span="6">
        <a-card>
          <a-statistic title="动态集合" :value="dynamicCollections.length">
            <template #prefix><DatabaseOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="总集合数" :value="allCollections.length">
            <template #prefix><AppstoreOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="内置集合" :value="builtinCount">
            <template #prefix><LockOutlined /></template>
          </a-statistic>
        </a-card>
      </a-col>
      <a-col :span="6">
        <a-card>
          <a-statistic title="系统状态" value="运行中">
            <template #prefix><CheckCircleOutlined style="color: #52c41a" /></template>
          </a-statistic>
        </a-card>
      </a-col>
    </a-row>

    <a-card title="快捷入口" style="margin-top: 16px">
      <a-space wrap>
        <a-button
          v-for="c in dynamicCollections"
          :key="c.name"
          @click="$router.push('/datasource')"
        >
          <TableOutlined /> {{ c.display || c.name }}
        </a-button>
        <a-button type="dashed" @click="$router.push('/datasource')">
          <PlusOutlined /> 新建集合
        </a-button>
      </a-space>
      <a-empty
        v-if="dynamicCollections.length === 0"
        description="暂无动态集合，去数据源中创建"
      >
        <a-button type="primary" @click="$router.push('/datasource')">
          前往数据源
        </a-button>
      </a-empty>
    </a-card>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from "@/stores/user";

const userStore = useUserStore();
const dynamicCollections = computed(() => userStore.dynamicCollections);
const allCollections = computed(() => userStore.collections);
const builtinCount = computed(
  () => allCollections.value.filter((c) => c.source === "builtin").length,
);
</script>
