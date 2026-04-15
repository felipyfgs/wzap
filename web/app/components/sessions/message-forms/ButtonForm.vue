<script setup lang="ts">
import type { ButtonItem } from '~/composables/useMessageSender'

defineProps<{
  state: Record<string, unknown>
  buttons: ButtonItem[]
}>()

const emit = defineEmits<{
  addButton: []
  removeButton: [index: number]
}>()
</script>

<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <UFormField label="Body" name="body">
    <UTextarea
      v-model="state.body"
      :rows="3"
      autoresize
      placeholder="Message body"
      class="w-full"
    />
  </UFormField>
  <UFormField label="Footer" name="footer">
    <UInput v-model="state.footer" placeholder="Footer text (optional)" class="w-full" />
  </UFormField>
  <div class="space-y-2">
    <p class="text-sm font-medium text-highlighted">
      Buttons
    </p>
    <div v-for="(btn, i) in buttons" :key="i" class="flex items-center gap-2">
      <UInput v-model="btn.id" placeholder="ID" class="w-24" />
      <UInput v-model="btn.text" placeholder="Button text" class="flex-1" />
      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="sm"
        :disabled="buttons.length <= 1"
        @click="emit('removeButton', i)"
      />
    </div>
    <UButton
      label="Add Button"
      icon="i-lucide-plus"
      color="neutral"
      variant="subtle"
      size="sm"
      @click="emit('addButton')"
    />
  </div>
  <!-- eslint-enable vue/no-mutating-props -->
</template>
