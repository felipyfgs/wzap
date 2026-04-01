<script setup lang="ts">
const { api, isAuthenticated } = useWzap()

const sessions = ref<any[]>([])
const selectedSession = ref('')
const webhooks = ref<any[]>([])
const loading = ref(true)

async function fetchSessions() {
  try {
    const res: any = await api('/sessions')
    sessions.value = res.data || []
    if (sessions.value.length > 0 && !selectedSession.value) {
      selectedSession.value = sessions.value[0].id
    }
  } catch {
    sessions.value = []
  }
}

async function fetchWebhooks() {
  if (!selectedSession.value) {
    webhooks.value = []
    return
  }
  loading.value = true
  try {
    const res: any = await api(`/sessions/${selectedSession.value}/webhooks`)
    webhooks.value = res.data || []
  } catch {
    webhooks.value = []
  }
  loading.value = false
}

async function deleteWebhook(wid: string) {
  try {
    await api(`/sessions/${selectedSession.value}/webhooks/${wid}`, { method: 'DELETE' })
    await fetchWebhooks()
  } catch {
    // handle error
  }
}

async function toggleWebhook(wh: any) {
  try {
    await api(`/sessions/${selectedSession.value}/webhooks/${wh.id}`, {
      method: 'PUT',
      body: { enabled: !wh.enabled }
    })
    await fetchWebhooks()
  } catch {
    // handle error
  }
}

watch(selectedSession, () => fetchWebhooks())

onMounted(async () => {
  if (!isAuthenticated.value) {
    navigateTo('/login')
    return
  }
  await fetchSessions()
  await fetchWebhooks()
})
</script>

<template>
  <UDashboardPanel id="webhooks">
    <template #header>
      <UDashboardNavbar title="Webhooks">
        <template #right>
          <USelectMenu
            v-if="sessions.length"
            v-model="selectedSession"
            :items="sessions.map(s => ({ label: s.name, value: s.id }))"
            placeholder="Select session"
            class="w-48"
          />
        </template>
      </UDashboardNavbar>
    </template>

    <div class="p-4 space-y-4">
      <div v-if="loading" class="text-center py-8">
        <UIcon name="i-lucide-loader-2" class="animate-spin text-2xl" />
      </div>

      <UCard v-for="wh in webhooks" :key="wh.id">
        <div class="flex items-center justify-between">
          <div>
            <h3 class="font-semibold">{{ wh.url }}</h3>
            <p class="text-sm text-(--ui-text-muted)">
              Events: {{ (wh.events || []).join(', ') || 'All' }}
            </p>
          </div>
          <div class="flex items-center gap-2">
            <UBadge :color="wh.enabled ? 'success' : 'neutral'" variant="subtle">
              {{ wh.enabled ? 'Active' : 'Disabled' }}
            </UBadge>
            <UButton
              :icon="wh.enabled ? 'i-lucide-pause' : 'i-lucide-play'"
              size="sm"
              variant="soft"
              @click="toggleWebhook(wh)"
            />
            <UButton
              icon="i-lucide-trash-2"
              size="sm"
              variant="soft"
              color="error"
              @click="deleteWebhook(wh.id)"
            />
          </div>
        </div>
      </UCard>

      <p v-if="!loading && webhooks.length === 0" class="text-center text-(--ui-text-muted) py-8">
        No webhooks configured for this session.
      </p>
    </div>
  </UDashboardPanel>
</template>
