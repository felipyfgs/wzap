<script setup lang="ts">
const props = defineProps<{
  sessionId: string
  chatJid: string
}>()

const { api } = useWzap()
const toast = useToast()
const open = defineModel<boolean>('open', { default: false })
const loading = ref(false)
const presenceType = ref('composing')

const presenceOptions = [
  { label: 'Composing (typing…)', value: 'composing' },
  { label: 'Recording (audio…)', value: 'recording' },
  { label: 'Paused', value: 'paused' }
]

async function onSubmit() {
  loading.value = true
  try {
    await api(`/sessions/${props.sessionId}/messages/presence`, {
      method: 'POST',
      body: { phone: props.chatJid, type: presenceType.value }
    })
    toast.add({ title: 'Presence sent', color: 'success' })
    open.value = false
  } catch {
    toast.add({ title: 'Failed to send presence', color: 'error' })
  }
  loading.value = false
}
</script>

<template>
  <UModal
    v-model:open="open"
    title="Set Presence"
    description="Send a presence state to this chat"
    icon="i-lucide-radio"
  >
    <slot />

    <template #body>
      <div class="space-y-4">
        <UFormField label="Presence Type">
          <USelectMenu
            v-model="presenceType"
            :items="presenceOptions"
            value-key="value"
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
            label="Send"
            color="primary"
            :loading="loading"
            @click="onSubmit"
          />
        </div>
      </div>
    </template>
  </UModal>
</template>
