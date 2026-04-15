<script setup lang="ts">
defineProps<{ state: Record<string, unknown> }>()
</script>

<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <UFormField label="Contact Name" name="name">
    <UInput v-model="state.name" placeholder="John Doe" class="w-full" />
  </UFormField>
  <UFormField label="vCard" name="vcard">
    <UTextarea
      v-model="state.vcard"
      :rows="6"
      autoresize
      placeholder="BEGIN:VCARD&#10;VERSION:3.0&#10;FN:John Doe&#10;TEL:+5511999999999&#10;END:VCARD"
      class="w-full"
    />
  </UFormField>
  <!-- eslint-enable vue/no-mutating-props -->
</template>
