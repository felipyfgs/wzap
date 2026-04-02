<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'
import { getPaginationRowModel } from '@tanstack/table-core'
import type { Row } from '@tanstack/table-core'
import type { Webhook } from '~/types'

const UButton = resolveComponent('UButton')
const UBadge = resolveComponent('UBadge')
const UDropdownMenu = resolveComponent('UDropdownMenu')

const route = useRoute()
const { api } = useWzap()
const toast = useToast()
const table = useTemplateRef('table')

const sessionId = computed(() => route.params.id as string)
const webhooks = ref<Webhook[]>([])
const loading = ref(true)
const rowSelection = ref({})
const pagination = ref({ pageIndex: 0, pageSize: 10 })

async function fetchWebhooks() {
  loading.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}/webhooks`)
    webhooks.value = res.data || []
  } catch {
    webhooks.value = []
  }
  loading.value = false
}

async function toggleWebhook(wh: Webhook) {
  try {
    await api(`/sessions/${sessionId.value}/webhooks/${wh.id}`, {
      method: 'PUT',
      body: { enabled: !wh.enabled }
    })
    await fetchWebhooks()
  } catch {
    toast.add({ title: 'Failed to update webhook', color: 'error' })
  }
}

async function deleteWebhook(id: string) {
  try {
    await api(`/sessions/${sessionId.value}/webhooks/${id}`, { method: 'DELETE' })
    toast.add({ title: 'Webhook deleted', color: 'success' })
    await fetchWebhooks()
  } catch {
    toast.add({ title: 'Failed to delete webhook', color: 'error' })
  }
}

function getRowItems(row: Row<Webhook>) {
  return [
    {
      label: row.original.enabled ? 'Disable' : 'Enable',
      icon: row.original.enabled ? 'i-lucide-pause' : 'i-lucide-play',
      onSelect() { toggleWebhook(row.original) }
    },
    { type: 'separator' as const },
    {
      label: 'Delete',
      icon: 'i-lucide-trash',
      color: 'error' as const,
      onSelect() { deleteWebhook(row.original.id) }
    }
  ]
}

const columns: TableColumn<Webhook>[] = [
  {
    accessorKey: 'url',
    header: 'URL',
    cell: ({ row }) =>
      h('div', [
        h('p', { class: 'font-medium text-highlighted truncate max-w-xs' }, row.original.url),
        h('p', { class: 'text-sm text-muted' }, row.original.id)
      ])
  },
  {
    accessorKey: 'events',
    header: 'Events',
    cell: ({ row }) => {
      const evts = row.original.events || []
      return h('p', { class: 'text-sm text-muted' }, evts.join(', ') || 'All')
    }
  },
  {
    accessorKey: 'enabled',
    header: 'Status',
    cell: ({ row }) => {
      const color = row.original.enabled ? 'success' as const : 'neutral' as const
      return h(UBadge, { variant: 'subtle', color }, () => row.original.enabled ? 'Active' : 'Disabled')
    }
  },
  {
    id: 'actions',
    cell: ({ row }) =>
      h('div', { class: 'text-right' },
        h(UDropdownMenu, { content: { align: 'end' }, items: getRowItems(row) }, () =>
          h(UButton, { icon: 'i-lucide-ellipsis-vertical', color: 'neutral', variant: 'ghost', class: 'ml-auto' })
        )
      )
  }
]

watch(sessionId, fetchWebhooks, { immediate: true })
</script>

<template>
  <UDashboardPanel id="session-webhooks">
    <template #header>
      <UDashboardNavbar title="Webhooks">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <WebhooksAddModal :session-id="sessionId" @created="fetchWebhooks" />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <UTable
        ref="table"
        v-model:row-selection="rowSelection"
        v-model:pagination="pagination"
        :pagination-options="{ getPaginationRowModel: getPaginationRowModel() }"
        class="shrink-0"
        :data="webhooks"
        :columns="columns"
        :loading="loading"
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
            <UIcon name="i-lucide-webhook" class="size-8" />
            <p class="text-sm">No webhooks configured for this session.</p>
          </div>
        </template>
      </UTable>

      <div class="flex items-center justify-end gap-3 border-t border-default pt-4 mt-auto">
        <UPagination
          :default-page="(table?.tableApi?.getState().pagination.pageIndex || 0) + 1"
          :items-per-page="table?.tableApi?.getState().pagination.pageSize"
          :total="table?.tableApi?.getFilteredRowModel().rows.length"
          @update:page="(p: number) => table?.tableApi?.setPageIndex(p - 1)"
        />
      </div>
    </template>
  </UDashboardPanel>
</template>
