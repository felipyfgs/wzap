<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/vue-table'

const route = useRoute()
const { api } = useWzap()
const toast = useToast()

const sessionId = computed(() => route.params.id as string)
const { archiveChat, muteChat, pinChat, deleteChat, markRead, markUnread } = useChatOperations(sessionId)

const labelModalOpen = ref(false)
const labelMode = ref<'add-chat' | 'remove-chat' | 'add-message' | 'remove-message'>('add-chat')
const labelTargetJid = ref('')
const labelTargetMessageId = ref('')

interface Message {
  id: string
  chatJid: string
  senderJid: string
  fromMe: boolean
  msgType: string
  body: string
  timestamp: string
}

const messages = ref<Message[]>([])
const loading = ref(true)
const pagination = ref({ pageIndex: 0, pageSize: 50 })
const table = useTemplateRef('table')

async function fetchMessages() {
  loading.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}/messages?limit=200&offset=0`)
    messages.value = res.data || []
  } catch {
    messages.value = []
  }
  loading.value = false
}

const columns: TableColumn<Message>[] = [{
  accessorKey: 'timestamp',
  header: 'Time',
  cell: ({ row }) => new Date(row.original.timestamp).toLocaleString()
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
}, {
  id: 'actions',
  header: '',
  cell: ({ row }) => {
    const items = [
      { label: 'Archive chat', icon: 'i-lucide-archive', onSelect: () => wrapAction(() => archiveChat(row.original.chatJid)) },
      { label: 'Mute chat', icon: 'i-lucide-bell-off', onSelect: () => wrapAction(() => muteChat(row.original.chatJid)) },
      { label: 'Pin chat', icon: 'i-lucide-pin', onSelect: () => wrapAction(() => pinChat(row.original.chatJid)) },
      { label: 'Mark read', icon: 'i-lucide-check-check', onSelect: () => wrapAction(() => markRead(row.original.chatJid, [row.original.id])) },
      { label: 'Mark unread', icon: 'i-lucide-eye-off', onSelect: () => wrapAction(() => markUnread(row.original.chatJid)) },
      { type: 'separator' as const },
      { label: 'Add label to chat', icon: 'i-lucide-tag', onSelect: () => { labelMode.value = 'add-chat'; labelTargetJid.value = row.original.chatJid; labelModalOpen.value = true } },
      { label: 'Remove label from chat', icon: 'i-lucide-tag-off', onSelect: () => { labelMode.value = 'remove-chat'; labelTargetJid.value = row.original.chatJid; labelModalOpen.value = true } },
      { label: 'Add label to message', icon: 'i-lucide-bookmark', onSelect: () => { labelMode.value = 'add-message'; labelTargetJid.value = row.original.chatJid; labelTargetMessageId.value = row.original.id; labelModalOpen.value = true } },
      { label: 'Remove label from message', icon: 'i-lucide-bookmark-minus', onSelect: () => { labelMode.value = 'remove-message'; labelTargetJid.value = row.original.chatJid; labelTargetMessageId.value = row.original.id; labelModalOpen.value = true } },
      { type: 'separator' as const },
      { label: 'Delete chat', icon: 'i-lucide-trash', color: 'error' as const, onSelect: () => wrapAction(() => deleteChat(row.original.chatJid)) }
    ]
    return h(resolveComponent('UDropdownMenu'), { items, content: { align: 'end' } }, () =>
      h(resolveComponent('UButton'), { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', size: 'xs' })
    )
  }
}]

async function wrapAction(fn: () => Promise<void>) {
  try {
    await fn()
    toast.add({ title: 'Done', color: 'success' })
  } catch {
    toast.add({ title: 'Action failed', color: 'error' })
  }
}

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
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            size="sm"
            @click="fetchMessages"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <UTable
        ref="table"
        v-model:pagination="pagination"
        :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
        :columns="columns"
        :data="messages"
        :loading="loading"
        class="shrink-0"
        :ui="{
          base: 'table-fixed border-separate border-spacing-0',
          thead: '[&>tr]:bg-elevated/50 [&>tr]:after:content-none',
          tbody: '[&>tr]:last:[&>td]:border-b-0',
          th: 'py-2 first:rounded-l-lg last:rounded-r-lg border-y border-default first:border-l last:border-r',
          td: 'border-b border-default',
          separator: 'h-0'
        }"
      >
        <template #empty>
          <div class="flex flex-col items-center justify-center py-16 gap-3 text-muted">
            <UIcon name="i-lucide-message-square" class="size-8" />
            <p class="text-sm">
              No messages found.
            </p>
          </div>
        </template>
      </UTable>

      <div class="flex items-center justify-between gap-3 border-t border-default pt-4 mt-auto">
        <span class="text-sm text-muted">Showing last 200 messages</span>
        <UPagination
          :page="(table?.tableApi?.getState().pagination.pageIndex || 0) + 1"
          :items-per-page="table?.tableApi?.getState().pagination.pageSize"
          :total="table?.tableApi?.getFilteredRowModel().rows.length"
          @update:page="(p: number) => table?.tableApi?.setPageIndex(p - 1)"
        />
      </div>

      <SessionsLabelActionModal
        v-model:open="labelModalOpen"
        :session-id="sessionId"
        :jid="labelTargetJid"
        :mode="labelMode"
        :message-id="labelTargetMessageId"
      />
    </template>
  </UDashboardPanel>
</template>
