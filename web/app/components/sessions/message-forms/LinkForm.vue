<script setup lang="ts">
defineProps<{ state: Record<string, unknown> }>()
</script>

<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <UFormField label="URL" name="url">
    <UInput v-model="state.url" placeholder="https://example.com" class="w-full" />
  </UFormField>
  <UFormField label="Title" name="title">
    <UInput v-model="state.title" placeholder="Link title" class="w-full" />
  </UFormField>
  <UFormField label="Description" name="description">
    <UInput v-model="state.description" placeholder="Link description" class="w-full" />
  </UFormField>
  <!-- eslint-enable vue/no-mutating-props -->
</template>
