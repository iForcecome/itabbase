<template>
  <CollectionView :name="name" />
</template>

<script setup lang="ts">
import CollectionView from "@/views/CollectionView.vue";

defineProps<{ name: string }>();
</script>
