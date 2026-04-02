<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'

const route = useRoute()
const { api } = useWzap()

const sessionId = computed(() => route.params.id as string)

interface Group {
  jid: string
  name: string
  topic?: string
  participants?: unknown[]
}

const groups = ref<Group[]>([])
const loading = ref(true)
const search = ref('')

async function fetchGroups() {
  loading.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}/groups`)
    groups.value = res.data || []
  } catch {
    groups.value = []
  }
  loading.value = false
}

const filtered = computed(() => {
  const q = search.value.toLowerCase()
  if (!q) return groups.value
  return groups.value.filter(g =>
    g.name?.toLowerCase().includes(q) || g.jid.toLowerCase().includes(q)
  )
})

const columns: TableColumn<Group>[] = [{
  accessorKey: 'name',
  header: 'Name',
  cell: ({ row }) => row.original.name || '—'
}, {
  accessorKey: 'jid',
  header: 'JID'
}, {
  accessorKey: 'participants',
  header: 'Participants',
  cell: ({ row }) => row.original.participants?.length ?? '—'
}, {
  accessorKey: 'topic',
  header: 'Topic',
  cell: ({ row }) => row.original.topic?.slice(0, 60) || '—'
}]

watch(sessionId, fetchGroups, { immediate: true })
</script>

<template>
  <UDashboardPanel id="session-groups">
    <template #header>
      <UDashboardNavbar title="Groups">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <UButton icon="i-lucide-refresh-cw" color="neutral" variant="ghost" size="sm" @click="fetchGroups" />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <template #left>
          <UInput v-model="search" icon="i-lucide-search" placeholder="Search groups…" class="max-w-xs" />
        </template>
        <template #right>
          <span class="text-sm text-muted">{{ filtered.length }} group(s)</span>
        </template>
      </UDashboardToolbar>
    </template>

    <template #body>
      <UTable :columns="columns" :data="filtered" :loading="loading" class="w-full">
        <template #empty>
          <div class="flex flex-col items-center justify-center py-16 gap-3 text-muted">
            <UIcon name="i-lucide-users-2" class="size-8" />
            <p class="text-sm">No groups found.</p>
          </div>
        </template>
      </UTable>
    </template>
  </UDashboardPanel>
</template>
