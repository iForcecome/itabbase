<template>
  <div>
    <a-tabs v-model:activeKey="activeTab" type="card" @change="onTabChange">
      <a-tab-pane key="__manage" tab="集合配置">
        <CollectionManager />
      </a-tab-pane>
      <a-tab-pane
        v-for="c in dynamicCollections"
        :key="c.name"
        :tab="c.display || c.name"
      >
        <CollectionData :name="c.name" />
      </a-tab-pane>
    </a-tabs>
  </div>
</template>

<script setup lang="ts">
import { useUserStore } from "@/stores/user";
import CollectionManager from "./CollectionManager.vue";
import CollectionData from "./CollectionData.vue";

const userStore = useUserStore();
const dynamicCollections = computed(() => userStore.dynamicCollections);

const activeTab = ref("__manage");

function onTabChange(key: string | number) {
  activeTab.value = String(key);
}

watch(dynamicCollections, (cols) => {
  if (activeTab.value !== "__manage" && !cols.some((c) => c.name === activeTab.value)) {
    activeTab.value = "__manage";
  }
});
</script>
