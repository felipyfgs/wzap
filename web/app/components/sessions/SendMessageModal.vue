<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const props = defineProps<{ sessionId: string }>()

const { api } = useWzap()
const toast = useToast()
const open = ref(false)
const loading = ref(false)

const schema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number or JID'),
  text: z.string().min(1, 'Message cannot be empty')
})

type Schema = z.output<typeof schema>

const state = reactive<Partial<Schema>>({ phone: '', text: '' })

async function onSubmit(event: FormSubmitEvent<Schema>) {
  loading.value = true
  try {
    const jid = event.data.phone.includes('@')
      ? event.data.phone
      : `${event.data.phone.replace(/\D/g, '')}@s.whatsapp.net`

    await api(`/sessions/${props.sessionId}/messages/text`, {
      method: 'POST',
      body: { jid, text: event.data.text }
    })
    toast.add({ title: 'Message sent', color: 'success' })
    state.phone = ''
    state.text = ''
    open.value = false
  } catch {
    toast.add({ title: 'Failed to send message', color: 'error' })
  }
  loading.value = false
}
</script>

<template>
  <UModal v-model:open="open" title="Send Message" description="Send a WhatsApp text message">
    <UButton label="Send Message" icon="i-lucide-send" color="primary" />

    <template #body>
      <UForm :schema="schema" :state="state" class="space-y-4" @submit="onSubmit">
        <UFormField label="Phone / JID" name="phone" description="E.g. 5511999999999 or full JID">
          <UInput v-model="state.phone" placeholder="5511999999999" class="w-full" />
        </UFormField>
        <UFormField label="Message" name="text">
          <UTextarea v-model="state.text" :rows="4" autoresize placeholder="Type your message…" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton label="Cancel" color="neutral" variant="subtle" @click="open = false" />
          <UButton label="Send" color="primary" type="submit" :loading="loading" icon="i-lucide-send" />
        </div>
      </UForm>
    </template>
  </UModal>
</template>
