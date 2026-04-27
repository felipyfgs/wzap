<script setup lang="ts">
import type { TableColumn, DropdownMenuItem } from '@nuxt/ui'
import type { Session } from '~/types'
import { sessionStatusColor } from '~/utils'

const { api } = useWzap()
const { sessions, loadingSessions, refreshSessions } = useSession()
const toast = useToast()

const health = ref<{ status: string, services: Record<string, boolean> } | null>(null)
const refreshing = ref(false)

const nameFilter = ref('')
const statusFilter = ref<'all' | string>('all')
const statusOptions = [
  { label: 'All', value: 'all' },
  { label: 'Connected', value: 'connected' },
  { label: 'Disconnected', value: 'disconnected' },
  { label: 'Pairing', value: 'pairing' },
  { label: 'Error', value: 'error' }
]

// ── Modal refs ─────────────────────────────────────────────
const qrModal = useTemplateRef('qrModal')
const pairModal = useTemplateRef('pairModal')
const sendModal = useTemplateRef('sendModal')
const chatwootModal = useTemplateRef('chatwootModal')
const elodeskModal = useTemplateRef('elodeskModal')
const webhooksModal = useTemplateRef('webhooksModal')
const tokenModal = useTemplateRef('tokenModal')
const settingsModal = useTemplateRef('settingsModal')
const eventsDrawer = useTemplateRef('eventsDrawer')

const activeSession = ref<Session | null>(null)

const confirmDeleteOpen = ref(false)
const confirmDeleteId = ref('')
const confirmModal = useTemplateRef('confirmModal')
const confirmLogoutOpen = ref(false)
const confirmLogoutId = ref('')
const confirmLogoutModal = useTemplateRef('confirmLogoutModal')

async function refreshAll() {
  refreshing.value = true
  try {
    const [, healthRes] = await Promise.all([
      refreshSessions(),
      api('/health') as Promise<{ data: { status: string, services: Record<string, boolean> } }>
    ])
    health.value = healthRes.data || null
  } catch {
    health.value = null
  }
  refreshing.value = false
}

const stats = computed(() => {
  const total = sessions.value.length
  const connected = sessions.value.filter(s => s.status === 'connected').length
  const offline = sessions.value.filter(s => s.status === 'disconnected' || s.status === 'error').length
  const pairing = sessions.value.filter(s => s.status === 'pairing' || s.status === 'connecting').length
  return { total, connected, offline, pairing }
})

const filteredSessions = computed(() =>
  sessions.value.filter((s) => {
    const q = nameFilter.value.toLowerCase()
    const matchName = !q
      || s.name.toLowerCase().includes(q)
      || s.id.toLowerCase().includes(q)
      || (s.jid || '').toLowerCase().includes(q)
    const matchStatus = statusFilter.value === 'all' || s.status === statusFilter.value
    return matchName && matchStatus
  })
)

// ── Actions ────────────────────────────────────────────────
async function connectSession(s: Session) {
  try {
    const res: { data: unknown } = await api(`/sessions/${s.id}/connect`, { method: 'POST' })
    if ((res.data as { status?: string } | null)?.status === 'PAIRING') {
      activeSession.value = s
      await nextTick()
      qrModal.value?.show()
    } else {
      toast.add({ title: 'Session connecting', color: 'success' })
    }
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to connect', color: 'error' })
  }
}

async function disconnectSession(s: Session) {
  try {
    await api(`/sessions/${s.id}/disconnect`, { method: 'POST' })
    toast.add({ title: 'Session disconnected', color: 'neutral' })
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to disconnect', color: 'error' })
  }
}

async function restartSession(s: Session) {
  try {
    await api(`/sessions/${s.id}/restart`, { method: 'POST' })
    toast.add({ title: 'Restarting…', color: 'primary' })
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to restart', color: 'error' })
  }
}

async function logoutSession(id: string) {
  try {
    await api(`/sessions/${id}/logout`, { method: 'POST' })
    toast.add({ title: 'Device logged out', color: 'success' })
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to logout', color: 'error' })
  }
}

async function deleteSession(id: string) {
  try {
    await api(`/sessions/${id}`, { method: 'DELETE' })
    toast.add({ title: 'Session deleted', color: 'success' })
    await refreshSessions()
  } catch {
    toast.add({ title: 'Failed to delete', color: 'error' })
  }
}

function openModal(ref: { show?: () => void } | null | undefined, s: Session) {
  activeSession.value = s
  nextTick(() => ref?.show?.())
}

function openQR(s: Session) {
  openModal(qrModal.value, s)
}
function openPair(s: Session) {
  openModal(pairModal.value, s)
}
function openSend(s: Session) {
  openModal(sendModal.value, s)
}
function openChatwoot(s: Session) {
  openModal(chatwootModal.value, s)
}
function openElodesk(s: Session) {
  openModal(elodeskModal.value, s)
}
function openWebhooks(s: Session) {
  openModal(webhooksModal.value, s)
}
function openToken(s: Session) {
  openModal(tokenModal.value, s)
}
function openSettings(s: Session) {
  openModal(settingsModal.value, s)
}
function openEvents(s?: Session) {
  activeSession.value = s || null
  nextTick(() => eventsDrawer.value?.show?.())
}

function rowMenu(s: Session): DropdownMenuItem[][] {
  const isConnected = s.status === 'connected'
  const isPairing = s.status === 'pairing' || s.status === 'connecting'
  return [[
    isConnected
      ? { label: 'Disconnect', icon: 'i-lucide-unplug', onSelect: () => disconnectSession(s) }
      : { label: 'Connect', icon: 'i-lucide-plug', onSelect: () => connectSession(s) },
    isPairing || !s.jid
      ? { label: 'Show QR', icon: 'i-lucide-qr-code', onSelect: () => openQR(s) }
      : undefined,
    { label: 'Pair phone', icon: 'i-lucide-smartphone', onSelect: () => openPair(s) },
    { label: 'Send message', icon: 'i-lucide-send', onSelect: () => openSend(s) },
    { label: 'Live events', icon: 'i-lucide-radio', onSelect: () => openEvents(s) }
  ].filter(Boolean) as DropdownMenuItem[], [
    { label: 'Chatwoot', icon: 'i-lucide-plug-zap', onSelect: () => openChatwoot(s) },
    { label: 'Elodesk', icon: 'i-lucide-building-2', onSelect: () => openElodesk(s) },
    { label: 'Webhooks', icon: 'i-lucide-webhook', onSelect: () => openWebhooks(s) },
    { label: 'API Token', icon: 'i-lucide-key', onSelect: () => openToken(s) },
    { label: 'Settings', icon: 'i-lucide-settings-2', onSelect: () => openSettings(s) }
  ], [
    { label: 'Restart', icon: 'i-lucide-rotate-cw', onSelect: () => restartSession(s) },
    isConnected
      ? {
          label: 'Logout device',
          icon: 'i-lucide-log-out',
          color: 'warning' as const,
          onSelect: () => {
            confirmLogoutId.value = s.id
            confirmLogoutOpen.value = true
          }
        }
      : undefined,
    {
      label: 'Delete',
      icon: 'i-lucide-trash-2',
      color: 'error' as const,
      onSelect: () => {
        confirmDeleteId.value = s.id
        confirmDeleteOpen.value = true
      }
    }
  ].filter(Boolean) as DropdownMenuItem[]]
}

function sessionInitials(name: string): string {
  return name.slice(0, 2).toUpperCase()
}

function timeAgo(dateStr?: string): string {
  if (!dateStr) return ''
  const diff = Date.now() - new Date(dateStr).getTime()
  const mins = Math.floor(diff / 60000)
  if (mins < 1) return 'just now'
  if (mins < 60) return `${mins}m ago`
  const hours = Math.floor(mins / 60)
  if (hours < 24) return `${hours}h ago`
  const days = Math.floor(hours / 24)
  if (days < 30) return `${days}d ago`
  return new Date(dateStr).toLocaleDateString()
}

function phoneFromJid(jid?: string): string {
  if (!jid) return ''
  return jid.replace(/@.*$/, '')
}

const columns: TableColumn<Session>[] = [{
  accessorKey: 'name',
  header: 'Session',
  cell: ({ row }) => h('div', { class: 'flex items-center gap-2.5 min-w-0' }, [
    h('span', { class: 'flex size-8 items-center justify-center rounded-full bg-elevated text-[11px] font-bold ring-1 ring-default shrink-0' }, sessionInitials(row.original.name)),
    h('div', { class: 'min-w-0' }, [
      h('p', { class: 'font-medium text-highlighted text-sm truncate' }, row.original.name),
      h('p', { class: 'text-[11px] text-muted font-mono truncate' }, row.original.id)
    ])
  ])
}, {
  accessorKey: 'status',
  header: 'Status'
}, {
  accessorKey: 'jid',
  header: 'Phone',
  cell: ({ row }) => {
    const p = phoneFromJid(row.original.jid)
    return p ? `+${p}` : h('span', { class: 'text-muted italic text-xs' }, 'not paired')
  }
}, {
  accessorKey: 'createdAt',
  header: 'Created',
  cell: ({ row }) => h('span', { class: 'text-xs text-muted' }, timeAgo(row.original.createdAt))
}, {
  id: 'actions',
  header: ''
}]

onMounted(refreshAll)
</script>

<template>
  <UDashboardPanel id="home">
    <template #header>
      <UDashboardNavbar title="wzap" :ui="{ right: 'gap-2' }">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <UBadge
            v-if="health"
            :color="health.status === 'UP' ? 'success' : 'warning'"
            variant="subtle"
            size="sm"
          >
            API {{ health.status }}
          </UBadge>
          <UButton
            icon="i-lucide-radio"
            label="Live"
            color="neutral"
            variant="ghost"
            size="sm"
            @click="openEvents()"
          />
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            size="sm"
            :loading="refreshing"
            @click="refreshAll"
          />
          <SessionsAddModal @created="refreshAll" />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <template #left>
          <UInput
            v-model="nameFilter"
            icon="i-lucide-search"
            placeholder="Search sessions…"
            size="sm"
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
          <span class="text-xs text-muted">{{ filteredSessions.length }} / {{ sessions.length }}</span>
        </template>
      </UDashboardToolbar>
    </template>

    <template #body>
      <div class="space-y-6">
        <!-- Stats -->
        <UPageGrid class="lg:grid-cols-4 gap-4 sm:gap-6 lg:gap-px">
          <UPageCard
            icon="i-lucide-smartphone"
            title="Sessions"
            variant="subtle"
            :ui="{ container: 'gap-y-1.5', wrapper: 'items-start', leading: 'p-2.5 rounded-full bg-primary/10 ring ring-inset ring-primary/25', title: 'font-normal text-muted text-xs uppercase' }"
            class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
          >
            <span class="text-2xl font-semibold text-highlighted">{{ stats.total }}</span>
          </UPageCard>
          <UPageCard
            icon="i-lucide-wifi"
            title="Connected"
            variant="subtle"
            :ui="{ container: 'gap-y-1.5', wrapper: 'items-start', leading: 'p-2.5 rounded-full bg-success/10 ring ring-inset ring-success/25', title: 'font-normal text-muted text-xs uppercase' }"
            class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
          >
            <span class="text-2xl font-semibold text-success">{{ stats.connected }}</span>
          </UPageCard>
          <UPageCard
            icon="i-lucide-qr-code"
            title="Pairing"
            variant="subtle"
            :ui="{ container: 'gap-y-1.5', wrapper: 'items-start', leading: 'p-2.5 rounded-full bg-info/10 ring ring-inset ring-info/25', title: 'font-normal text-muted text-xs uppercase' }"
            class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
          >
            <span class="text-2xl font-semibold text-highlighted">{{ stats.pairing }}</span>
          </UPageCard>
          <UPageCard
            icon="i-lucide-wifi-off"
            title="Offline"
            variant="subtle"
            :ui="{ container: 'gap-y-1.5', wrapper: 'items-start', leading: 'p-2.5 rounded-full bg-neutral/10 ring ring-inset ring-neutral/25', title: 'font-normal text-muted text-xs uppercase' }"
            class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
          >
            <span class="text-2xl font-semibold text-highlighted">{{ stats.offline }}</span>
          </UPageCard>
        </UPageGrid>

        <!-- Sessions table -->
        <UCard :ui="{ body: 'p-0 sm:p-0' }">
          <UTable
            :columns="columns"
            :data="filteredSessions"
            :loading="loadingSessions"
            class="w-full"
            :ui="TABLE_UI"
          >
            <template #status-cell="{ row }">
              <UBadge
                :color="sessionStatusColor(row.original.status)"
                variant="subtle"
                class="capitalize"
                size="sm"
              >
                {{ row.original.status }}
              </UBadge>
            </template>

            <template #actions-cell="{ row }">
              <div class="flex justify-end gap-1">
                <UTooltip :text="row.original.status === 'connected' ? 'Disconnect' : 'Connect'">
                  <UButton
                    :icon="row.original.status === 'connected' ? 'i-lucide-unplug' : 'i-lucide-plug'"
                    size="xs"
                    :color="row.original.status === 'connected' ? 'warning' : 'success'"
                    variant="ghost"
                    @click="row.original.status === 'connected' ? disconnectSession(row.original) : connectSession(row.original)"
                  />
                </UTooltip>
                <UTooltip
                  v-if="row.original.status === 'pairing' || !row.original.jid"
                  text="Show QR"
                >
                  <UButton
                    icon="i-lucide-qr-code"
                    size="xs"
                    color="info"
                    variant="ghost"
                    @click="openQR(row.original)"
                  />
                </UTooltip>
                <UTooltip text="Send message">
                  <UButton
                    icon="i-lucide-send"
                    size="xs"
                    color="primary"
                    variant="ghost"
                    :disabled="row.original.status !== 'connected'"
                    @click="openSend(row.original)"
                  />
                </UTooltip>
                <UTooltip text="Webhooks">
                  <UButton
                    icon="i-lucide-webhook"
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    @click="openWebhooks(row.original)"
                  />
                </UTooltip>
                <UTooltip text="Chatwoot">
                  <UButton
                    icon="i-lucide-plug-zap"
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    @click="openChatwoot(row.original)"
                  />
                </UTooltip>
                <UTooltip text="Elodesk">
                  <UButton
                    icon="i-lucide-building-2"
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    @click="openElodesk(row.original)"
                  />
                </UTooltip>
                <UDropdownMenu :items="rowMenu(row.original)" :content="{ align: 'end' }">
                  <UButton
                    icon="i-lucide-ellipsis-vertical"
                    size="xs"
                    color="neutral"
                    variant="ghost"
                  />
                </UDropdownMenu>
              </div>
            </template>

            <template #empty>
              <div class="flex flex-col items-center justify-center py-16 gap-3 text-muted">
                <UIcon name="i-lucide-smartphone" class="size-8" />
                <p class="text-sm">
                  No sessions yet.
                </p>
                <p class="text-xs max-w-sm text-center">
                  Click <strong>New Session</strong> to create one. The full API surface is also available — see the Swagger UI at <code class="font-mono">/swagger</code>.
                </p>
              </div>
            </template>
          </UTable>
        </UCard>
      </div>

      <!-- Modals / Drawers (rendered once, addressed via activeSession) -->
      <template v-if="activeSession">
        <SessionsQRModal
          ref="qrModal"
          :session-id="activeSession.id"
          :session-name="activeSession.name"
          @connected="refreshSessions"
        />
        <SessionsPairPhoneModal
          ref="pairModal"
          :session-id="activeSession.id"
          @paired="refreshSessions"
        />
        <SessionsSendMessageModal
          ref="sendModal"
          :session-id="activeSession.id"
        />
        <SessionsChatwootModal
          ref="chatwootModal"
          :session-id="activeSession.id"
          :session-name="activeSession.name"
          @updated="refreshSessions"
        />
        <SessionsElodeskModal
          ref="elodeskModal"
          :session-id="activeSession.id"
          :session-name="activeSession.name"
          @updated="refreshSessions"
        />
        <SessionsWebhooksModal
          ref="webhooksModal"
          :session-id="activeSession.id"
          :session-name="activeSession.name"
        />
        <SessionsTokenModal
          ref="tokenModal"
          :session-id="activeSession.id"
          :session-name="activeSession.name"
          :token="activeSession.token"
          @rotated="refreshSessions"
        />
        <SessionsSettingsModal
          ref="settingsModal"
          :session-id="activeSession.id"
          :session-name="activeSession.name"
          @updated="refreshSessions"
        />
      </template>

      <LiveEventsDrawer
        ref="eventsDrawer"
        :session-id-filter="activeSession?.id"
      />

      <SessionsConfirmModal
        ref="confirmModal"
        v-model:open="confirmDeleteOpen"
        title="Delete Session"
        description="This permanently removes the session and its WhatsApp credentials."
        confirm-label="Delete"
        confirm-color="error"
        icon="i-lucide-trash"
        @confirm="async () => { await deleteSession(confirmDeleteId); confirmModal?.done() }"
      />

      <SessionsConfirmModal
        ref="confirmLogoutModal"
        v-model:open="confirmLogoutOpen"
        title="Logout device"
        description="Unlinks this session from the paired phone. You'll need to scan the QR code again."
        confirm-label="Logout"
        confirm-color="warning"
        icon="i-lucide-log-out"
        @confirm="async () => { await logoutSession(confirmLogoutId); confirmLogoutModal?.done() }"
      />
    </template>
  </UDashboardPanel>
</template>
