<script setup lang="ts">
defineProps<{
  state: Record<string, unknown>
  msgType: string
  mediaFileName: string
  mediaBase64: string
  mediaMimeType: string
}>()

const emit = defineEmits<{
  fileSelect: [event: Event]
}>()

const acceptMap: Record<string, string> = {
  image: 'image/*',
  video: 'video/*',
  audio: 'audio/*',
  document: '*/*'
}
</script>

<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <UFormField label="File">
    <div class="flex items-center gap-2 w-full">
      <input
        type="file"
        class="block w-full text-sm text-muted
          file:mr-2 file:py-1.5 file:px-3 file:rounded-md
          file:border-0 file:text-sm file:font-medium
          file:bg-elevated file:text-highlighted
          hover:file:bg-accentened file:cursor-pointer"
        :accept="acceptMap[msgType] || '*/*'"
        @change="emit('fileSelect', $event)"
      >
    </div>
    <template v-if="mediaFileName" #hint>
      <span class="text-xs text-muted">{{ mediaFileName }}</span>
    </template>
  </UFormField>
  <USeparator label="or" />
  <UFormField label="URL" name="url">
    <UInput v-model="state.url" placeholder="https://example.com/file.jpg" class="w-full" />
  </UFormField>
  <UFormField label="Caption" name="caption">
    <UTextarea
      v-model="state.caption"
      :rows="2"
      autoresize
      placeholder="Caption (optional)"
      class="w-full"
    />
  </UFormField>
  <template v-if="msgType === 'document'">
    <UFormField label="File Name" name="fileName">
      <UInput v-model="state.fileName" placeholder="document.pdf" class="w-full" />
    </UFormField>
  </template>
  <!-- eslint-enable vue/no-mutating-props -->
</template>
