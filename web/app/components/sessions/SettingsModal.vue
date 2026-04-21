<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'
import type { Session } from '~/types'

const props = defineProps<{ sessionId: string, sessionName: string }>()
const emit = defineEmits<{ updated: [] }>()

const { api } = useWzap()
const toast = useToast()

const open = ref(false)
const loading = ref(false)
const saving = ref(false)

const schema = z.object({
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
type Schema = z.output<typeof schema>

const state = reactive<Partial<Schema>>({
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

const profileName = ref('')
const profileStatus = ref('')
const profilePictureUrl = ref('')
const savingName = ref(false)
const savingStatus = ref(false)
const savingPicture = ref(false)

const protocolOptions = [
  { label: 'HTTP', value: 'http' },
  { label: 'HTTPS', value: 'https' },
  { label: 'SOCKS5', value: 'socks5' }
]

async function load() {
  loading.value = true
  try {
    const res: { data: Session } = await api(`/sessions/${props.sessionId}`)
    const s = res.data
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

async function onSubmit(event: FormSubmitEvent<Schema>) {
  saving.value = true
  try {
    await api(`/sessions/${props.sessionId}`, {
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
          : null
      }
    })
    toast.add({ title: 'Settings saved', color: 'success' })
    emit('updated')
  } catch {
    toast.add({ title: 'Failed to save settings', color: 'error' })
  }
  saving.value = false
}

async function saveProfileName() {
  if (!profileName.value.trim()) return
  savingName.value = true
  try {
    await api(`/sessions/${props.sessionId}/profile/name`, {
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
    await api(`/sessions/${props.sessionId}/contacts/status`, {
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
    await api(`/sessions/${props.sessionId}/contacts/profile-picture`, {
      method: 'POST',
      body: { image: profilePictureUrl.value.trim() }
    })
    toast.add({ title: 'Profile picture updated', color: 'success' })
  } catch {
    toast.add({ title: 'Failed to update profile picture', color: 'error' })
  }
  savingPicture.value = false
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
    :title="`Settings — ${sessionName}`"
    description="Behavior, proxy, and profile for this session."
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <div v-if="loading" class="flex justify-center py-10">
        <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
      </div>

      <UTabs
        v-else
        :items="[
          { label: 'Behavior', value: 'behavior', icon: 'i-lucide-sliders' },
          { label: 'Proxy', value: 'proxy', icon: 'i-lucide-shield' },
          { label: 'Profile', value: 'profile', icon: 'i-lucide-user' }
        ]"
        default-value="behavior"
        class="w-full"
      >
        <template #content="{ item }">
          <div v-if="item.value !== 'profile'" class="pt-4">
            <UForm
              :schema="schema"
              :state="state"
              class="space-y-4"
              @submit="onSubmit"
            >
              <template v-if="item.value === 'behavior'">
                <div class="grid grid-cols-2 gap-x-6 gap-y-3">
                  <UFormField label="Always Online" name="alwaysOnline">
                    <USwitch v-model="state.alwaysOnline" />
                  </UFormField>
                  <UFormField label="Auto Read" name="readMessages">
                    <USwitch v-model="state.readMessages" />
                  </UFormField>
                  <UFormField label="Reject Calls" name="rejectCall">
                    <USwitch v-model="state.rejectCall" />
                  </UFormField>
                  <UFormField label="Ignore Groups" name="ignoreGroups">
                    <USwitch v-model="state.ignoreGroups" />
                  </UFormField>
                  <UFormField label="Ignore Status" name="ignoreStatus">
                    <USwitch v-model="state.ignoreStatus" />
                  </UFormField>
                </div>
                <UFormField v-if="state.rejectCall" label="Reject message" name="msgRejectCall">
                  <UInput v-model="state.msgRejectCall" placeholder="Sorry, I can't answer calls." class="w-full" />
                </UFormField>
              </template>

              <template v-else-if="item.value === 'proxy'">
                <div class="grid grid-cols-2 gap-3">
                  <UFormField label="Host" name="proxyHost">
                    <UInput v-model="state.proxyHost" placeholder="proxy.example.com" class="w-full" />
                  </UFormField>
                  <UFormField label="Port" name="proxyPort">
                    <UInput
                      v-model.number="state.proxyPort"
                      type="number"
                      placeholder="8080"
                      class="w-full"
                    />
                  </UFormField>
                </div>
                <div class="grid grid-cols-3 gap-3">
                  <UFormField label="Protocol" name="proxyProtocol">
                    <USelect
                      v-model="state.proxyProtocol"
                      :items="protocolOptions"
                      value-key="value"
                      class="w-full"
                    />
                  </UFormField>
                  <UFormField label="Username" name="proxyUsername">
                    <UInput v-model="state.proxyUsername" placeholder="user" class="w-full" />
                  </UFormField>
                  <UFormField label="Password" name="proxyPassword">
                    <UInput v-model="state.proxyPassword" type="password" class="w-full" />
                  </UFormField>
                </div>
              </template>

              <div class="flex justify-end">
                <UButton
                  label="Save"
                  color="primary"
                  size="sm"
                  type="submit"
                  :loading="saving"
                />
              </div>
            </UForm>
          </div>

          <div v-else class="pt-4 space-y-4">
            <div class="flex gap-2">
              <UInput v-model="profileName" placeholder="Profile name" class="flex-1" />
              <UButton
                label="Save"
                size="sm"
                color="primary"
                :loading="savingName"
                @click="saveProfileName"
              />
            </div>
            <div class="flex gap-2">
              <UInput v-model="profileStatus" placeholder="Status message" class="flex-1" />
              <UButton
                label="Save"
                size="sm"
                color="primary"
                :loading="savingStatus"
                @click="saveStatus"
              />
            </div>
            <div class="flex gap-2">
              <UInput v-model="profilePictureUrl" placeholder="https://example.com/photo.jpg" class="flex-1" />
              <UButton
                label="Update"
                size="sm"
                color="primary"
                :loading="savingPicture"
                @click="saveProfilePicture"
              />
            </div>
            <p class="text-xs text-muted">
              Profile changes apply when the session is connected.
            </p>
          </div>
        </template>
      </UTabs>
    </template>
  </UModal>
</template>
