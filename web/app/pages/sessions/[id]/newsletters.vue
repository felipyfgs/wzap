<script setup lang="ts">
const route = useRoute()
const toast = useToast()

const sessionId = computed(() => route.params.id as string)
const { listNewsletters, subscribe, unsubscribe, muteNewsletter, getInviteLink } = useNewsletter(sessionId)

const newsletters = ref<Record<string, unknown>[]>([])
const loading = ref(false)
const filter = ref('')

async function fetchNewsletters() {
  loading.value = true
  try {
    newsletters.value = await listNewsletters()
  } catch {
    newsletters.value = []
  }
  loading.value = false
}

const filtered = computed(() => {
  if (!filter.value) return newsletters.value
  const q = filter.value.toLowerCase()
  return newsletters.value.filter((n: Record<string, unknown>) =>
    (n.name || '').toLowerCase().includes(q) || (n.description || '').toLowerCase().includes(q)
  )
})

async function _handleSubscribe(jid: string) {
  try {
    await subscribe(jid)
    toast.add({ title: 'Subscribed', color: 'success' })
    await fetchNewsletters()
  } catch {
    toast.add({ title: 'Failed to subscribe', color: 'error' })
  }
}

async function handleUnsubscribe(jid: string) {
  try {
    await unsubscribe(jid)
    toast.add({ title: 'Unsubscribed', color: 'success' })
    await fetchNewsletters()
  } catch {
    toast.add({ title: 'Failed to unsubscribe', color: 'error' })
  }
}

async function handleMute(jid: string, mute: boolean) {
  try {
    await muteNewsletter(jid, mute)
    toast.add({ title: mute ? 'Muted' : 'Unmuted', color: 'success' })
    await fetchNewsletters()
  } catch {
    toast.add({ title: 'Failed to update mute', color: 'error' })
  }
}

async function copyInviteLink(jid: string) {
  try {
    const res = await getInviteLink(jid)
    const link = res?.link || res?.url || res
    if (typeof link === 'string') {
      await navigator.clipboard.writeText(link)
      toast.add({ title: 'Invite link copied', color: 'success' })
    }
  } catch {
    toast.add({ title: 'Failed to get invite link', color: 'error' })
  }
}

function dropdownItems(n: Record<string, unknown>) {
  return [
    { label: 'Copy invite link', icon: 'i-lucide-link', onSelect: () => copyInviteLink(n.jid) },
    { label: n.muted ? 'Unmute' : 'Mute', icon: n.muted ? 'i-lucide-bell' : 'i-lucide-bell-off', onSelect: () => handleMute(n.jid, !n.muted) },
    { type: 'separator' as const },
    { label: 'Unsubscribe', icon: 'i-lucide-log-out', color: 'error' as const, onSelect: () => handleUnsubscribe(n.jid) }
  ]
}

const createModal = useTemplateRef('createModal')

onMounted(() => fetchNewsletters())
watch(sessionId, fetchNewsletters)
</script>

<template>
  <UDashboardPanel id="session-newsletters">
    <template #header>
      <UDashboardNavbar title="Newsletters">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <SessionsNewsletterCreateModal ref="createModal" :session-id="sessionId" @created="fetchNewsletters" />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <template #left>
          <UInput
            v-model="filter"
            icon="i-lucide-search"
            placeholder="Filter newsletters..."
            class="max-w-xs"
          />
        </template>
        <template #right>
          <span class="text-sm text-muted">{{ filtered.length }} newsletter(s)</span>
        </template>
      </UDashboardToolbar>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <div v-else-if="filtered.length === 0" class="flex flex-col items-center justify-center py-24 gap-3 text-muted">
        <UIcon name="i-lucide-newspaper" class="size-10" />
        <p class="text-sm">
          No newsletters found.
        </p>
      </div>

      <div v-else class="grid grid-cols-1 sm:grid-cols-2 lg:grid-cols-3 gap-4">
        <UCard v-for="n in filtered" :key="n.jid" class="flex flex-col">
          <div class="flex items-start justify-between gap-2">
            <div class="min-w-0 flex-1">
              <p class="font-semibold text-sm text-highlighted truncate">
                {{ n.name || 'Unnamed' }}
              </p>
              <p v-if="n.description" class="text-xs text-muted line-clamp-2 mt-0.5">
                {{ n.description }}
              </p>
            </div>
            <UDropdownMenu :items="[dropdownItems(n)]" :content="{ align: 'end' }">
              <UButton
                icon="i-lucide-ellipsis-vertical"
                color="neutral"
                variant="ghost"
                size="xs"
              />
            </UDropdownMenu>
          </div>
          <div class="flex items-center gap-2 mt-2">
            <UBadge
              v-if="n.muted"
              color="neutral"
              variant="subtle"
              size="xs"
            >
              Muted
            </UBadge>
            <span v-if="n.subscriberCount != null" class="text-xs text-muted">{{ n.subscriberCount }} subscribers</span>
          </div>
        </UCard>
      </div>
    </template>
  </UDashboardPanel>
</template>
