<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'

const route = useRoute()
const { api } = useWzap()

const sessionId = computed(() => route.params.id as string)

interface Message {
  id: string
  chatJid: string
  senderJid: string
  fromMe: boolean
  msgType: string
  body: string
  timestamp: number
}

const messages = ref<Message[]>([])
const loading = ref(true)
const page = ref(1)
const pageSize = 50

async function fetchMessages() {
  loading.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}/messages?limit=${pageSize}&offset=${(page.value - 1) * pageSize}`)
    messages.value = res.data || []
  } catch {
    messages.value = []
  }
  loading.value = false
}

const columns: TableColumn<Message>[] = [{
  accessorKey: 'timestamp',
  header: 'Time',
  cell: ({ row }) => new Date(row.original.timestamp * 1000).toLocaleString()
}, {
  accessorKey: 'chatJid',
  header: 'Chat'
}, {
  accessorKey: 'msgType',
  header: 'Type'
}, {
  accessorKey: 'body',
  header: 'Content',
  cell: ({ row }) => row.original.body?.slice(0, 80) || '—'
}, {
  accessorKey: 'fromMe',
  header: 'Direction',
  cell: ({ row }) => row.original.fromMe ? 'Sent' : 'Received'
}]

watch(sessionId, fetchMessages, { immediate: true })
</script>

<template>
  <UDashboardPanel id="session-messages">
    <template #header>
      <UDashboardNavbar title="Messages">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <SessionsSendMessageModal :session-id="sessionId" />
          <UButton icon="i-lucide-refresh-cw" color="neutral" variant="ghost" size="sm" @click="fetchMessages" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <UTable
        :columns="columns"
        :data="messages"
        :loading="loading"
        class="w-full"
      >
        <template #empty>
          <div class="flex flex-col items-center justify-center py-16 gap-3 text-muted">
            <UIcon name="i-lucide-message-square" class="size-8" />
            <p class="text-sm">No messages found.</p>
          </div>
        </template>
      </UTable>
    </template>
  </UDashboardPanel>
</template>
