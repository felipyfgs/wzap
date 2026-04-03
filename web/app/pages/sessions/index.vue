<script setup lang="ts">
import type { Session } from '~/types'

const { api, isAuthenticated } = useWzap()
const { sessions, loadingSessions, refreshSessions } = useSession()
const toast = useToast()

const nameFilter = ref('')
const statusFilter = ref('all')
const visibleTokenIds = ref<string[]>([])
const qrModal = useTemplateRef('qrModal')
const qrSession = ref<Session | null>(null)

const statusOptions = [
  { label: 'All', value: 'all' },
  { label: 'Connected', value: 'connected' },
  { label: 'Disconnected', value: 'disconnected' },
  { label: 'Pairing', value: 'pairing' }
]

function toggleToken(id: string) {
  const idx = visibleTokenIds.value.indexOf(id)
  if (idx >= 0) visibleTokenIds.value.splice(idx, 1)
  else visibleTokenIds.value.push(id)
}

async function copyText(text: string, label: string) {
  await navigator.clipboard.writeText(text)
  toast.add({ title: `${label} copied`, color: 'success' })
}

async function connectSession(session: Session) {
  try {
    const res: any = await api(`/sessions/${session.id}/connect`, { method: 'POST' })
    if (res.data?.status === 'PAIRING') {
      qrSession.value = session
      await nextTick()
      qrModal.value?.show()
    } else {
      toast.add({ title: 'Session connected', color: 'success' })
    }
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to connect session', color: 'error' })
  }
}

async function disconnectSession(id: string) {
  try {
    await api(`/sessions/${id}/disconnect`, { method: 'POST' })
    toast.add({ title: 'Session disconnected', color: 'neutral' })
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to disconnect session', color: 'error' })
  }
}

async function deleteSession(id: string) {
  try {
    await api(`/sessions/${id}`, { method: 'DELETE' })
    toast.add({ title: 'Session deleted', color: 'success' })
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to delete session', color: 'error' })
  }
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
  sessions.value.filter(s => {
    const matchesName = !nameFilter.value || s.name.toLowerCase().includes(nameFilter.value.toLowerCase())
    const matchesStatus = statusFilter.value === 'all' || s.status === statusFilter.value
    return matchesName && matchesStatus
  })
)

onMounted(() => {
  if (!isAuthenticated.value) {
    navigateTo('/login')
    return
  }
  refreshSessions()
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
          <SessionsAddModal @created="refreshSessions" />
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
          <USelectMenu
            v-model="statusFilter"
            :items="statusOptions"
            value-key="value"
            :search-input="false"
            size="sm"
            color="neutral"
            class="w-36"
          />
        </template>
        <template #right>
          <span class="text-sm text-muted">{{ filteredSessions.length }} session(s)</span>
        </template>
      </UDashboardToolbar>
    </template>

    <template #body>
      <div v-if="loadingSessions" class="flex items-center justify-center py-24">
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
                <UBadge :color="sessionStatusColor(session.status)" variant="subtle" class="capitalize">
                  {{ session.status }}
                </UBadge>
                <UDropdownMenu :items="[dropdownItems(session)]" :content="{ align: 'end' }">
                  <UButton icon="i-lucide-ellipsis-vertical" color="neutral" variant="ghost" size="xs" />
                </UDropdownMenu>
              </div>
            </div>
          </template>

          <div class="space-y-2 text-sm">
            <!-- Session ID -->
            <div class="flex items-center justify-between gap-2 text-muted">
              <div class="flex items-center gap-2 min-w-0">
                <UIcon name="i-lucide-hash" class="size-3.5 shrink-0" />
                <span class="truncate font-mono text-xs">{{ session.id }}</span>
              </div>
              <UButton icon="i-lucide-copy" size="xs" color="neutral" variant="ghost" class="shrink-0" @click="copyText(session.id, 'Session ID')" />
            </div>

            <!-- Phone / JID -->
            <div class="flex items-center gap-2 text-muted">
              <UIcon name="i-lucide-phone" class="size-3.5 shrink-0" />
              <template v-if="session.jid">
                <span class="font-mono text-xs">+{{ parseJID(session.jid).phone }}</span>
                <UBadge v-if="parseJID(session.jid).device > 0" size="xs" color="neutral" variant="subtle">
                  Device {{ parseJID(session.jid).device }}
                </UBadge>
              </template>
              <span v-else class="text-xs italic">Not paired</span>
            </div>

            <!-- Token -->
            <div v-if="session.apiKey" class="flex items-center justify-between gap-2 text-muted">
              <div class="flex items-center gap-2 min-w-0">
                <UIcon name="i-lucide-key" class="size-3.5 shrink-0" />
                <span class="font-mono text-xs truncate">
                  {{ visibleTokenIds.includes(session.id) ? session.apiKey : '••••••••••••' }}
                </span>
              </div>
              <div class="flex items-center gap-1 shrink-0">
                <UButton
                  :icon="visibleTokenIds.includes(session.id) ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                  size="xs" color="neutral" variant="ghost"
                  @click="toggleToken(session.id)"
                />
                <UButton
                  icon="i-lucide-copy"
                  size="xs" color="neutral" variant="ghost"
                  @click="copyText(session.apiKey!, 'Token')"
                />
              </div>
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
        @connected="refreshSessions"
      />
    </template>
  </UDashboardPanel>
</template>
