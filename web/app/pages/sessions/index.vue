<script setup lang="ts">
import type { Session } from '~/types'

const { api, isAuthenticated } = useWzap()
const { refreshSessions: refreshShared } = useSession()
const toast = useToast()

const sessions = ref<Session[]>([])
const loading = ref(true)
const nameFilter = ref('')
const qrModal = useTemplateRef('qrModal')
const qrSession = ref<Session | null>(null)

async function fetchSessions() {
  loading.value = true
  try {
    const res: any = await api('/sessions')
    sessions.value = res.data || []
    await refreshShared()
  } catch {
    sessions.value = []
  }
  loading.value = false
}

async function connectSession(session: Session) {
  try {
    const res: any = await api(`/sessions/${session.id}/connect`, { method: 'POST' })
    const status = res.data?.status
    if (status === 'PAIRING') {
      qrSession.value = session
      await nextTick()
      qrModal.value?.show()
    } else {
      toast.add({ title: 'Session connected', color: 'success' })
    }
    await fetchSessions()
  } catch {
    toast.add({ title: 'Failed to connect session', color: 'error' })
  }
}

async function disconnectSession(id: string) {
  try {
    await api(`/sessions/${id}/disconnect`, { method: 'POST' })
    toast.add({ title: 'Session disconnected', color: 'neutral' })
    await fetchSessions()
  } catch {
    toast.add({ title: 'Failed to disconnect session', color: 'error' })
  }
}

async function deleteSession(id: string) {
  try {
    await api(`/sessions/${id}`, { method: 'DELETE' })
    toast.add({ title: 'Session deleted', color: 'success' })
    await fetchSessions()
  } catch {
    toast.add({ title: 'Failed to delete session', color: 'error' })
  }
}

function statusColor(status: string) {
  const map: Record<string, 'success' | 'warning' | 'error' | 'neutral' | 'info'> = {
    connected: 'success',
    connecting: 'warning',
    pairing: 'info',
    disconnected: 'neutral',
    error: 'error'
  }
  return map[status?.toLowerCase()] ?? 'neutral'
}

function dropdownItems(session: Session) {
  return [
    { label: 'Open', icon: 'i-lucide-arrow-right', onSelect: () => navigateTo(`/sessions/${session.id}`) },
    { type: 'separator' as const },
    session.status === 'connected'
      ? { label: 'Disconnect', icon: 'i-lucide-unplug', onSelect: () => disconnectSession(session.id) }
      : { label: 'Connect', icon: 'i-lucide-plug', onSelect: () => connectSession(session) },
    { type: 'separator' as const },
    { label: 'Delete', icon: 'i-lucide-trash', color: 'error' as const, onSelect: () => deleteSession(session.id) }
  ]
}

const filteredSessions = computed(() =>
  sessions.value.filter(s =>
    !nameFilter.value || s.name.toLowerCase().includes(nameFilter.value.toLowerCase())
  )
)

onMounted(() => {
  if (!isAuthenticated.value) {
    navigateTo('/login')
    return
  }
  fetchSessions()
})
</script>

<template>
  <UDashboardPanel id="sessions">
    <template #header>
      <UDashboardNavbar title="Sessions">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <SessionsAddModal @created="fetchSessions" />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <template #left>
          <UInput
            v-model="nameFilter"
            icon="i-lucide-search"
            placeholder="Filter sessions..."
            class="max-w-xs"
          />
        </template>
        <template #right>
          <span class="text-sm text-muted">{{ filteredSessions.length }} session(s)</span>
        </template>
      </UDashboardToolbar>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <div
        v-else-if="filteredSessions.length === 0"
        class="flex flex-col items-center justify-center py-24 gap-3 text-muted"
      >
        <UIcon name="i-lucide-smartphone" class="size-10" />
        <p class="text-sm">No sessions found. Create one to get started.</p>
      </div>

      <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <UCard
          v-for="session in filteredSessions"
          :key="session.id"
          class="flex flex-col gap-0"
          :ui="{ body: 'flex-1' }"
        >
          <template #header>
            <div class="flex items-center justify-between">
              <div class="flex items-center gap-2 min-w-0">
                <UIcon name="i-lucide-smartphone" class="size-4 shrink-0 text-muted" />
                <span class="font-semibold truncate">{{ session.name }}</span>
              </div>
              <div class="flex items-center gap-2 shrink-0">
                <UBadge :color="statusColor(session.status)" variant="subtle" class="capitalize">
                  {{ session.status }}
                </UBadge>
                <UDropdownMenu :items="[dropdownItems(session)]" :content="{ align: 'end' }">
                  <UButton icon="i-lucide-ellipsis-vertical" color="neutral" variant="ghost" size="xs" />
                </UDropdownMenu>
              </div>
            </div>
          </template>

          <div class="space-y-1.5 text-sm">
            <div class="flex items-center gap-2 text-muted">
              <UIcon name="i-lucide-hash" class="size-3.5 shrink-0" />
              <span class="truncate font-mono text-xs">{{ session.id }}</span>
            </div>
            <div class="flex items-center gap-2 text-muted">
              <UIcon name="i-lucide-phone" class="size-3.5 shrink-0" />
              <span>{{ session.jid || 'Not paired' }}</span>
            </div>
          </div>

          <template #footer>
            <div class="flex items-center gap-2">
              <UButton
                icon="i-lucide-arrow-right"
                label="Open"
                size="sm"
                color="primary"
                variant="soft"
                class="flex-1"
                :to="`/sessions/${session.id}`"
              />
              <UButton
                v-if="session.status !== 'connected'"
                icon="i-lucide-plug"
                size="sm"
                color="neutral"
                variant="ghost"
                @click="connectSession(session)"
              />
              <UButton
                v-else
                icon="i-lucide-unplug"
                size="sm"
                color="warning"
                variant="ghost"
                @click="disconnectSession(session.id)"
              />
              <UButton
                icon="i-lucide-trash-2"
                size="sm"
                color="error"
                variant="ghost"
                @click="deleteSession(session.id)"
              />
            </div>
          </template>
        </UCard>
      </div>

      <SessionsQRModal
        v-if="qrSession"
        ref="qrModal"
        :session-id="qrSession.id"
        :session-name="qrSession.name"
        @connected="fetchSessions"
      />
    </template>
  </UDashboardPanel>
</template>
