<script setup lang="ts">
defineProps<{
  mediaFileName: string
}>()

const emit = defineEmits<{
  fileSelect: [event: Event]
}>()
</script>

<template>
  <UFormField label="Sticker File">
    <input
      type="file"
      class="block w-full text-sm text-muted
        file:mr-2 file:py-1.5 file:px-3 file:rounded-md
        file:border-0 file:text-sm file:font-medium
        file:bg-elevated file:text-highlighted
        hover:file:bg-accentened file:cursor-pointer"
      accept="image/webp,image/png"
      @change="emit('fileSelect', $event)"
    >
    <template v-if="mediaFileName" #hint>
      <span class="text-xs text-muted">{{ mediaFileName }}</span>
    </template>
  </UFormField>
</template>
