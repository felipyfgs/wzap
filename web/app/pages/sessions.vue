<script setup lang="ts">
const { api, isAuthenticated } = useWzap()

const sessions = ref<any[]>([])
const loading = ref(true)
const showCreate = ref(false)
const newName = ref('')
const creating = ref(false)

async function fetchSessions() {
  loading.value = true
  try {
    const res: any = await api('/sessions')
    sessions.value = res.data || []
  } catch {
    sessions.value = []
  }
  loading.value = false
}

async function createSession() {
  creating.value = true
  try {
    await api('/sessions', {
      method: 'POST',
      body: { name: newName.value }
    })
    newName.value = ''
    showCreate.value = false
    await fetchSessions()
  } catch {
    // handle error
  }
  creating.value = false
}

async function connectSession(id: string) {
  try {
    await api(`/sessions/${id}/connect`, { method: 'POST' })
    await fetchSessions()
  } catch {
    // handle error
  }
}

async function disconnectSession(id: string) {
  try {
    await api(`/sessions/${id}/disconnect`, { method: 'POST' })
    await fetchSessions()
  } catch {
    // handle error
  }
}

async function deleteSession(id: string) {
  try {
    await api(`/sessions/${id}`, { method: 'DELETE' })
    await fetchSessions()
  } catch {
    // handle error
  }
}

onMounted(() => {
  if (!isAuthenticated.value) {
    navigateTo('/login')
    return
  }
  fetchSessions()
})
</script>

<template>
  <UDashboardPanel id="sessions">
    <template #header>
      <UDashboardNavbar title="Sessions">
        <template #right>
          <UButton icon="i-lucide-plus" label="New Session" color="primary" @click="showCreate = true" />
        </template>
      </UDashboardNavbar>
    </template>

    <div class="p-4 space-y-4">
      <UCard v-if="showCreate">
        <form @submit.prevent="createSession" class="flex items-end gap-3">
          <UFormField label="Session Name" class="flex-1">
            <UInput v-model="newName" placeholder="my-session" />
          </UFormField>
          <UButton type="submit" :loading="creating" color="primary">Create</UButton>
          <UButton variant="ghost" @click="showCreate = false">Cancel</UButton>
        </form>
      </UCard>

      <div v-if="loading" class="text-center py-8">
        <UIcon name="i-lucide-loader-2" class="animate-spin text-2xl" />
      </div>

      <UCard v-for="session in sessions" :key="session.id">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="font-semibold text-lg">{{ session.name }}</h3>
            <p class="text-sm text-(--ui-text-muted)">
              {{ session.id }}
              <span v-if="session.jid"> &middot; {{ session.jid }}</span>
            </p>
          </div>
          <div class="flex items-center gap-2">
            <UBadge :color="session.status === 'connected' ? 'success' : 'error'" variant="subtle">
              {{ session.status }}
            </UBadge>
            <UButton
              v-if="session.status !== 'connected'"
              icon="i-lucide-plug"
              size="sm"
              variant="soft"
              color="primary"
              @click="connectSession(session.id)"
            >
              Connect
            </UButton>
            <UButton
              v-else
              icon="i-lucide-unplug"
              size="sm"
              variant="soft"
              color="warning"
              @click="disconnectSession(session.id)"
            >
              Disconnect
            </UButton>
            <UButton
              icon="i-lucide-trash-2"
              size="sm"
              variant="soft"
              color="error"
              @click="deleteSession(session.id)"
            />
          </div>
        </div>
      </UCard>

      <p v-if="!loading && sessions.length === 0" class="text-center text-(--ui-text-muted) py-8">
        No sessions yet. Create one to get started.
      </p>
    </div>
  </UDashboardPanel>
</template>
