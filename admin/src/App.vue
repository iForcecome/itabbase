<template>
  <a-config-provider :theme="antTheme">
    <RouterView />
  </a-config-provider>
</template>

<script setup lang="ts">
const antTheme = {
  token: {
    colorPrimary: "#1677ff",
    borderRadius: 6,
  },
};
</script>
