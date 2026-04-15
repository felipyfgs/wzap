<script setup lang="ts">
import type { Status } from '~/composables/useStatus'

const route = useRoute()
const sessionId = computed(() => route.params.id as string)
const { fetchStatuses, fetchContactStatuses } = useStatus()
const toast = useToast()

const loading = ref(false)
const statuses = ref<Status[]>([])
const groupedBySender = ref<Map<string, Status[]>>(new Map())
const viewModalOpen = ref(false)
const viewStatuses = ref<Status[]>([])
const viewIndex = ref(0)
const viewSenderName = ref('')

async function loadStatuses() {
  loading.value = true
  try {
    statuses.value = await fetchStatuses(sessionId.value)
    const groups = new Map<string, Status[]>()
    for (const s of statuses.value) {
      const key = s.senderJid
      if (!groups.has(key)) groups.set(key, [])
      groups.get(key)!.push(s)
    }
    groupedBySender.value = groups
  } catch {
    toast.add({ title: 'Failed to load statuses', color: 'error' })
  } finally {
    loading.value = false
  }
}

function getSenderName(senderJid: string): string {
  const statuses = groupedBySender.value.get(senderJid)
  return statuses?.[0]?.senderName || senderJid.split('@')[0]
}

async function openView(senderJid: string) {
  try {
    const contactStatuses = await fetchContactStatuses(sessionId.value, senderJid)
    if (contactStatuses.length === 0) return
    viewStatuses.value = contactStatuses
    viewIndex.value = 0
    viewSenderName.value = contactStatuses[0]?.senderName || getSenderName(senderJid)
    viewModalOpen.value = true
  } catch {
    toast.add({ title: 'Failed to load contact statuses', color: 'error' })
  }
}

function getLatestStatus(senderJid: string): Status | undefined {
  return groupedBySender.value.get(senderJid)?.[0]
}

function getSenderJids(): string[] {
  return Array.from(groupedBySender.value.keys())
}

onMounted(() => loadStatuses())
</script>

<template>
  <UDashboardPanel id="session-status">
    <template #header>
      <UDashboardNavbar title="Status">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

        <template #right>
          <SessionsStatusSendModal :session-id="sessionId" @sent="loadStatuses" />
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            size="sm"
            :loading="loading"
            @click="loadStatuses"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="loading && statuses.length === 0" class="flex items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <div v-else-if="statuses.length === 0" class="flex flex-col items-center justify-center py-24 gap-3 text-muted">
        <UIcon name="i-lucide-circle-dot" class="size-10" />
        <p class="text-sm">No statuses available</p>
      </div>

      <div v-else class="space-y-1">
        <SessionsStatusStoryCard
          v-for="senderJid in getSenderJids()"
          :key="senderJid"
          :sender-jid="senderJid"
          :sender-name="getSenderName(senderJid)"
          :latest-status="getLatestStatus(senderJid)!"
          :has-unviewed="true"
          @click="openView(senderJid)"
        />
      </div>

      <!-- View Modal -->
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