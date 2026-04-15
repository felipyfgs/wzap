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
const emoji = ref('')

const quickEmojis = ['👍', '❤️', '😂', '😮', '😢', '🙏']

async function onSubmit(selectedEmoji?: string) {
  const reaction = selectedEmoji || emoji.value
  if (!reaction) return
  loading.value = true
  try {
    await api(`/sessions/${props.sessionId}/messages/reaction`, {
      method: 'POST',
      body: { phone: props.chatJid, id: props.messageId, emoji: reaction }
    })
    toast.add({ title: 'Reaction sent', color: 'success' })
    open.value = false
    emit('done')
  } catch {
    toast.add({ title: 'Failed to send reaction', color: 'error' })
  }
  loading.value = false
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="React to Message"
    description="Send an emoji reaction"
    icon="i-lucide-smile"
  >
    <slot />

    <template #body>
      <div class="space-y-4">
        <div class="flex gap-2 flex-wrap">
          <UButton
            v-for="e in quickEmojis"
            :key="e"
            :label="e"
            size="lg"
            color="neutral"
            variant="subtle"
            @click="onSubmit(e)"
          />
        </div>
        <UFormField label="Or enter an emoji">
          <UInput v-model="emoji" placeholder="😀" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton
            label="Cancel"
            color="neutral"
            variant="subtle"
            @click="open = false"
          />
          <UButton
            label="React"
            color="primary"
            :loading="loading"
            :disabled="!emoji.trim()"
            @click="onSubmit()"
          />
        </div>
      </div>
    </template>
  </UModal>
</template>
