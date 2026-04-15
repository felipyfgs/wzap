<script setup lang="ts">
defineProps<{ state: Record<string, unknown> }>()
</script>

<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <UFormField label="Message" name="body">
    <UTextarea
      v-model="state.body"
      :rows="4"
      autoresize
      placeholder="Type your message…"
      class="w-full"
    />
  </UFormField>
  <!-- eslint-enable vue/no-mutating-props -->
</template>
