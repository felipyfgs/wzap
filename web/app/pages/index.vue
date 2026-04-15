<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { Session } from '~/types'
import { sessionStatusColor } from '~/utils'

const { api } = useWzap()

const sessions = ref<Session[]>([])
const health = ref<{ status: string, services: Record<string, boolean> } | null>(null)
const loading = ref(false)

async function fetchData() {
  loading.value = true
  try {
    const [sessionsRes, healthRes]: [{ data: unknown }, { data: unknown }] = await Promise.all([
      api('/sessions'),
      api('/health')
    ])
    sessions.value = sessionsRes.data || []
    health.value = healthRes.data || null
  } catch {
    sessions.value = []
  }
  loading.value = false
}

const stats = computed(() => {
  const total = sessions.value.length
  const connected = sessions.value.filter(s => s.status === 'connected').length
  const disconnected = sessions.value.filter(s => s.status === 'disconnected' || s.status === 'error').length
  return { total, connected, disconnected }
})

function statusColor(status: string) {
  return sessionStatusColor(status)
}

const cardUi = {
  container: 'gap-y-1.5',
  wrapper: 'items-start',
  leading: 'p-2.5 rounded-full bg-primary/10 ring ring-inset ring-primary/25',
  title: 'font-normal text-muted text-xs uppercase'
}

const columns: TableColumn<Session>[] = [{
  accessorKey: 'name',
  header: 'Name',
  cell: ({ row }) => h('div', [
    h('p', { class: 'font-medium text-highlighted' }, row.original.name),
    h('p', { class: 'text-xs text-muted font-mono' }, row.original.id)
  ])
}, {
  accessorKey: 'status',
  header: 'Status'
}, {
  accessorKey: 'jid',
  header: 'Phone',
  cell: ({ row }) => row.original.jid?.replace(/@.*$/, '') || '—'
}, {
  id: 'actions',
  cell: ({ row }) => h('div', { class: 'flex justify-end gap-1' }, [
    h(resolveComponent('UButton'), {
      label: 'Open', icon: 'i-lucide-arrow-right', size: 'xs',
      color: 'neutral', variant: 'outline',
      to: `/sessions/${row.original.id}`
    })
  ])
}]

onMounted(fetchData)
</script>

<template>
  <UDashboardPanel id="home">
    <template #header>
      <UDashboardNavbar title="Dashboard" :ui="{ right: 'gap-2' }">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            size="sm"
            :loading="loading"
            @click="fetchData"
          />
          <UButton
            label="New Session"
            icon="i-lucide-plus"
            size="sm"
            color="primary"
            to="/sessions"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <div v-else class="space-y-6">
        <!-- Stats row -->
        <UPageGrid class="lg:grid-cols-4 gap-4 sm:gap-6 lg:gap-px">
          <UPageCard
            icon="i-lucide-smartphone"
            title="Sessions"
            variant="subtle"
            :ui="cardUi"
            class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
          >
            <span class="text-2xl font-semibold text-highlighted">{{ stats.total }}</span>
          </UPageCard>

          <UPageCard
            icon="i-lucide-wifi"
            title="Connected"
            variant="subtle"
            :ui="cardUi"
            class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
          >
            <span class="text-2xl font-semibold text-success">{{ stats.connected }}</span>
          </UPageCard>

          <UPageCard
            icon="i-lucide-wifi-off"
            title="Offline"
            variant="subtle"
            :ui="cardUi"
            class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
          >
            <span class="text-2xl font-semibold text-highlighted">{{ stats.disconnected }}</span>
          </UPageCard>

          <UPageCard
            icon="i-lucide-activity"
            title="API Status"
            variant="subtle"
            :ui="cardUi"
            class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
          >
            <UBadge
              v-if="health"
              :color="health.status === 'UP' ? 'success' : 'warning'"
              variant="subtle"
              size="lg"
            >
              {{ health.status }}
            </UBadge>
            <span v-else class="text-sm text-muted">—</span>
          </UPageCard>
        </UPageGrid>

        <!-- Sessions table -->
        <UCard>
          <template #header>
            <div class="flex items-center justify-between">
              <p class="font-semibold text-highlighted">
                Sessions
              </p>
              <UButton
                label="View all"
                icon="i-lucide-arrow-right"
                size="xs"
                color="neutral"
                variant="ghost"
                to="/sessions"
              />
            </div>
          </template>

          <UTable
            :columns="columns"
            :data="sessions"
            :loading="loading"
            class="w-full"
            :ui="TABLE_UI"
          >
            <template #status-cell="{ row }">
              <UBadge :color="statusColor(row.original.status)" variant="subtle" class="capitalize">
                {{ row.original.status }}
              </UBadge>
            </template>

            <template #empty>
              <div class="flex flex-col items-center justify-center py-12 gap-3 text-muted">
                <UIcon name="i-lucide-smartphone" class="size-8" />
                <p class="text-sm">
                  No sessions yet.
                </p>
                <UButton
                  label="Create Session"
                  icon="i-lucide-plus"
                  size="sm"
                  color="primary"
                  to="/sessions"
                />
              </div>
            </template>
          </UTable>
        </UCard>
      </div>
    </template>
  </UDashboardPanel>
</template>
