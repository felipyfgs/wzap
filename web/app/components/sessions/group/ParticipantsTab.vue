<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import type { Row } from '@tanstack/table-core'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { GroupDetail, GroupParticipant, JoinRequest } from './types'

const props = defineProps<{
  sessionId: string
  group: GroupDetail
  members: GroupParticipant[]
  joinRequests: JoinRequest[]
}>()
const emit = defineEmits<{ updated: [] }>()

const { api } = useWzap()
const toast = useToast()

const addPhone = ref('')
const addingParticipant = ref(false)
const participantSearch = ref('')
const roleFilter = ref<'all' | 'admins' | 'members'>('all')
const sorting = ref([{ id: 'participant', desc: false }])
const rowSelection = ref<Record<string, boolean>>({})
const pagination = ref({ pageIndex: 0, pageSize: 20 })

const roleFilterOptions = [
  { label: 'All', value: 'all' },
  { label: 'Admins', value: 'admins' },
  { label: 'Members', value: 'members' }
]

const page = computed({
  get: () => (pagination.value.pageIndex || 0) + 1,
  set: (v: number) => { pagination.value = { ...pagination.value, pageIndex: v - 1 } }
})

const table = useTemplateRef('table') as Ref<{ tableApi?: { getFilteredSelectedRowModel: () => { rows: Array<{ original: { jid: string } }> } } } | null>
const adminCount = computed(() => props.members.filter(p => p.isAdmin || p.isSuperAdmin).length)
const memberCount = computed(() => props.members.filter(p => !p.isAdmin && !p.isSuperAdmin).length)
const selectedCount = computed((): number => table.value?.tableApi?.getFilteredSelectedRowModel().rows.length ?? 0)
const selectedJids = computed((): string[] => table.value?.tableApi?.getFilteredSelectedRowModel().rows.map(r => r.original.jid) ?? [])

const filteredParticipants = computed(() => {
  let list = props.members
  if (roleFilter.value === 'admins') list = list.filter(p => p.isAdmin || p.isSuperAdmin)
  else if (roleFilter.value === 'members') list = list.filter(p => !p.isAdmin && !p.isSuperAdmin)
  const q = participantSearch.value.toLowerCase()
  if (q) list = list.filter(p => p.jid.toLowerCase().includes(q) || p.phoneNumber?.toLowerCase().includes(q) || p.displayName?.toLowerCase().includes(q))
  return list
})

watch([participantSearch, roleFilter], () => {
  pagination.value = { ...pagination.value, pageIndex: 0 }
  rowSelection.value = {}
})

watch(() => props.group.jid, () => {
  rowSelection.value = {}
  roleFilter.value = 'all'
  participantSearch.value = ''
  pagination.value = { ...pagination.value, pageIndex: 0 }
})

function formatPhone(jid: string): string {
  return '+' + jid.split('@')[0]
}

function getDisplayInfo(p: GroupParticipant): { name: string, description?: string, initials: string } {
  const phone = p.phoneNumber ? formatPhone(p.phoneNumber) : formatPhone(p.jid)
  const name = p.displayName || phone
  const description = p.displayName ? phone : undefined
  const initials = (p.displayName || (p.phoneNumber || p.jid).split('@')[0] || '??').slice(0, 2).toUpperCase()
  return { name, description, initials }
}

async function addParticipant() {
  if (!addPhone.value.trim()) return
  addingParticipant.value = true
  try {
    await api(`/sessions/${props.sessionId}/groups/participants`, {
      method: 'POST',
      body: { groupJid: props.group.jid, participants: [addPhone.value.trim()], action: 'add' }
    })
    toast.add({ title: 'Participant added', color: 'success' })
    addPhone.value = ''
    emit('updated')
  } catch {
    toast.add({ title: 'Failed to add participant', color: 'error' })
  }
  addingParticipant.value = false
}

async function participantAction(participantJid: string, action: string) {
  try {
    await api(`/sessions/${props.sessionId}/groups/participants`, {
      method: 'POST',
      body: { groupJid: props.group.jid, participants: [participantJid], action }
    })
    toast.add({ title: `Participant ${action}d`, color: 'success' })
    emit('updated')
  } catch {
    toast.add({ title: `Failed to ${action} participant`, color: 'error' })
  }
}

async function handleJoinRequest(jid: string, action: 'approve' | 'reject') {
  try {
    await api(`/sessions/${props.sessionId}/groups/requests/action`, {
      method: 'POST',
      body: { groupJid: props.group.jid, participants: [jid], action }
    })
    toast.add({ title: `Request ${action}d`, color: 'success' })
    emit('updated')
  } catch {
    toast.add({ title: `Failed to ${action} request`, color: 'error' })
  }
}

async function bulkAction(action: 'promote' | 'demote' | 'remove') {
  if (!selectedJids.value.length) return
  try {
    await api(`/sessions/${props.sessionId}/groups/participants`, {
      method: 'POST',
      body: { groupJid: props.group.jid, participants: selectedJids.value, action }
    })
    toast.add({ title: `${selectedJids.value.length} participant(s) ${action}d`, color: 'success' })
    rowSelection.value = {}
    emit('updated')
  } catch {
    toast.add({ title: `Failed to ${action} participants`, color: 'error' })
  }
}

function roleWeight(p: GroupParticipant) {
  if (p.isSuperAdmin) return 0
  if (p.isAdmin) return 1
  return 2
}

function getRowItems(row: Row<GroupParticipant>) {
  const p = row.original
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  const items: any[] = [
    { type: 'label', label: 'Actions' },
    {
      label: 'Copy JID',
      icon: 'i-lucide-copy',
      onSelect() {
        navigator.clipboard.writeText(p.jid)
        toast.add({ title: 'JID copied', color: 'success' })
      }
    },
    { type: 'separator' }
  ]
  if (p.isAdmin || p.isSuperAdmin) {
    items.push({ label: 'Demote to member', icon: 'i-lucide-arrow-down', onSelect: () => participantAction(p.jid, 'demote') })
  } else {
    items.push({ label: 'Promote to admin', icon: 'i-lucide-arrow-up', onSelect: () => participantAction(p.jid, 'promote') })
  }
  items.push({ type: 'separator' })
  items.push({ label: 'Remove from group', icon: 'i-lucide-user-x', color: 'error', onSelect: () => participantAction(p.jid, 'remove') })
  return items
}

const UBadge = resolveComponent('UBadge')
const UButton = resolveComponent('UButton')
const UDropdownMenu = resolveComponent('UDropdownMenu')
const UCheckbox = resolveComponent('UCheckbox')
const UUser = resolveComponent('UUser')

const participantColumns = computed<TableColumn<GroupParticipant>[]>(() => {
  const cols: TableColumn<GroupParticipant>[] = []

  if (props.group.isAdmin) {
    cols.push({
      id: 'select',
      header: ({ table }) =>
        h(UCheckbox, {
          'modelValue': table.getIsSomePageRowsSelected() ? 'indeterminate' : table.getIsAllPageRowsSelected(),
          'onUpdate:modelValue': (value: boolean | 'indeterminate') => table.toggleAllPageRowsSelected(!!value),
          'ariaLabel': 'Select all'
        }),
      cell: ({ row }) =>
        h(UCheckbox, {
          'modelValue': row.getIsSelected(),
          'onUpdate:modelValue': (value: boolean | 'indeterminate') => row.toggleSelected(!!value),
          'ariaLabel': 'Select row'
        }),
      enableSorting: false
    })
  }

  cols.push({
    id: 'participant',
    accessorFn: row => getDisplayInfo(row).name,
    header: ({ column }) => {
      const isSorted = column.getIsSorted()
      return h(UButton, {
        color: 'neutral', variant: 'ghost', label: 'Participant',
        icon: isSorted === 'asc' ? 'i-lucide-arrow-up-narrow-wide' : isSorted === 'desc' ? 'i-lucide-arrow-down-wide-narrow' : 'i-lucide-arrow-up-down',
        class: '-mx-2.5',
        onClick: () => column.toggleSorting(column.getIsSorted() === 'asc')
      })
    },
    enableSorting: true,
    cell: ({ row }) => {
      const { name, description, initials } = getDisplayInfo(row.original)
      return h(UUser, { name, description, avatar: { text: initials }, size: 'sm' })
    }
  }, {
    id: 'role',
    header: ({ column }) => {
      const isSorted = column.getIsSorted()
      return h(UButton, {
        color: 'neutral', variant: 'ghost', label: 'Role',
        icon: isSorted === 'asc' ? 'i-lucide-arrow-up-narrow-wide' : isSorted === 'desc' ? 'i-lucide-arrow-down-wide-narrow' : 'i-lucide-arrow-up-down',
        class: '-mx-2.5',
        onClick: () => column.toggleSorting(column.getIsSorted() === 'asc')
      })
    },
    enableSorting: true,
    sortingFn: (rowA, rowB) => roleWeight(rowA.original) - roleWeight(rowB.original),
    cell: ({ row }) => {
      const label = row.original.isSuperAdmin ? 'Owner' : row.original.isAdmin ? 'Admin' : 'Member'
      const color = row.original.isSuperAdmin ? 'info' : row.original.isAdmin ? 'success' : 'neutral'
      return h(UBadge, { label, color, variant: 'subtle', size: 'sm' })
    }
  })

  if (props.group.isAdmin) {
    cols.push({
      id: 'actions',
      header: '',
      enableSorting: false,
      cell: ({ row }) =>
        h('div', { class: 'text-right' },
          h(UDropdownMenu, { items: getRowItems(row), content: { align: 'end' } }, () =>
            h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', size: 'xs' })
          )
        )
    })
  }

  return cols
})
</script>

<template>
  <div class="space-y-3 pt-4">
    <div v-if="group.isAdmin" class="flex gap-2">
      <UInput
        v-model="addPhone"
        placeholder="5511999999999"
        size="sm"
        class="flex-1"
      />
      <UButton
        icon="i-lucide-user-plus"
        size="sm"
        color="neutral"
        :loading="addingParticipant"
        @click="addParticipant"
      />
    </div>

    <div class="flex gap-2">
      <UInput
        v-model="participantSearch"
        icon="i-lucide-search"
        placeholder="Search…"
        size="sm"
        class="flex-1"
      />
      <USelect
        v-model="roleFilter"
        :items="roleFilterOptions"
        value-key="value"
        size="sm"
        class="w-28"
      />
    </div>

    <p class="text-xs text-muted">
      {{ members.length }} total · {{ adminCount }} admins · {{ memberCount }} members
    </p>

    <div v-if="selectedCount > 0" class="flex items-center gap-2 rounded-lg bg-elevated px-3 py-2">
      <span class="text-xs text-muted">{{ selectedCount }} of {{ filteredParticipants.length }} selected</span>
      <UButton
        icon="i-lucide-x"
        size="xs"
        color="neutral"
        variant="ghost"
        @click="rowSelection = {}"
      />
      <USeparator orientation="vertical" class="h-4" />
      <UButton
        label="Promote"
        icon="i-lucide-arrow-up"
        size="xs"
        color="neutral"
        variant="outline"
        @click="bulkAction('promote')"
      />
      <UButton
        label="Demote"
        icon="i-lucide-arrow-down"
        size="xs"
        color="neutral"
        variant="outline"
        @click="bulkAction('demote')"
      />
      <UButton
        label="Remove"
        icon="i-lucide-user-x"
        size="xs"
        color="error"
        variant="soft"
        @click="bulkAction('remove')"
      />
    </div>

    <UTable
      ref="table"
      v-model:pagination="pagination"
      v-model:sorting="sorting"
      v-model:row-selection="rowSelection"
      :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
      :sorting-options="{}"
      :columns="participantColumns"
      :data="filteredParticipants"
      :get-row-id="(row: GroupParticipant) => row.jid"
      :ui="{ base: 'table-fixed border-separate border-spacing-0', th: 'py-1.5 text-xs', td: 'py-1.5 text-sm' }"
    />

    <div v-if="filteredParticipants.length > pagination.pageSize" class="flex items-center justify-between border-t border-default pt-3 mt-1">
      <span class="text-xs text-muted">
        Page {{ page }} of {{ Math.ceil(filteredParticipants.length / pagination.pageSize) }}
      </span>
      <UPagination
        v-model:page="page"
        :items-per-page="pagination.pageSize"
        :total="filteredParticipants.length"
      />
    </div>

    <template v-if="group.isAdmin && joinRequests.length">
      <USeparator />
      <p class="text-sm font-medium text-highlighted">
        Join Requests ({{ joinRequests.length }})
      </p>
      <div v-for="req in joinRequests" :key="req.jid" class="flex items-center justify-between py-1">
        <span class="text-sm font-mono truncate">{{ req.jid }}</span>
        <div class="flex gap-1">
          <UButton
            icon="i-lucide-check"
            size="xs"
            color="success"
            variant="soft"
            @click="handleJoinRequest(req.jid, 'approve')"
          />
          <UButton
            icon="i-lucide-x"
            size="xs"
            color="error"
            variant="soft"
            @click="handleJoinRequest(req.jid, 'reject')"
          />
        </div>
      </div>
    </template>
  </div>
</template>
