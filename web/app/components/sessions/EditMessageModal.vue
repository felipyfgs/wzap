<script setup lang="ts">
const props = defineProps<{
  sessionId: string
  chatJid: string
  messageId: string
}>()

const emit = defineEmits<{ done: [] }>()

const { api } = useWzap()
const toast = useToast()
const open = defineModel<boolean>('open', { default: false })
const loading = ref(false)
const newText = ref('')

async function onSubmit() {
  loading.value = true
  try {
    await api(`/sessions/${props.sessionId}/messages/edit`, {
      method: 'POST',
      body: { phone: props.chatJid, id: props.messageId, body: newText.value }
    })
    toast.add({ title: 'Message edited', color: 'success' })
    open.value = false
    emit('done')
  } catch {
    toast.add({ title: 'Failed to edit message', color: 'error' })
  }
  loading.value = false
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="Edit Message"
    description="Edit the message content"
    icon="i-lucide-pencil"
  >
    <slot />

    <template #body>
      <div class="space-y-4">
        <UFormField label="New content">
          <UTextarea
            v-model="newText"
            :rows="4"
            autoresize
            placeholder="Enter new message text…"
            class="w-full"
          />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton
            label="Cancel"
            color="neutral"
            variant="subtle"
            @click="open = false"
          />
          <UButton
            label="Save"
            color="primary"
            :loading="loading"
            :disabled="!newText.trim()"
            @click="onSubmit"
          />
        </div>
      </div>
    </template>
  </UModal>
</template>
