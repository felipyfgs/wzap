<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const props = defineProps<{ sessionId: string }>()
const emit = defineEmits<{ created: [] }>()

const { api } = useWzap()
const toast = useToast()
const open = ref(false)
const loading = ref(false)

const eventOptions = [
  'All', 'Message', 'Connected', 'Disconnected', 'ConnectFailure', 'LoggedOut',
  'PairSuccess', 'QR', 'Receipt', 'GroupInfo', 'Contact', 'Presence', 'HistorySync'
].map(e => ({ label: e, value: e }))

const schema = z.object({
  url: z.string().url('Invalid URL'),
  events: z.array(z.string()).min(1, 'Select at least one event'),
  secret: z.string().optional(),
  natsEnabled: z.boolean()
})

type Schema = z.output<typeof schema>

const state = reactive<Partial<Schema>>({ url: '', events: [], secret: '', natsEnabled: false })

async function onSubmit(event: FormSubmitEvent<Schema>) {
  loading.value = true
  try {
    await api(`/sessions/${props.sessionId}/webhooks`, {
      method: 'POST',
      body: { url: event.data.url, events: event.data.events, secret: event.data.secret || undefined, natsEnabled: event.data.natsEnabled }
    })
    toast.add({ title: 'Webhook created', color: 'success' })
    state.url = ''
    state.events = []
    state.secret = ''
    state.natsEnabled = false
    open.value = false
    emit('created')
  } catch {
    toast.add({ title: 'Failed to create webhook', color: 'error' })
  }
  loading.value = false
}
</script>

<template>
  <UModal v-model:open="open" title="New Webhook" description="Register a webhook for this session">
    <UButton label="New Webhook" icon="i-lucide-plus" color="primary" />

    <template #body>
      <UForm
        :schema="schema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <UFormField label="URL" name="url">
          <UInput v-model="state.url" placeholder="https://example.com/hook" class="w-full" />
        </UFormField>
        <UFormField label="Events" name="events">
          <USelectMenu
            v-model="state.events"
            :items="eventOptions"
            value-key="value"
            multiple
            placeholder="Select events"
            class="w-full"
          />
        </UFormField>
        <UFormField label="Secret (optional)" name="secret">
          <UInput v-model="state.secret" placeholder="webhook-secret" class="w-full" />
        </UFormField>
        <UFormField label="NATS Streaming" name="natsEnabled">
          <USwitch v-model="state.natsEnabled" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton
            label="Cancel"
            color="neutral"
            variant="subtle"
            @click="open = false"
          />
          <UButton
            label="Create"
            color="primary"
            type="submit"
            :loading="loading"
          />
        </div>
      </UForm>
    </template>
  </UModal>
</template>
