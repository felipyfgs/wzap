<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/vue-table'

const route = useRoute()
const { api } = useWzap()

const sessionId = computed(() => route.params.id as string)
const { archiveChat, muteChat, pinChat, deleteChat, markRead, markUnread } = useChatOperations(sessionId)
const { wrapAction } = useActionWrapper()

const labelModalOpen = ref(false)
const labelMode = ref<'add-chat' | 'remove-chat' | 'add-message' | 'remove-message'>('add-chat')
const labelTargetJid = ref('')
const labelTargetMessageId = ref('')

const confirmDeleteChatOpen = ref(false)
const confirmDeleteChatJid = ref('')
const confirmDeleteChatModal = useTemplateRef('confirmDeleteChatModal')

const editModalOpen = ref(false)
const editTargetJid = ref('')
const editTargetId = ref('')

const reactModalOpen = ref(false)
const reactTargetJid = ref('')
const reactTargetId = ref('')

const forwardModalOpen = ref(false)
const forwardTargetJid = ref('')
const forwardTargetId = ref('')

const presenceModalOpen = ref(false)
const presenceTargetJid = ref('')

const confirmDeleteMsgOpen = ref(false)
const confirmDeleteMsgJid = ref('')
const confirmDeleteMsgId = ref('')
const confirmDeleteMsgModal = useTemplateRef('confirmDeleteMsgModal')

async function deleteMessage(chatJid: string, messageId: string) {
  await api(`/sessions/${sessionId.value}/messages/delete`, {
    method: 'POST',
    body: { phone: chatJid, id: messageId }
  })
}

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
const loading = ref(false)
const pagination = ref({ pageIndex: 0, pageSize: 50 })
const table = useTemplateRef('table')

async function fetchMessages() {
  loading.value = true
  try {
    const res: { data: unknown } = await api(`/sessions/${sessionId.value}/messages?limit=200&offset=0`)
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
      { label: 'Add label to chat', icon: 'i-lucide-tag', onSelect: () => {
        labelMode.value = 'add-chat'
        labelTargetJid.value = row.original.chatJid
        labelModalOpen.value = true
      } },
      { label: 'Remove label from chat', icon: 'i-lucide-tag-off', onSelect: () => {
        labelMode.value = 'remove-chat'
        labelTargetJid.value = row.original.chatJid
        labelModalOpen.value = true
      } },
      { label: 'Add label to message', icon: 'i-lucide-bookmark', onSelect: () => {
        labelMode.value = 'add-message'
        labelTargetJid.value = row.original.chatJid
        labelTargetMessageId.value = row.original.id
        labelModalOpen.value = true
      } },
      { label: 'Remove label from message', icon: 'i-lucide-bookmark-minus', onSelect: () => {
        labelMode.value = 'remove-message'
        labelTargetJid.value = row.original.chatJid
        labelTargetMessageId.value = row.original.id
        labelModalOpen.value = true
      } },
      { type: 'separator' as const },
      ...(row.original.fromMe
        ? [{ label: 'Edit message', icon: 'i-lucide-pencil', onSelect: () => {
            editTargetJid.value = row.original.chatJid
            editTargetId.value = row.original.id
            editModalOpen.value = true
          } }]
        : []),
      { label: 'React', icon: 'i-lucide-smile', onSelect: () => {
        reactTargetJid.value = row.original.chatJid
        reactTargetId.value = row.original.id
        reactModalOpen.value = true
      } },
      { label: 'Forward', icon: 'i-lucide-forward', onSelect: () => {
        forwardTargetJid.value = row.original.chatJid
        forwardTargetId.value = row.original.id
        forwardModalOpen.value = true
      } },
      { label: 'Set presence', icon: 'i-lucide-radio', onSelect: () => {
        presenceTargetJid.value = row.original.chatJid
        presenceModalOpen.value = true
      } },
      { type: 'separator' as const },
      { label: 'Delete message', icon: 'i-lucide-trash', color: 'error' as const, onSelect: () => {
        confirmDeleteMsgJid.value = row.original.chatJid
        confirmDeleteMsgId.value = row.original.id
        confirmDeleteMsgOpen.value = true
      } },
      { label: 'Delete chat', icon: 'i-lucide-trash', color: 'error' as const, onSelect: () => {
        confirmDeleteChatJid.value = row.original.chatJid
        confirmDeleteChatOpen.value = true
      } }
    ]
    return h(resolveComponent('UDropdownMenu'), { items, content: { align: 'end' } }, () =>
      h(resolveComponent('UButton'), { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', size: 'xs' })
    )
  }
}]

onMounted(() => fetchMessages())
watch(sessionId, fetchMessages)
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
        :ui="TABLE_UI"
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
      <SessionsConfirmModal
        ref="confirmDeleteChatModal"
        v-model:open="confirmDeleteChatOpen"
        title="Delete Chat"
        description="Are you sure you want to delete this chat? This action cannot be undone."
        confirm-label="Delete"
        confirm-color="error"
        icon="i-lucide-trash"
        @confirm="async () => { await wrapAction(() => deleteChat(confirmDeleteChatJid), { success: 'Chat deleted', error: 'Failed: Chat deleted' }); confirmDeleteChatModal?.done() }"
      />

      <SessionsConfirmModal
        ref="confirmDeleteMsgModal"
        v-model:open="confirmDeleteMsgOpen"
        title="Delete Message"
        description="Are you sure you want to delete this message?"
        confirm-label="Delete"
        confirm-color="error"
        icon="i-lucide-trash"
        @confirm="async () => { await wrapAction(() => deleteMessage(confirmDeleteMsgJid, confirmDeleteMsgId), { success: 'Message deleted', error: 'Failed: Message deleted' }); confirmDeleteMsgModal?.done() }"
      />

      <SessionsEditMessageModal
        v-model:open="editModalOpen"
        :session-id="sessionId"
        :chat-jid="editTargetJid"
        :message-id="editTargetId"
      />

      <SessionsReactMessageModal
        v-model:open="reactModalOpen"
        :session-id="sessionId"
        :chat-jid="reactTargetJid"
        :message-id="reactTargetId"
      />

      <SessionsForwardMessageModal
        v-model:open="forwardModalOpen"
        :session-id="sessionId"
        :chat-jid="forwardTargetJid"
        :message-id="forwardTargetId"
      />

      <SessionsPresenceModal
        v-model:open="presenceModalOpen"
        :session-id="sessionId"
        :chat-jid="presenceTargetJid"
      />
    </template>
  </UDashboardPanel>
</template>
