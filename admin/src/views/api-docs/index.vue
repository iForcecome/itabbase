<template>
  <div class="api-docs-wrapper">
    <div ref="containerRef" class="swagger-container" />
  </div>
</template>

<script setup lang="ts">
import SwaggerUIBundle from "swagger-ui-dist/swagger-ui-bundle.js";
import "swagger-ui-dist/swagger-ui.css";

const containerRef = ref<HTMLElement>();

onMounted(() => {
  if (!containerRef.value) return;
  SwaggerUIBundle({
    domNode: containerRef.value,
    url: "/api/docs.json",
    deepLinking: true,
    presets: [SwaggerUIBundle.presets.apis, SwaggerUIBundle.SwaggerUIStandalonePreset],
    layout: "BaseLayout",
    withCredentials: true,
  });
});
</script>

<style scoped>
.api-docs-wrapper {
  height: 100%;
  background: #fff;
  border-radius: 8px;
  overflow: auto;
}

.swagger-container {
  min-height: 100%;
}

/* 修正嵌入时 topbar 显示问题 */
:deep(.swagger-ui .topbar) {
  display: none;
}

:deep(.swagger-ui) {
  font-family: inherit;
}
</style>
