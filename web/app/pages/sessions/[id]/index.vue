<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'

const route = useRoute()
const { api } = useWzap()
const { current: session, profile, refreshCurrent, refreshSessions } = useSession()
const toast = useToast()

const sessionId = computed(() => route.params.id as string)
const loading = ref(false)
const qrModal = useTemplateRef('qrModal')
const pairModal = useTemplateRef('pairModal')

async function fetchSession() {
  loading.value = true
  try {
    await refreshCurrent(sessionId.value)
  } finally {
    loading.value = false
  }
}

async function connect() {
  try {
    const res: { data: unknown } = await api(`/sessions/${sessionId.value}/connect`, { method: 'POST' })
    if (res.data?.status === 'PAIRING') {
      await nextTick()
      qrModal.value?.show()
    } else {
      toast.add({ title: 'Session connected', color: 'success' })
    }
    await fetchSession()
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to connect', color: 'error' })
  }
}

async function disconnect() {
  try {
    await api(`/sessions/${sessionId.value}/disconnect`, { method: 'POST' })
    toast.add({ title: 'Session disconnected', color: 'neutral' })
    await fetchSession()
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to disconnect', color: 'error' })
  }
}

async function reconnect() {
  try {
    await api(`/sessions/${sessionId.value}/reconnect`, { method: 'POST' })
    toast.add({ title: 'Reconnecting…', color: 'info' })
    await fetchSession()
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to reconnect', color: 'error' })
  }
}

async function restart() {
  try {
    await api(`/sessions/${sessionId.value}/restart`, { method: 'POST' })
    toast.add({ title: 'Restarting session…', color: 'info' })
    await fetchSession()
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to restart', color: 'error' })
  }
}

async function logout() {
  try {
    await api(`/sessions/${sessionId.value}/logout`, { method: 'POST' })
    toast.add({ title: 'Session logged out', color: 'neutral' })
    await fetchSession()
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to logout', color: 'error' })
  }
}

async function deleteSession() {
  try {
    await api(`/sessions/${sessionId.value}`, { method: 'DELETE' })
    toast.add({ title: 'Session deleted', color: 'success' })
    await refreshSessions()
    navigateTo('/sessions')
  } catch {
    toast.add({ title: 'Failed to delete session', color: 'error' })
  }
}

const isConnected = computed(() => session.value?.status === 'connected')
const isPairing = computed(() => ['pairing', 'connecting'].includes(session.value?.status || ''))

const navbarActions = computed<DropdownMenuItem[][]>(() => [
  [{
    label: 'Copy Session ID',
    icon: 'i-lucide-copy',
    onSelect() {
      navigator.clipboard.writeText(session.value?.id || '')
      toast.add({ title: 'Session ID copied', color: 'success' })
    }
  }, {
    label: 'Copy API Key',
    icon: 'i-lucide-key',
    disabled: !session.value?.apiKey,
    onSelect() {
      navigator.clipboard.writeText(session.value?.apiKey || '')
      toast.add({ title: 'API Key copied', color: 'success' })
    }
  }],
  [{
    label: 'Reconnect',
    icon: 'i-lucide-rotate-ccw',
    onSelect: reconnect
  }, {
    label: 'Restart',
    icon: 'i-lucide-power',
    onSelect: restart
  }],
  [{
    label: 'Logout device',
    icon: 'i-lucide-log-out',
    color: 'error' as const,
    onSelect: logout
  }, {
    label: 'Delete session',
    icon: 'i-lucide-trash',
    color: 'error' as const,
    onSelect: deleteSession
  }]
])

onMounted(() => fetchSession())
watch(sessionId, fetchSession)
</script>

<template>
  <UDashboardPanel id="session-overview">
    <template #header>
      <!-- Navbar -->
      <UDashboardNavbar :title="session?.name || 'Session'" :ui="{ right: 'gap-2' }">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <UButton
            v-if="session && isPairing"
            icon="i-lucide-qr-code"
            label="Show QR"
            size="sm"
            color="info"
            variant="soft"
            @click="qrModal?.show()"
          />
          <UButton
            v-if="session && !isConnected"
            icon="i-lucide-phone"
            label="Pair by Phone"
            size="sm"
            color="neutral"
            variant="soft"
            @click="pairModal?.show()"
          />
          <UButton
            v-if="session && !isConnected"
            icon="i-lucide-plug"
            label="Connect"
            size="sm"
            color="primary"
            @click="connect"
          />
          <UButton
            v-else-if="session && isConnected"
            icon="i-lucide-unplug"
            label="Disconnect"
            size="sm"
            color="warning"
            variant="soft"
            @click="disconnect"
          />

          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            size="sm"
            :loading="loading"
            @click="fetchSession"
          />

          <UDropdownMenu
            v-if="session"
            :items="navbarActions"
            :content="{ align: 'end' }"
          >
            <UButton
              icon="i-lucide-ellipsis-vertical"
              color="neutral"
              variant="ghost"
              size="sm"
            />
          </UDropdownMenu>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <!-- Loading -->
      <div v-if="loading" class="flex items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <!-- Not found -->
      <div v-else-if="!session" class="flex flex-col items-center justify-center py-24 gap-3 text-muted">
        <UIcon name="i-lucide-smartphone" class="size-10" />
        <p class="text-sm">
          Session not found.
        </p>
        <UButton
          label="Back to Sessions"
          icon="i-lucide-arrow-left"
          variant="soft"
          to="/sessions"
        />
      </div>

      <!-- Content -->
      <div v-else class="space-y-6">
        <SessionsStatsRow :session="session" :profile="profile" />

        <!-- Engine / Integration badges -->
        <div v-if="session.engine === 'cloud_api' || session.chatwootEnabled" class="flex items-center gap-2">
          <UBadge
            v-if="session.engine === 'cloud_api'"
            color="info"
            variant="subtle"
            icon="i-lucide-cloud"
          >
            Cloud API
          </UBadge>
          <UBadge
            v-if="session.chatwootEnabled"
            color="warning"
            variant="subtle"
            icon="i-lucide-message-circle"
          >
            Chatwoot
          </UBadge>
        </div>

        <!-- Bloco B: cards individuais -->
        <div class="grid lg:grid-cols-2 gap-4">
          <SessionsConnectionCard :session="session" />
          <SessionsSettingsCard :session="session" />
        </div>

        <UCard v-if="session.proxy?.host">
          <template #header>
            <p class="font-semibold text-highlighted">
              Proxy
            </p>
          </template>
          <p class="text-sm font-mono text-highlighted">
            {{ session.proxy.protocol || 'http' }}://{{ session.proxy.host }}:{{ session.proxy.port }}
          </p>
        </UCard>

        <!-- Bloco C: Danger Zone -->
        <UCard class="ring-1 ring-error/20">
          <template #header>
            <div class="flex items-center gap-2">
              <UIcon name="i-lucide-triangle-alert" class="size-4 text-error" />
              <p class="font-semibold text-error">
                Danger Zone
              </p>
            </div>
          </template>

          <div class="space-y-3">
            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm font-medium">
                  Reconnect session
                </p>
                <p class="text-xs text-muted">
                  Force disconnect and reconnect to WhatsApp.
                </p>
              </div>
              <UButton
                label="Reconnect"
                icon="i-lucide-rotate-ccw"
                size="sm"
                color="neutral"
                variant="outline"
                @click="reconnect"
              />
            </div>

            <USeparator />

            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm font-medium">
                  Restart session
                </p>
                <p class="text-xs text-muted">
                  Stop and restart the WhatsApp engine for this session.
                </p>
              </div>
              <UButton
                label="Restart"
                icon="i-lucide-power"
                size="sm"
                color="warning"
                variant="soft"
                @click="restart"
              />
            </div>

            <USeparator />

            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm font-medium">
                  Logout device
                </p>
                <p class="text-xs text-muted">
                  Unpair this device from WhatsApp. A new QR scan will be required.
                </p>
              </div>
              <UButton
                label="Logout"
                icon="i-lucide-log-out"
                size="sm"
                color="error"
                variant="soft"
                @click="logout"
              />
            </div>

            <USeparator />

            <div class="flex items-center justify-between">
              <div>
                <p class="text-sm font-medium">
                  Delete session
                </p>
                <p class="text-xs text-muted">
                  Permanently remove this session and all its data.
                </p>
              </div>
              <UButton
                label="Delete"
                icon="i-lucide-trash"
                size="sm"
                color="error"
                variant="solid"
                @click="deleteSession"
              />
            </div>
          </div>
        </UCard>
      </div>

      <!-- QR Modal -->
      <SessionsQRModal
        v-if="session"
        ref="qrModal"
        :session-id="session.id"
        :session-name="session.name"
        @connected="fetchSession"
      />

      <!-- Pair by Phone Modal -->
      <SessionsPairPhoneModal
        v-if="session"
        ref="pairModal"
        :session-id="session.id"
        @paired="fetchSession"
      />
    </template>
  </UDashboardPanel>
</template>
