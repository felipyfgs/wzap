<script setup lang="ts">
import type { Status } from '~/composables/useStatus'

const route = useRoute()
const sessionId = computed(() => route.params.id as string)
const { fetchStatuses, fetchContactStatuses } = useStatus()
const toast = useToast()

const PAGE_SIZE = 100

const loading = ref(false)
const loadingMore = ref(false)
const hasMore = ref(true)
const offset = ref(0)
const groupedBySender = ref<Map<string, Status[]>>(new Map())
const viewedJids = ref<Set<string>>(new Set())
const viewModalOpen = ref(false)
const viewStatuses = ref<Status[]>([])
const viewIndex = ref(0)
const viewSenderName = ref('')
const sentinelRef = ref<HTMLElement | null>(null)

function mergeStatuses(list: Status[]) {
  const groups = groupedBySender.value
  for (const s of list) {
    if (!groups.has(s.senderJid)) groups.set(s.senderJid, [])
    const existing = groups.get(s.senderJid)!
    if (!existing.find(e => e.id === s.id)) existing.push(s)
  }
}

async function loadStatuses(reset = false) {
  if (reset) {
    groupedBySender.value = new Map()
    offset.value = 0
    hasMore.value = true
    viewedJids.value = new Set()
  }
  loading.value = true
  try {
    const batch = await fetchStatuses(sessionId.value, PAGE_SIZE, offset.value)
    mergeStatuses(batch)
    offset.value += batch.length
    hasMore.value = batch.length === PAGE_SIZE
  } catch {
    toast.add({ title: 'Failed to load statuses', color: 'error' })
  } finally {
    loading.value = false
  }
}

async function loadMore() {
  if (loadingMore.value || !hasMore.value) return
  loadingMore.value = true
  try {
    const batch = await fetchStatuses(sessionId.value, PAGE_SIZE, offset.value)
    mergeStatuses(batch)
    offset.value += batch.length
    hasMore.value = batch.length === PAGE_SIZE
  } catch {
    // silently fail on scroll load
  } finally {
    loadingMore.value = false
  }
}

function getSenderName(senderJid: string): string {
  const s = groupedBySender.value.get(senderJid)
  return s?.[0]?.senderName || senderJid.split('@')[0]
}

function getLatestStatus(senderJid: string): Status | undefined {
  return groupedBySender.value.get(senderJid)?.[0]
}

const unviewedJids = computed(() =>
  Array.from(groupedBySender.value.keys()).filter(j => !viewedJids.value.has(j))
)
const viewedJidsList = computed(() =>
  Array.from(groupedBySender.value.keys()).filter(j => viewedJids.value.has(j))
)

async function openView(senderJid: string) {
  try {
    const contactStatuses = await fetchContactStatuses(sessionId.value, senderJid)
    if (contactStatuses.length === 0) return
    viewStatuses.value = contactStatuses
    viewIndex.value = 0
    viewSenderName.value = contactStatuses[0]?.senderName || getSenderName(senderJid)
    viewModalOpen.value = true
    viewedJids.value.add(senderJid)
  } catch {
    toast.add({ title: 'Failed to load contact statuses', color: 'error' })
  }
}

onMounted(async () => {
  await loadStatuses(true)

  const observer = new IntersectionObserver(
    entries => { if (entries[0]?.isIntersecting) loadMore() },
    { threshold: 0.1 }
  )
  watchEffect(() => {
    if (sentinelRef.value) observer.observe(sentinelRef.value)
  })
  onUnmounted(() => observer.disconnect())
})
</script>

<template>
  <UDashboardPanel id="session-status">
    <template #header>
      <UDashboardNavbar title="Status">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <SessionsStatusSendModal :session-id="sessionId" @sent="loadStatuses(true)" />
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            size="sm"
            :loading="loading"
            @click="loadStatuses(true)"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="loading && groupedBySender.size === 0" class="flex items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <div v-else-if="groupedBySender.size === 0" class="flex flex-col items-center justify-center py-24 gap-3 text-muted">
        <UIcon name="i-lucide-circle-dot" class="size-10" />
        <p class="text-sm">Nenhum status disponível</p>
      </div>

      <div v-else>
        <!-- Atualizações recentes (não vistas) -->
        <div v-if="unviewedJids.length > 0">
          <p class="px-4 pt-3 pb-1 text-xs font-semibold text-muted uppercase tracking-wider">
            Atualizações recentes
          </p>
          <SessionsStatusStoryCard
            v-for="jid in unviewedJids"
            :key="jid"
            :session-id="sessionId"
            :sender-jid="jid"
            :sender-name="getSenderName(jid)"
            :latest-status="getLatestStatus(jid)!"
            :has-unviewed="true"
            @click="openView(jid)"
          />
        </div>

        <!-- Visualizados -->
        <div v-if="viewedJidsList.length > 0">
          <p class="px-4 pt-4 pb-1 text-xs font-semibold text-muted uppercase tracking-wider">
            Visualizados
          </p>
          <SessionsStatusStoryCard
            v-for="jid in viewedJidsList"
            :key="jid"
            :session-id="sessionId"
            :sender-jid="jid"
            :sender-name="getSenderName(jid)"
            :latest-status="getLatestStatus(jid)!"
            :has-unviewed="false"
            @click="openView(jid)"
          />
        </div>

        <!-- Sentinel de scroll infinito -->
        <div ref="sentinelRef" class="flex items-center justify-center py-4">
          <UIcon v-if="loadingMore" name="i-lucide-loader-2" class="size-5 animate-spin text-muted" />
          <span v-else-if="!hasMore && groupedBySender.size > 0" class="text-xs text-muted">
            Todos os status carregados
          </span>
        </div>
      </div>

      <!-- Modal de visualização -->
      <SessionsStatusViewModal
        v-if="viewModalOpen"
        :statuses="viewStatuses"
        :initial-index="viewIndex"
        :sender-name="viewSenderName"
        @close="viewModalOpen = false"
      />
    </template>
  </UDashboardPanel>
</template>