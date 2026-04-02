<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const route = useRoute()
const { api } = useWzap()
const toast = useToast()

const sessionId = computed(() => route.params.id as string)

const loading = ref(true)
const saving = ref(false)

const settingsSchema = z.object({
  name: z.string().min(1, 'Name is required'),
  alwaysOnline: z.boolean(),
  readMessages: z.boolean(),
  rejectCall: z.boolean(),
  msgRejectCall: z.string().optional(),
  ignoreGroups: z.boolean(),
  ignoreStatus: z.boolean(),
  proxyHost: z.string().optional(),
  proxyPort: z.number().optional(),
  proxyProtocol: z.string().optional(),
  proxyUsername: z.string().optional(),
  proxyPassword: z.string().optional()
})

type SettingsSchema = z.output<typeof settingsSchema>

const state = reactive<Partial<SettingsSchema>>({
  name: '',
  alwaysOnline: false,
  readMessages: false,
  rejectCall: false,
  msgRejectCall: '',
  ignoreGroups: false,
  ignoreStatus: false,
  proxyHost: '',
  proxyPort: undefined,
  proxyProtocol: 'http',
  proxyUsername: '',
  proxyPassword: ''
})

async function fetchSession() {
  loading.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}`)
    const s = res.data
    state.name = s.name
    state.alwaysOnline = s.settings?.alwaysOnline ?? false
    state.readMessages = s.settings?.readMessages ?? false
    state.rejectCall = s.settings?.rejectCall ?? false
    state.msgRejectCall = s.settings?.msgRejectCall ?? ''
    state.ignoreGroups = s.settings?.ignoreGroups ?? false
    state.ignoreStatus = s.settings?.ignoreStatus ?? false
    state.proxyHost = s.proxy?.host ?? ''
    state.proxyPort = s.proxy?.port ?? undefined
    state.proxyProtocol = s.proxy?.protocol ?? 'http'
    state.proxyUsername = s.proxy?.username ?? ''
    state.proxyPassword = s.proxy?.password ?? ''
  } catch {
    toast.add({ title: 'Failed to load session', color: 'error' })
  }
  loading.value = false
}

async function onSubmit(event: FormSubmitEvent<SettingsSchema>) {
  saving.value = true
  try {
    await api(`/sessions/${sessionId.value}`, {
      method: 'PUT',
      body: {
        name: event.data.name,
        settings: {
          alwaysOnline: event.data.alwaysOnline,
          readMessages: event.data.readMessages,
          rejectCall: event.data.rejectCall,
          msgRejectCall: event.data.rejectCall ? (event.data.msgRejectCall || '') : '',
          ignoreGroups: event.data.ignoreGroups,
          ignoreStatus: event.data.ignoreStatus
        },
        proxy: event.data.proxyHost ? {
          host: event.data.proxyHost,
          port: event.data.proxyPort,
          protocol: event.data.proxyProtocol,
          username: event.data.proxyUsername,
          password: event.data.proxyPassword
        } : null
      }
    })
    toast.add({ title: 'Settings saved', color: 'success' })
  } catch {
    toast.add({ title: 'Failed to save settings', color: 'error' })
  }
  saving.value = false
}

const protocolOptions = [
  { label: 'HTTP', value: 'http' },
  { label: 'HTTPS', value: 'https' },
  { label: 'SOCKS5', value: 'socks5' }
]

watch(sessionId, fetchSession, { immediate: true })
</script>

<template>
  <UDashboardPanel id="session-settings">
    <template #header>
      <UDashboardNavbar title="Settings">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <UForm
        v-else
        id="session-settings-form"
        :schema="settingsSchema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <!-- General -->
        <UPageCard
          title="General"
          description="Basic session information."
          variant="naked"
          orientation="horizontal"
          class="mb-0"
        >
          <UButton
            form="session-settings-form"
            label="Save changes"
            color="neutral"
            type="submit"
            :loading="saving"
            class="w-fit lg:ms-auto"
          />
        </UPageCard>

        <UPageCard variant="subtle">
          <UFormField
            name="name"
            label="Session name"
            description="Identifier used in the API and this dashboard."
            required
            class="flex max-sm:flex-col justify-between items-start gap-4"
          >
            <UInput v-model="state.name" placeholder="my-session" autocomplete="off" class="w-full max-w-xs" />
          </UFormField>
        </UPageCard>

        <!-- Behavior -->
        <UPageCard
          title="Behavior"
          description="Configure how this session reacts to incoming events."
          variant="naked"
          orientation="horizontal"
          class="mt-6 mb-0"
        />

        <UPageCard variant="subtle">
          <UFormField
            name="alwaysOnline"
            label="Always Online"
            description="Keep presence as Online even when the app is idle."
            class="flex max-sm:flex-col justify-between sm:items-center gap-4"
          >
            <UToggle v-model="state.alwaysOnline" />
          </UFormField>

          <USeparator />

          <UFormField
            name="readMessages"
            label="Auto Read Messages"
            description="Automatically mark incoming messages as read."
            class="flex max-sm:flex-col justify-between sm:items-center gap-4"
          >
            <UToggle v-model="state.readMessages" />
          </UFormField>

          <USeparator />

          <UFormField
            name="rejectCall"
            label="Reject Calls"
            description="Automatically reject incoming WhatsApp calls."
            class="flex max-sm:flex-col justify-between sm:items-center gap-4"
          >
            <UToggle v-model="state.rejectCall" />
          </UFormField>

          <template v-if="state.rejectCall">
            <USeparator />
            <UFormField
              name="msgRejectCall"
              label="Reject call message"
              description="Message sent automatically when a call is rejected. Leave blank to reject silently."
              class="flex max-sm:flex-col justify-between items-start gap-4"
            >
              <UInput v-model="state.msgRejectCall" placeholder="Sorry, I can't answer calls." class="w-full max-w-xs" />
            </UFormField>
          </template>

          <USeparator />

          <UFormField
            name="ignoreGroups"
            label="Ignore Groups"
            description="Do not emit webhook events for group messages."
            class="flex max-sm:flex-col justify-between sm:items-center gap-4"
          >
            <UToggle v-model="state.ignoreGroups" />
          </UFormField>

          <USeparator />

          <UFormField
            name="ignoreStatus"
            label="Ignore Status"
            description="Do not emit webhook events for WhatsApp Status updates."
            class="flex max-sm:flex-col justify-between sm:items-center gap-4"
          >
            <UToggle v-model="state.ignoreStatus" />
          </UFormField>
        </UPageCard>

        <!-- Proxy -->
        <UPageCard
          title="Proxy"
          description="Route WhatsApp traffic through a proxy server."
          variant="naked"
          orientation="horizontal"
          class="mt-6 mb-0"
        />

        <UPageCard variant="subtle">
          <UFormField
            name="proxyHost"
            label="Host"
            description="Proxy hostname or IP address. Leave blank to disable."
            class="flex max-sm:flex-col justify-between items-start gap-4"
          >
            <UInput v-model="state.proxyHost" placeholder="proxy.example.com" class="w-full max-w-xs" />
          </UFormField>

          <USeparator />

          <UFormField
            name="proxyPort"
            label="Port"
            class="flex max-sm:flex-col justify-between items-start gap-4"
          >
            <UInput
              v-model.number="state.proxyPort"
              type="number"
              placeholder="8080"
              class="w-full max-w-xs"
            />
          </UFormField>

          <USeparator />

          <UFormField
            name="proxyProtocol"
            label="Protocol"
            class="flex max-sm:flex-col justify-between items-start gap-4"
          >
            <USelect v-model="state.proxyProtocol" :items="protocolOptions" value-key="value" class="w-full max-w-xs" />
          </UFormField>

          <USeparator />

          <UFormField
            name="proxyUsername"
            label="Username"
            description="Optional proxy authentication."
            class="flex max-sm:flex-col justify-between items-start gap-4"
          >
            <UInput v-model="state.proxyUsername" placeholder="username" autocomplete="off" class="w-full max-w-xs" />
          </UFormField>

          <USeparator />

          <UFormField
            name="proxyPassword"
            label="Password"
            class="flex max-sm:flex-col justify-between items-start gap-4"
          >
            <UInput v-model="state.proxyPassword" type="password" placeholder="••••••••" autocomplete="off" class="w-full max-w-xs" />
          </UFormField>
        </UPageCard>
      </UForm>
    </template>
  </UDashboardPanel>
</template>
