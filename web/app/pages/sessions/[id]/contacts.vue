<script setup lang="ts">
import type { TableColumn } from '@nuxt/ui'

const route = useRoute()
const { api } = useWzap()

const sessionId = computed(() => route.params.id as string)

interface Contact {
  jid: string
  pushName: string
  businessName?: string
}

const contacts = ref<Contact[]>([])
const loading = ref(true)
const search = ref('')

async function fetchContacts() {
  loading.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}/contacts`)
    contacts.value = res.data || []
  } catch {
    contacts.value = []
  }
  loading.value = false
}

const filtered = computed(() => {
  const q = search.value.toLowerCase()
  if (!q) return contacts.value
  return contacts.value.filter(c =>
    c.jid.toLowerCase().includes(q) || c.pushName?.toLowerCase().includes(q)
  )
})

const columns: TableColumn<Contact>[] = [{
  accessorKey: 'jid',
  header: 'JID'
}, {
  accessorKey: 'pushName',
  header: 'Name',
  cell: ({ row }) => row.original.pushName || '—'
}, {
  accessorKey: 'businessName',
  header: 'Business',
  cell: ({ row }) => row.original.businessName || '—'
}]

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
          <UButton icon="i-lucide-refresh-cw" color="neutral" variant="ghost" size="sm" @click="fetchContacts" />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <template #left>
          <UInput v-model="search" icon="i-lucide-search" placeholder="Search contacts…" class="max-w-xs" />
        </template>
        <template #right>
          <span class="text-sm text-muted">{{ filtered.length }} contact(s)</span>
        </template>
      </UDashboardToolbar>
    </template>

    <template #body>
      <UTable :columns="columns" :data="filtered" :loading="loading" class="w-full">
        <template #empty>
          <div class="flex flex-col items-center justify-center py-16 gap-3 text-muted">
            <UIcon name="i-lucide-users" class="size-8" />
            <p class="text-sm">No contacts found.</p>
          </div>
        </template>
      </UTable>
    </template>
  </UDashboardPanel>
</template>
