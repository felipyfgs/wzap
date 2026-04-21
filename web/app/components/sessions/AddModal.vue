<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const emit = defineEmits<{ created: [] }>()

const { api } = useWzap()
const toast = useToast()
const open = ref(false)
const loading = ref(false)
const showProxy = ref(false)
const showWebhook = ref(false)

const eventOptions = [
  'All', 'Message', 'Connected', 'Disconnected', 'ConnectFailure', 'LoggedOut',
  'PairSuccess', 'QR', 'Receipt', 'GroupInfo', 'Contact', 'Presence', 'HistorySync'
].map(e => ({ label: e, value: e }))

const protocolOptions = [
  { label: 'HTTP', value: 'http' },
  { label: 'HTTPS', value: 'https' },
  { label: 'SOCKS5', value: 'socks5' }
]

const schema = z.object({
  name: z.string().min(1, 'Required').max(64, 'Too long').regex(/^[a-zA-Z0-9_-]+$/, 'Only letters, numbers, _ and -'),
  token: z.string().optional(),
  settings: z.object({
    alwaysOnline: z.boolean(),
    rejectCall: z.boolean(),
    msgRejectCall: z.string().optional(),
    readMessages: z.boolean(),
    ignoreGroups: z.boolean(),
    ignoreStatus: z.boolean()
  }),
  proxy: z.object({
    host: z.string().optional(),
    port: z.coerce.number().int().min(1).max(65535).optional().or(z.literal(0)),
    protocol: z.string().optional(),
    username: z.string().optional(),
    password: z.string().optional()
  }),
  webhookURL: z.string().optional(),
  webhookEvents: z.array(z.string()).optional()
})

type Schema = z.output<typeof schema>

const state = reactive<Schema>({
  name: '',
  token: '',
  settings: {
    alwaysOnline: false,
    rejectCall: false,
    msgRejectCall: '',
    readMessages: false,
    ignoreGroups: false,
    ignoreStatus: false
  },
  proxy: { host: '', port: 0, protocol: '', username: '', password: '' },
  webhookURL: '',
  webhookEvents: []
})

function generateToken() {
  const arr = new Uint8Array(24)
  window.crypto.getRandomValues(arr)
  state.token = 'sk_' + Array.from(arr).map(b => b.toString(16).padStart(2, '0')).join('')
}

function resetState() {
  state.name = ''
  state.token = ''
  state.settings = { alwaysOnline: false, rejectCall: false, msgRejectCall: '', readMessages: false, ignoreGroups: false, ignoreStatus: false }
  state.proxy = { host: '', port: 0, protocol: '', username: '', password: '' }
  state.webhookURL = ''
  state.webhookEvents = []
  showProxy.value = false
  showWebhook.value = false
}

async function onSubmit(event: FormSubmitEvent<Schema>) {
  loading.value = true
  try {
    const body: Record<string, unknown> = { name: event.data.name }

    if (event.data.token) body.token = event.data.token

    const s = event.data.settings
    if (s.alwaysOnline || s.rejectCall || s.readMessages || s.ignoreGroups || s.ignoreStatus || s.msgRejectCall) {
      body.settings = s
    }

    if (showProxy.value && event.data.proxy.host) {
      body.proxy = event.data.proxy
    }

    if (showWebhook.value && event.data.webhookURL) {
      body.webhook = { url: event.data.webhookURL, events: event.data.webhookEvents }
    }

    await api('/sessions', { method: 'POST', body })
    toast.add({ title: 'Session created', color: 'success' })
    resetState()
    open.value = false
    emit('created')
  } catch {
    toast.add({ title: 'Failed to create session', color: 'error' })
  }
  loading.value = false
}
</script>

<template>
  <UModal v-model:open="open" title="New Session" description="Create a new WhatsApp session">
    <UButton label="New Session" icon="i-lucide-plus" color="primary" />

    <template #body>
      <UForm
        :schema="schema"
        :state="state"
        class="space-y-5"
        @submit="onSubmit"
        @error="(e) => toast.add({ title: 'Validation error', description: e.errors[0]?.message, color: 'error' })"
      >
        <!-- Basic -->
        <div class="space-y-3">
          <UFormField label="Name" name="name" required>
            <UInput v-model="state.name" placeholder="my-session" class="w-full" />
          </UFormField>
          <UFormField label="API Token" name="token" hint="Leave blank to auto-generate">
            <UInput v-model="state.token" placeholder="sk_..." class="w-full">
              <template #trailing>
                <UTooltip text="Generate token">
                  <UButton
                    icon="i-lucide-refresh-cw"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    tabindex="-1"
                    @click.prevent="generateToken"
                  />
                </UTooltip>
              </template>
            </UInput>
          </UFormField>
        </div>

        <USeparator />

        <!-- Settings -->
        <div class="space-y-2">
          <p class="text-sm font-medium text-highlighted">
            Settings
          </p>
          <div class="grid grid-cols-2 gap-x-6 gap-y-2">
            <UFormField label="Always Online" name="settings.alwaysOnline">
              <USwitch v-model="state.settings.alwaysOnline" />
            </UFormField>
            <UFormField label="Read Messages" name="settings.readMessages">
              <USwitch v-model="state.settings.readMessages" />
            </UFormField>
            <UFormField label="Reject Calls" name="settings.rejectCall">
              <USwitch v-model="state.settings.rejectCall" />
            </UFormField>
            <UFormField label="Ignore Groups" name="settings.ignoreGroups">
              <USwitch v-model="state.settings.ignoreGroups" />
            </UFormField>
            <UFormField label="Ignore Status" name="settings.ignoreStatus">
              <USwitch v-model="state.settings.ignoreStatus" />
            </UFormField>
          </div>
          <UFormField v-if="state.settings.rejectCall" label="Reject Call Message" name="settings.msgRejectCall">
            <UInput v-model="state.settings.msgRejectCall" placeholder="Sorry, I can't answer calls." class="w-full" />
          </UFormField>
        </div>

        <USeparator />

        <!-- Proxy -->
        <div class="space-y-3">
          <button type="button" class="flex w-full items-center justify-between text-sm font-medium text-highlighted" @click="showProxy = !showProxy">
            <span>Proxy</span>
            <UIcon :name="showProxy ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="size-4 text-muted" />
          </button>
          <div v-if="showProxy" class="space-y-3">
            <div class="grid grid-cols-2 gap-3">
              <UFormField label="Host" name="proxy.host">
                <UInput v-model="state.proxy.host" placeholder="proxy.example.com" class="w-full" />
              </UFormField>
              <UFormField label="Port" name="proxy.port">
                <UInput
                  v-model="state.proxy.port"
                  type="number"
                  placeholder="8080"
                  class="w-full"
                />
              </UFormField>
            </div>
            <div class="grid grid-cols-3 gap-3">
              <UFormField label="Protocol" name="proxy.protocol">
                <USelect
                  v-model="state.proxy.protocol"
                  :items="protocolOptions"
                  placeholder="Protocol"
                  class="w-full"
                />
              </UFormField>
              <UFormField label="Username" name="proxy.username">
                <UInput v-model="state.proxy.username" placeholder="user" class="w-full" />
              </UFormField>
              <UFormField label="Password" name="proxy.password">
                <UInput
                  v-model="state.proxy.password"
                  type="password"
                  placeholder="••••"
                  class="w-full"
                />
              </UFormField>
            </div>
          </div>
        </div>

        <USeparator />

        <!-- Inline Webhook -->
        <div class="space-y-3">
          <button type="button" class="flex w-full items-center justify-between text-sm font-medium text-highlighted" @click="showWebhook = !showWebhook">
            <span>Webhook</span>
            <UIcon :name="showWebhook ? 'i-lucide-chevron-up' : 'i-lucide-chevron-down'" class="size-4 text-muted" />
          </button>
          <div v-if="showWebhook" class="space-y-3">
            <UFormField label="URL" name="webhookURL">
              <UInput v-model="state.webhookURL" placeholder="https://example.com/hook" class="w-full" />
            </UFormField>
            <UFormField label="Events" name="webhookEvents">
              <USelectMenu
                v-model="state.webhookEvents"
                :items="eventOptions"
                value-key="value"
                multiple
                placeholder="Select events"
                class="w-full"
              />
            </UFormField>
          </div>
        </div>

        <div class="flex justify-end gap-2 pt-1">
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
