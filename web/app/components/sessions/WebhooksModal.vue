<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

interface Webhook {
  id: string
  url: string
  events: string[]
  enabled: boolean
  natsEnabled?: boolean
  secret?: string
}

const props = defineProps<{ sessionId: string, sessionName: string }>()

const { api } = useWzap()
const toast = useToast()

const open = ref(false)
const loading = ref(false)
const saving = ref(false)
const webhooks = ref<Webhook[]>([])
const showForm = ref(false)

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

function resetForm() {
  state.url = ''
  state.events = []
  state.secret = ''
  state.natsEnabled = false
}

async function load() {
  loading.value = true
  try {
    const res: { data: unknown } = await api(`/sessions/${props.sessionId}/webhooks`)
    webhooks.value = (res.data as Webhook[]) || []
  } catch {
    webhooks.value = []
  }
  loading.value = false
}

async function onSubmit(event: FormSubmitEvent<Schema>) {
  saving.value = true
  try {
    await api(`/sessions/${props.sessionId}/webhooks`, {
      method: 'POST',
      body: {
        url: event.data.url,
        events: event.data.events,
        secret: event.data.secret || undefined,
        natsEnabled: event.data.natsEnabled
      }
    })
    toast.add({ title: 'Webhook created', color: 'success' })
    resetForm()
    showForm.value = false
    await load()
  } catch {
    toast.add({ title: 'Failed to create webhook', color: 'error' })
  }
  saving.value = false
}

async function toggle(w: Webhook) {
  try {
    await api(`/sessions/${props.sessionId}/webhooks/${w.id}`, {
      method: 'PUT',
      body: { enabled: !w.enabled }
    })
    await load()
  } catch {
    toast.add({ title: 'Failed to update webhook', color: 'error' })
  }
}

async function remove(w: Webhook) {
  try {
    await api(`/sessions/${props.sessionId}/webhooks/${w.id}`, { method: 'DELETE' })
    toast.add({ title: 'Webhook removed', color: 'success' })
    await load()
  } catch {
    toast.add({ title: 'Failed to remove webhook', color: 'error' })
  }
}

function show() {
  open.value = true
  load()
}
defineExpose({ show })
</script>

<template>
  <UModal
    v-model:open="open"
    :title="`Webhooks — ${sessionName}`"
    description="HTTP callbacks triggered by session events."
    :ui="{ content: 'sm:max-w-3xl' }"
  >
    <template #body>
      <div class="space-y-4">
        <div class="flex items-center justify-between">
          <p class="text-xs text-muted">
            {{ webhooks.length }} webhook(s) registered
          </p>
          <UButton
            :label="showForm ? 'Cancel' : 'Add Webhook'"
            :icon="showForm ? 'i-lucide-x' : 'i-lucide-plus'"
            size="xs"
            :color="showForm ? 'neutral' : 'primary'"
            :variant="showForm ? 'subtle' : 'solid'"
            @click="showForm = !showForm; showForm && resetForm()"
          />
        </div>

        <!-- Inline form -->
        <UCard v-if="showForm" :ui="{ body: 'p-4' }">
          <UForm
            :schema="schema"
            :state="state"
            class="space-y-3"
            @submit="onSubmit"
          >
            <UFormField label="URL" name="url" required>
              <UInput v-model="state.url" placeholder="https://example.com/hook" class="w-full" />
            </UFormField>
            <UFormField label="Events" name="events" required>
              <USelectMenu
                v-model="state.events"
                :items="eventOptions"
                value-key="value"
                multiple
                placeholder="Select events"
                class="w-full"
              />
            </UFormField>
            <div class="grid grid-cols-2 gap-3">
              <UFormField label="Secret (HMAC)" name="secret">
                <UInput v-model="state.secret" placeholder="optional" class="w-full" />
              </UFormField>
              <UFormField label="NATS Streaming" name="natsEnabled">
                <USwitch v-model="state.natsEnabled" />
              </UFormField>
            </div>
            <div class="flex justify-end">
              <UButton
                label="Create"
                color="primary"
                type="submit"
                size="sm"
                :loading="saving"
              />
            </div>
          </UForm>
        </UCard>

        <!-- List -->
        <div v-if="loading" class="flex justify-center py-8">
          <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
        </div>

        <div v-else-if="webhooks.length === 0" class="flex flex-col items-center justify-center py-8 gap-2 text-muted">
          <UIcon name="i-lucide-webhook" class="size-8" />
          <p class="text-sm">
            No webhooks registered
          </p>
        </div>

        <div v-else class="space-y-2">
          <div
            v-for="w in webhooks"
            :key="w.id"
            class="flex items-center gap-3 rounded-lg border border-default px-3 py-2.5"
          >
            <UBadge
              :color="w.enabled ? 'success' : 'neutral'"
              variant="subtle"
              size="xs"
            >
              {{ w.enabled ? 'active' : 'disabled' }}
            </UBadge>
            <div class="min-w-0 flex-1">
              <p class="text-sm font-mono text-highlighted truncate">
                {{ w.url }}
              </p>
              <p class="text-xs text-muted">
                {{ w.events.join(', ') || 'no events' }}
                <span v-if="w.natsEnabled" class="ml-2 text-info">· NATS</span>
              </p>
            </div>
            <UTooltip :text="w.enabled ? 'Disable' : 'Enable'">
              <UButton
                :icon="w.enabled ? 'i-lucide-pause' : 'i-lucide-play'"
                size="xs"
                color="neutral"
                variant="ghost"
                @click="toggle(w)"
              />
            </UTooltip>
            <UTooltip text="Delete">
              <UButton
                icon="i-lucide-trash-2"
                size="xs"
                color="error"
                variant="ghost"
                @click="remove(w)"
              />
            </UTooltip>
          </div>
        </div>
      </div>
    </template>
  </UModal>
</template>
