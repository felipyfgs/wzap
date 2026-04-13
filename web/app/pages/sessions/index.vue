<script setup lang="ts">
import type { Session } from '~/types'

const { api } = useWzap()
const { sessions, loadingSessions, refreshSessions } = useSession()
const toast = useToast()

const pictures = reactive<Map<string, string | null>>(new Map())

async function loadPictures() {
  const connected = sessions.value.filter((s) => s.status === 'connected')
  await Promise.allSettled(
    connected.map(async (s) => {
      try {
        const res: any = await api(`/sessions/${s.id}/profile`)
        pictures.set(s.id, res.data?.pictureUrl || null)
      } catch {
        pictures.set(s.id, null)
      }
    })
  )
}

const nameFilter = ref('')
const statusFilter = ref('all')
const qrModal = useTemplateRef('qrModal')
const qrSession = ref<Session | null>(null)

const statusOptions = [
  { label: 'All', value: 'all' },
  { label: 'Connected', value: 'connected' },
  { label: 'Disconnected', value: 'disconnected' },
  { label: 'Pairing', value: 'pairing' }
]

async function copyText(text: string, label: string) {
  await navigator.clipboard.writeText(text)
  toast.add({ title: `${label} copied`, color: 'success' })
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

function platformIcon(platform?: string): string {
  if (!platform) return ''
  const map: Record<string, string> = {
    android: 'i-simple-icons-android',
    ios: 'i-simple-icons-apple',
    web: 'i-lucide-globe',
    desktop: 'i-lucide-monitor'
  }
  return map[platform.toLowerCase()] || 'i-lucide-smartphone'
}

function sessionInitials(name: string): string {
  return name.slice(0, 2).toUpperCase()
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
  sessions.value.filter((s) => {
    const matchesName = !nameFilter.value || s.name.toLowerCase().includes(nameFilter.value.toLowerCase())
    const matchesStatus = statusFilter.value === 'all' || s.status === statusFilter.value
    return matchesName && matchesStatus
  })
)

onMounted(async () => {
  await refreshSessions()
  loadPictures()
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
        <p class="text-sm">
          No sessions found. Create one to get started.
        </p>
      </div>

      <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <UCard
          v-for="session in filteredSessions"
          :key="session.id"
          class="flex flex-col gap-0"
          :ui="{ body: 'flex-1 !p-0' }"
        >
          <div class="flex items-center gap-3 px-4 py-3">
            <div class="relative shrink-0">
              <img
                v-if="pictures.get(session.id)"
                :src="pictures.get(session.id)!"
                class="size-9 rounded-full object-cover ring-1 ring-default"
                @error="pictures.set(session.id, null)"
              >
              <span
                v-else
                class="flex size-9 items-center justify-center rounded-full bg-elevated text-xs font-bold ring-1 ring-default"
              >
                {{ sessionInitials(session.name) }}
              </span>
              <span
                class="absolute -bottom-0.5 -right-0.5 size-2.5 rounded-full ring-2 ring-white dark:ring-gray-900"
                :class="{
                  'bg-success': session.status === 'connected',
                  'bg-warning': session.status === 'connecting',
                  'bg-info': session.status === 'pairing',
                  'bg-muted': session.status === 'disconnected',
                  'bg-error': session.status === 'error'
                }"
              />
            </div>
            <div class="min-w-0 flex-1">
              <div class="flex items-center justify-between gap-1">
                <span class="font-semibold text-sm text-highlighted truncate">{{ session.name }}</span>
                <UDropdownMenu :items="[dropdownItems(session)]" :content="{ align: 'end' }">
                  <UButton
                    icon="i-lucide-ellipsis-vertical"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    class="shrink-0 -mr-1"
                  />
                </UDropdownMenu>
              </div>
              <div class="flex items-center gap-1.5 text-xs text-muted mt-0.5">
                <span v-if="session.businessName || session.pushName" class="truncate">{{ session.businessName || session.pushName }}</span>
                <span v-if="session.businessName || session.pushName" class="text-dimmed">·</span>
                <UBadge
                  :color="sessionStatusColor(session.status)"
                  variant="subtle"
                  size="xs"
                  class="capitalize"
                >
                  {{ session.status }}
                </UBadge>
                <UBadge
                  v-if="session.engine === 'cloud_api'"
                  color="info"
                  variant="subtle"
                  size="xs"
                >
                  Cloud API
                </UBadge>
                <UBadge
                  v-if="session.chatwootEnabled"
                  color="warning"
                  variant="subtle"
                  size="xs"
                >
                  Chatwoot
                </UBadge>
              </div>
            </div>
          </div>

          <div class="border-t border-default px-4 py-2">
            <div class="flex flex-wrap items-center gap-x-3 gap-y-1 text-xs text-muted">
              <span v-if="session.jid" class="flex items-center gap-1">
                <UIcon name="i-lucide-phone" class="size-3" />
                +{{ parseJID(session.jid).phone }}
              </span>
              <span v-else class="italic">Not paired</span>
              <span v-if="session.platform" class="flex items-center gap-1 capitalize">
                <UIcon :name="platformIcon(session.platform)" class="size-3" />
                {{ session.platform }}
              </span>
              <span v-if="session.proxy?.host" class="flex items-center gap-1">
                <UIcon name="i-lucide-shield" class="size-3" />
                {{ session.proxy.host }}
              </span>
              <span v-if="session.createdAt" class="flex items-center gap-1">
                <UIcon name="i-lucide-clock" class="size-3" />
                {{ timeAgo(session.createdAt) }}
              </span>
            </div>
          </div>

          <div class="border-t border-default px-4 py-2 flex items-center gap-1.5">
            <UButton
              icon="i-lucide-arrow-right"
              label="Open"
              size="xs"
              color="primary"
              variant="soft"
              class="flex-1"
              :to="`/sessions/${session.id}`"
            />
            <UButton
              v-if="session.status !== 'connected'"
              icon="i-lucide-plug"
              size="xs"
              color="neutral"
              variant="ghost"
              @click="connectSession(session)"
            />
            <UButton
              v-else
              icon="i-lucide-unplug"
              size="xs"
              color="warning"
              variant="ghost"
              @click="disconnectSession(session.id)"
            />
            <UButton
              icon="i-lucide-trash-2"
              size="xs"
              color="error"
              variant="ghost"
              @click="deleteSession(session.id)"
            />
          </div>
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
