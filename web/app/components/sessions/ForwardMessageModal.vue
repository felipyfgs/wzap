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
const destinationPhone = ref('')

async function onSubmit() {
  loading.value = true
  try {
    const phone = destinationPhone.value.includes('@')
      ? destinationPhone.value
      : destinationPhone.value.replace(/\D/g, '')
    await api(`/sessions/${props.sessionId}/messages/forward`, {
      method: 'POST',
      body: { phone: props.chatJid, id: props.messageId, to: phone }
    })
    toast.add({ title: 'Message forwarded', color: 'success' })
    open.value = false
    emit('done')
  } catch {
    toast.add({ title: 'Failed to forward message', color: 'error' })
  }
  loading.value = false
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="Forward Message"
    description="Forward this message to another chat"
    icon="i-lucide-forward"
  >
    <slot />

    <template #body>
      <div class="space-y-4">
        <UFormField label="Destination Phone / JID" description="E.g. 5511999999999 or full JID">
          <UInput v-model="destinationPhone" placeholder="5511999999999" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton
            label="Cancel"
            color="neutral"
            variant="subtle"
            @click="open = false"
          />
          <UButton
            label="Forward"
            color="primary"
            icon="i-lucide-forward"
            :loading="loading"
            :disabled="!destinationPhone.trim()"
            @click="onSubmit"
          />
        </div>
      </div>
    </template>
  </UModal>
</template>
