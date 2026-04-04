<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const route = useRoute()
const { api } = useWzap()
const { refreshCurrent } = useSession()
const toast = useToast()

const sessionId = computed(() => route.params.id as string)

const loading = ref(true)
const saving = ref(false)
const sessionName = ref('')
const sessionEngine = ref('')

const settingsSchema = z.object({
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
  proxyPassword: z.string().optional(),
  phoneNumberId: z.string().optional(),
  accessToken: z.string().optional(),
  businessAccountId: z.string().optional(),
  appSecret: z.string().optional(),
  webhookVerifyToken: z.string().optional()
})

type SettingsSchema = z.output<typeof settingsSchema>

const state = reactive<Partial<SettingsSchema>>({
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
  proxyPassword: '',
  phoneNumberId: '',
  accessToken: '',
  businessAccountId: '',
  appSecret: '',
  webhookVerifyToken: ''
})

const isCloudApi = computed(() => sessionEngine.value === 'cloud_api')

async function fetchSession() {
  loading.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}`)
    const s = res.data
    sessionName.value = s.name
    sessionEngine.value = s.engine ?? ''
    state.phoneNumberId = s.phoneNumberId ?? ''
    state.businessAccountId = s.businessAccountId ?? ''
    state.accessToken = ''
    state.appSecret = ''
    state.webhookVerifyToken = ''
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
        settings: {
          alwaysOnline: event.data.alwaysOnline,
          readMessages: event.data.readMessages,
          rejectCall: event.data.rejectCall,
          msgRejectCall: event.data.rejectCall ? (event.data.msgRejectCall || '') : '',
          ignoreGroups: event.data.ignoreGroups,
          ignoreStatus: event.data.ignoreStatus
        },
        proxy: event.data.proxyHost
          ? {
              host: event.data.proxyHost,
              port: event.data.proxyPort,
              protocol: event.data.proxyProtocol,
              username: event.data.proxyUsername,
              password: event.data.proxyPassword
            }
          : null,
        ...(isCloudApi.value
          ? {
              phoneNumberId: event.data.phoneNumberId || undefined,
              accessToken: event.data.accessToken || undefined,
              businessAccountId: event.data.businessAccountId || undefined,
              appSecret: event.data.appSecret || undefined,
              webhookVerifyToken: event.data.webhookVerifyToken || undefined
            }
          : {})
      }
    })
    toast.add({ title: 'Settings saved', color: 'success' })
    await refreshCurrent(sessionId.value)
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

const profileName = ref('')
const profileStatus = ref('')
const profilePictureUrl = ref('')
const savingName = ref(false)
const savingStatus = ref(false)
const savingPicture = ref(false)

async function saveProfileName() {
  if (!profileName.value.trim()) return
  savingName.value = true
  try {
    await api(`/sessions/${sessionId.value}/profile/name`, {
      method: 'POST',
      body: { name: profileName.value.trim() }
    })
    toast.add({ title: 'Profile name updated', color: 'success' })
  } catch {
    toast.add({ title: 'Failed to update profile name', color: 'error' })
  }
  savingName.value = false
}

async function saveStatus() {
  if (!profileStatus.value.trim()) return
  savingStatus.value = true
  try {
    await api(`/sessions/${sessionId.value}/contacts/status`, {
      method: 'POST',
      body: { status: profileStatus.value.trim() }
    })
    toast.add({ title: 'Status updated', color: 'success' })
  } catch {
    toast.add({ title: 'Failed to update status', color: 'error' })
  }
  savingStatus.value = false
}

async function saveProfilePicture() {
  if (!profilePictureUrl.value.trim()) return
  savingPicture.value = true
  try {
    await api(`/sessions/${sessionId.value}/contacts/profile-picture`, {
      method: 'POST',
      body: { image: profilePictureUrl.value.trim() }
    })
    toast.add({ title: 'Profile picture updated', color: 'success' })
  } catch {
    toast.add({ title: 'Failed to update profile picture', color: 'error' })
  }
  savingPicture.value = false
}

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
        <!-- Behavior -->
        <UPageCard
          title="Behavior"
          description="Configure how this session reacts to incoming events."
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
            name="alwaysOnline"
            label="Always Online"
            description="Keep presence as Online even when the app is idle."
            class="flex max-sm:flex-col justify-between sm:items-center gap-4"
          >
            <USwitch v-model="state.alwaysOnline" />
          </UFormField>

          <USeparator />

          <UFormField
            name="readMessages"
            label="Auto Read Messages"
            description="Automatically mark incoming messages as read."
            class="flex max-sm:flex-col justify-between sm:items-center gap-4"
          >
            <USwitch v-model="state.readMessages" />
          </UFormField>

          <USeparator />

          <UFormField
            name="rejectCall"
            label="Reject Calls"
            description="Automatically reject incoming WhatsApp calls."
            class="flex max-sm:flex-col justify-between sm:items-center gap-4"
          >
            <USwitch v-model="state.rejectCall" />
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
            <USwitch v-model="state.ignoreGroups" />
          </UFormField>

          <USeparator />

          <UFormField
            name="ignoreStatus"
            label="Ignore Status"
            description="Do not emit webhook events for WhatsApp Status updates."
            class="flex max-sm:flex-col justify-between sm:items-center gap-4"
          >
            <USwitch v-model="state.ignoreStatus" />
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
            <USelect
              v-model="state.proxyProtocol"
              :items="protocolOptions"
              value-key="value"
              class="w-full max-w-xs"
            />
          </UFormField>

          <USeparator />

          <UFormField
            name="proxyUsername"
            label="Username"
            description="Optional proxy authentication."
            class="flex max-sm:flex-col justify-between items-start gap-4"
          >
            <UInput
              v-model="state.proxyUsername"
              placeholder="username"
              autocomplete="off"
              class="w-full max-w-xs"
            />
          </UFormField>

          <USeparator />

          <UFormField
            name="proxyPassword"
            label="Password"
            class="flex max-sm:flex-col justify-between items-start gap-4"
          >
            <UInput
              v-model="state.proxyPassword"
              type="password"
              placeholder="••••••••"
              autocomplete="off"
              class="w-full max-w-xs"
            />
          </UFormField>
        </UPageCard>

        <!-- Cloud API (only for cloud_api engine) -->
        <template v-if="isCloudApi">
          <UPageCard
            title="Cloud API"
            description="Meta Cloud API credentials for this session."
            variant="naked"
            orientation="horizontal"
            class="mt-6 mb-0"
          />

          <UPageCard variant="subtle">
            <UFormField
              name="phoneNumberId"
              label="Phone Number ID"
              description="The Phone Number ID from Meta Business dashboard."
              class="flex max-sm:flex-col justify-between items-start gap-4"
            >
              <UInput v-model="state.phoneNumberId" placeholder="123456789" class="w-full max-w-xs" />
            </UFormField>

            <USeparator />

            <UFormField
              name="accessToken"
              label="Access Token"
              description="Leave blank to keep existing token."
              class="flex max-sm:flex-col justify-between items-start gap-4"
            >
              <UInput
                v-model="state.accessToken"
                type="password"
                placeholder="EAAx..."
                autocomplete="off"
                class="w-full max-w-xs"
              />
            </UFormField>

            <USeparator />

            <UFormField
              name="businessAccountId"
              label="Business Account ID"
              class="flex max-sm:flex-col justify-between items-start gap-4"
            >
              <UInput v-model="state.businessAccountId" placeholder="987654321" class="w-full max-w-xs" />
            </UFormField>

            <USeparator />

            <UFormField
              name="appSecret"
              label="App Secret"
              description="Leave blank to keep existing secret."
              class="flex max-sm:flex-col justify-between items-start gap-4"
            >
              <UInput
                v-model="state.appSecret"
                type="password"
                placeholder="App secret"
                autocomplete="off"
                class="w-full max-w-xs"
              />
            </UFormField>

            <USeparator />

            <UFormField
              name="webhookVerifyToken"
              label="Webhook Verify Token"
              description="Token used by Meta to verify your webhook endpoint."
              class="flex max-sm:flex-col justify-between items-start gap-4"
            >
              <UInput v-model="state.webhookVerifyToken" placeholder="verify-token" class="w-full max-w-xs" />
            </UFormField>
          </UPageCard>
        </template>
      </UForm>

      <!-- Chatwoot Integration -->
      <SessionsChatwootConfigCard :session-id="sessionId" :chatwoot-enabled="false" @updated="fetchSession" />

      <!-- Profile -->
      <UPageCard
        title="Profile"
        description="Update your WhatsApp profile name, status, and picture."
        variant="naked"
        orientation="horizontal"
        class="mt-6 mb-0"
      />

      <UPageCard variant="subtle">
        <UFormField
          label="Profile Name"
          description="Your display name on WhatsApp."
          class="flex max-sm:flex-col justify-between items-start gap-4"
        >
          <div class="flex gap-2 w-full max-w-xs">
            <UInput v-model="profileName" placeholder="Your name" class="flex-1" />
            <UButton
              label="Save"
              size="sm"
              color="primary"
              :loading="savingName"
              @click="saveProfileName"
            />
          </div>
        </UFormField>

        <USeparator />

        <UFormField
          label="Status Message"
          description="Your status text shown on WhatsApp."
          class="flex max-sm:flex-col justify-between items-start gap-4"
        >
          <div class="flex gap-2 w-full max-w-xs">
            <UInput v-model="profileStatus" placeholder="Hey there! I am using WhatsApp." class="flex-1" />
            <UButton
              label="Save"
              size="sm"
              color="primary"
              :loading="savingStatus"
              @click="saveStatus"
            />
          </div>
        </UFormField>

        <USeparator />

        <UFormField
          label="Profile Picture"
          description="Set your profile picture via URL."
          class="flex max-sm:flex-col justify-between items-start gap-4"
        >
          <div class="flex gap-2 w-full max-w-xs">
            <UInput v-model="profilePictureUrl" placeholder="https://example.com/photo.jpg" class="flex-1" />
            <UButton
              label="Update"
              size="sm"
              color="primary"
              :loading="savingPicture"
              @click="saveProfilePicture"
            />
          </div>
        </UFormField>
      </UPageCard>

      <SessionsPrivacyCard :session-id="sessionId" />
    </template>
  </UDashboardPanel>
</template>
