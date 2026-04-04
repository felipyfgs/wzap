<script setup lang="ts">
const props = defineProps<{
  sessionId: string
  jid: string
  mode: 'add-chat' | 'remove-chat' | 'add-message' | 'remove-message'
  messageId?: string
}>()

const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ done: [] }>()

const { addLabelToChat, removeLabelFromChat, addLabelToMessage, removeLabelFromMessage } = useLabels(props.sessionId)
const toast = useToast()

const labelId = ref('')
const submitting = ref(false)

const titleMap: Record<string, string> = {
  'add-chat': 'Add Label to Chat',
  'remove-chat': 'Remove Label from Chat',
  'add-message': 'Add Label to Message',
  'remove-message': 'Remove Label from Message'
}

async function submit() {
  if (!labelId.value.trim()) return
  submitting.value = true
  try {
    switch (props.mode) {
      case 'add-chat':
        await addLabelToChat(props.jid, labelId.value.trim())
        break
      case 'remove-chat':
        await removeLabelFromChat(props.jid, labelId.value.trim())
        break
      case 'add-message':
        await addLabelToMessage(props.jid, labelId.value.trim(), props.messageId || '')
        break
      case 'remove-message':
        await removeLabelFromMessage(props.jid, labelId.value.trim(), props.messageId || '')
        break
    }
    toast.add({ title: 'Label updated', color: 'success' })
    open.value = false
    labelId.value = ''
    emit('done')
  } catch {
    toast.add({ title: 'Failed to update label', color: 'error' })
  }
  submitting.value = false
}
</script>

<template>
  <UModal v-model:open="open" :title="titleMap[mode]">
    <template #body>
      <UFormField label="Label ID" required>
        <UInput v-model="labelId" placeholder="Enter label ID" class="w-full" />
      </UFormField>
    </template>
    <template #footer>
      <div class="flex justify-end gap-2">
        <UButton
          label="Cancel"
          color="neutral"
          variant="subtle"
          @click="open = false"
        />
        <UButton
          label="Apply"
          color="primary"
          :loading="submitting"
          @click="submit"
        />
      </div>
    </template>
  </UModal>
</template>
