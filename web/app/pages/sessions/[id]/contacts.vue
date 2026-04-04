<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/vue-table'

const route = useRoute()
const { api } = useWzap()
const toast = useToast()

const sessionId = computed(() => route.params.id as string)
const { archiveChat, muteChat, pinChat, deleteChat, markUnread } = useChatOperations(sessionId)

const labelModalOpen = ref(false)
const labelMode = ref<'add-chat' | 'remove-chat'>('add-chat')
const labelTargetJid = ref('')

const infoSlideoverOpen = ref(false)
const infoTargetJid = ref('')

interface Contact {
  jid: string
  name?: string
  pushName?: string
  businessName?: string
}

const contacts = ref<Contact[]>([])
const blockedList = ref<string[]>([])
const loading = ref(true)
const loadingBlocked = ref(false)
const search = ref('')
const contactFilter = ref('saved')
const viewTab = ref('contacts')
const table = useTemplateRef('table')

const filterOptions = [
  { label: 'Saved contacts', value: 'saved' },
  { label: 'All contacts', value: 'all' }
]

const viewOptions = [
  { label: 'Contacts', value: 'contacts' },
  { label: 'Blocked', value: 'blocked' }
]

const pagination = ref({
  pageIndex: 0,
  pageSize: 20
})

async function fetchContacts() {
  loading.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}/contacts?filter=${contactFilter.value}`)
    contacts.value = res.data || []
  } catch {
    contacts.value = []
  }
  loading.value = false
}

async function fetchBlockedList() {
  loadingBlocked.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}/contacts/blocklist`)
    blockedList.value = res.data || []
  } catch {
    blockedList.value = []
  }
  loadingBlocked.value = false
}

async function blockContact(jid: string) {
  try {
    const phone = jid.split('@')[0]
    await api(`/sessions/${sessionId.value}/contacts/block`, {
      method: 'POST',
      body: { phone }
    })
    toast.add({ title: 'Contact blocked', color: 'success' })
    if (viewTab.value === 'blocked') await fetchBlockedList()
  } catch {
    toast.add({ title: 'Failed to block contact', color: 'error' })
  }
}

async function unblockContact(jid: string) {
  try {
    const phone = jid.split('@')[0]
    await api(`/sessions/${sessionId.value}/contacts/unblock`, {
      method: 'POST',
      body: { phone }
    })
    toast.add({ title: 'Contact unblocked', color: 'success' })
    await fetchBlockedList()
  } catch {
    toast.add({ title: 'Failed to unblock contact', color: 'error' })
  }
}

async function checkContact(jid: string) {
  try {
    const phone = jid.split('@')[0]
    const res: any = await api(`/sessions/${sessionId.value}/contacts/check`, {
      method: 'POST',
      body: { phones: [phone] }
    })
    const results = res.data || []
    const found = results.find((r: any) => r.isRegistered || r.IsRegistered)
    toast.add({
      title: found ? 'Registered on WhatsApp' : 'Not registered',
      color: found ? 'success' : 'warning'
    })
  } catch {
    toast.add({ title: 'Failed to check contact', color: 'error' })
  }
}

async function subscribePresence(jid: string) {
  try {
    const phone = jid.split('@')[0]
    await api(`/sessions/${sessionId.value}/contacts/presence`, {
      method: 'POST',
      body: { phone }
    })
    toast.add({ title: 'Subscribed to presence', color: 'success' })
  } catch {
    toast.add({ title: 'Failed to subscribe to presence', color: 'error' })
  }
}

const filtered = computed(() => {
  const q = search.value.toLowerCase()
  if (!q) return contacts.value
  return contacts.value.filter(c =>
    c.jid.toLowerCase().includes(q)
    || c.name?.toLowerCase().includes(q)
    || c.pushName?.toLowerCase().includes(q)
    || c.businessName?.toLowerCase().includes(q)
  )
})

watch(search, () => {
  pagination.value.pageIndex = 0
})

watch(contactFilter, () => {
  pagination.value.pageIndex = 0
  fetchContacts()
})

watch(viewTab, (tab) => {
  if (tab === 'blocked') fetchBlockedList()
})

async function wrapAction(fn: () => Promise<void>, label: string) {
  try {
    await fn()
    toast.add({ title: label, color: 'success' })
  } catch {
    toast.add({ title: `Failed: ${label}`, color: 'error' })
  }
}

function getContactActions(c: Contact) {
  return [
    [
      { label: 'View Info', icon: 'i-lucide-info', onSelect: () => { infoTargetJid.value = c.jid; infoSlideoverOpen.value = true } },
      { label: 'Check WhatsApp', icon: 'i-lucide-search-check', onSelect: () => checkContact(c.jid) },
      { label: 'Subscribe Presence', icon: 'i-lucide-eye', onSelect: () => subscribePresence(c.jid) }
    ],
    [
      { label: 'Archive chat', icon: 'i-lucide-archive', onSelect: () => wrapAction(() => archiveChat(c.jid), 'Chat archived') },
      { label: 'Mute chat', icon: 'i-lucide-bell-off', onSelect: () => wrapAction(() => muteChat(c.jid), 'Chat muted') },
      { label: 'Pin chat', icon: 'i-lucide-pin', onSelect: () => wrapAction(() => pinChat(c.jid), 'Chat pinned') },
      { label: 'Mark unread', icon: 'i-lucide-eye-off', onSelect: () => wrapAction(() => markUnread(c.jid), 'Marked unread') }
    ],
    [
      { label: 'Add label', icon: 'i-lucide-tag', onSelect: () => { labelMode.value = 'add-chat'; labelTargetJid.value = c.jid; labelModalOpen.value = true } },
      { label: 'Remove label', icon: 'i-lucide-tag-off', onSelect: () => { labelMode.value = 'remove-chat'; labelTargetJid.value = c.jid; labelModalOpen.value = true } }
    ],
    [
      { label: 'Delete chat', icon: 'i-lucide-trash', color: 'error', onSelect: () => wrapAction(() => deleteChat(c.jid), 'Chat deleted') },
      { label: 'Block', icon: 'i-lucide-ban', color: 'error', onSelect: () => blockContact(c.jid) }
    ]
  ]
}

const UButton = resolveComponent('UButton')
const UDropdownMenu = resolveComponent('UDropdownMenu')

const columns = computed<TableColumn<Contact>[]>(() => [{
  accessorKey: 'name',
  header: 'Name',
  cell: ({ row }) => row.original.name || '—'
}, {
  accessorKey: 'pushName',
  header: 'Push Name',
  cell: ({ row }) => row.original.pushName || '—'
}, {
  accessorKey: 'businessName',
  header: 'Business',
  cell: ({ row }) => row.original.businessName || '—'
}, {
  accessorKey: 'jid',
  header: 'JID'
}, {
  id: 'actions',
  header: '',
  cell: ({ row }) => {
    return h('div', { class: 'text-right' },
      h(UDropdownMenu, { items: getContactActions(row.original), content: { align: 'end' } }, () =>
        h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', size: 'xs' })
      )
    )
  }
}])

watch(sessionId, fetchContacts, { immediate: true })
</script>

<template>
  <UDashboardPanel id="session-contacts">
    <template #header>
      <UDashboardNavbar title="Contacts">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <SessionsCheckNumberModal :session-id="sessionId" />
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            size="sm"
            @click="fetchContacts"
          />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <template #left>
          <UInput
            v-model="search"
            icon="i-lucide-search"
            placeholder="Search contacts…"
            class="max-w-xs"
          />
          <USelect
            v-model="contactFilter"
            :items="filterOptions"
            value-key="value"
            class="min-w-36"
          />
          <USelect
            v-model="viewTab"
            :items="viewOptions"
            value-key="value"
            class="w-32"
          />
        </template>
        <template #right>
          <span class="text-sm text-muted">{{ viewTab === 'contacts' ? `${filtered.length} contact(s)` : `${blockedList.length} blocked` }}</span>
        </template>
      </UDashboardToolbar>
    </template>

    <template #body>
      <!-- Contacts view -->
      <template v-if="viewTab === 'contacts'">
        <UTable
          ref="table"
          v-model:pagination="pagination"
          :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
          :columns="columns"
          :data="filtered"
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
              <UIcon name="i-lucide-users" class="size-8" />
              <p class="text-sm">
                No contacts found.
              </p>
            </div>
          </template>
        </UTable>

        <div class="flex items-center justify-between gap-3 border-t border-default pt-4 mt-auto">
          <span class="text-sm text-muted">{{ filtered.length }} contact(s)</span>
          <UPagination
            :page="(table?.tableApi?.getState().pagination.pageIndex || 0) + 1"
            :items-per-page="table?.tableApi?.getState().pagination.pageSize"
            :total="filtered.length"
            @update:page="(p: number) => table?.tableApi?.setPageIndex(p - 1)"
          />
        </div>
      </template>

      <!-- Blocked view -->
      <template v-else>
        <div v-if="loadingBlocked" class="flex items-center justify-center py-24">
          <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
        </div>

        <div v-else-if="!blockedList.length" class="flex flex-col items-center justify-center py-16 gap-3 text-muted">
          <UIcon name="i-lucide-ban" class="size-8" />
          <p class="text-sm">
            No blocked contacts.
          </p>
        </div>

        <div v-else class="space-y-1">
          <div v-for="jid in blockedList" :key="jid" class="flex items-center justify-between py-2 px-1">
            <span class="text-sm font-mono truncate">{{ jid }}</span>
            <UButton
              icon="i-lucide-unlock"
              label="Unblock"
              size="xs"
              color="neutral"
              variant="outline"
              @click="unblockContact(jid)"
            />
          </div>
        </div>
      </template>

      <SessionsLabelActionModal
        v-model:open="labelModalOpen"
        :session-id="sessionId"
        :jid="labelTargetJid"
        :mode="labelMode"
      />

      <SessionsContactInfoSlideover
        v-model:open="infoSlideoverOpen"
        :session-id="sessionId"
        :contact-jid="infoTargetJid"
      />
    </template>
  </UDashboardPanel>
</template>
