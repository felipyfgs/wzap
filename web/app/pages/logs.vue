<script setup lang="ts">
const { token, apiBase, isAuthenticated } = useWzap()

const events = ref<any[]>([])
const connected = ref(false)
const maxEvents = 200
let ws: WebSocket | null = null

function connect() {
  if (ws) ws.close()

  const wsUrl = apiBase.value.replace(/^http/, 'ws') + '/ws?token=' + token.value
  ws = new WebSocket(wsUrl)

  ws.onopen = () => {
    connected.value = true
  }

  ws.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data)
      events.value.unshift({
        id: Date.now(),
        timestamp: new Date().toLocaleTimeString(),
        ...data
      })
      if (events.value.length > maxEvents) {
        events.value = events.value.slice(0, maxEvents)
      }
    } catch {
      // ignore parse errors
    }
  }

  ws.onclose = () => {
    connected.value = false
  }

  ws.onerror = () => {
    connected.value = false
  }
}

function disconnect() {
  if (ws) {
    ws.close()
    ws = null
  }
}

function clearEvents() {
  events.value = []
}

onMounted(() => {
  if (!isAuthenticated.value) {
    navigateTo('/login')
    return
  }
})

onUnmounted(() => {
  disconnect()
})
</script>

<template>
  <UDashboardPanel id="logs">
    <template #header>
      <UDashboardNavbar title="Live Events">
        <template #right>
          <div class="flex items-center gap-2">
            <UBadge :color="connected ? 'success' : 'neutral'" variant="subtle">
              {{ connected ? 'Connected' : 'Disconnected' }}
            </UBadge>
            <UButton
              v-if="!connected"
              icon="i-lucide-play"
              label="Connect"
              color="primary"
              @click="connect"
            />
            <UButton
              v-else
              icon="i-lucide-square"
              label="Stop"
              color="error"
              variant="soft"
              @click="disconnect"
            />
            <UButton
              icon="i-lucide-trash-2"
              label="Clear"
              variant="ghost"
              @click="clearEvents"
            />
          </div>
        </template>
      </UDashboardNavbar>
    </template>

    <div class="p-4">
      <div v-if="events.length === 0" class="text-center text-(--ui-text-muted) py-12">
        <UIcon name="i-lucide-radio" class="text-4xl mb-2" />
        <p>No events yet. Connect to start receiving live events.</p>
      </div>

      <div v-else class="space-y-2 max-h-[calc(100vh-12rem)] overflow-y-auto">
        <UCard v-for="evt in events" :key="evt.id" class="text-sm">
          <div class="flex items-start justify-between">
            <div>
              <UBadge color="primary" variant="subtle" class="mr-2">{{ evt.event || 'unknown' }}</UBadge>
              <span class="text-(--ui-text-muted)">{{ evt.timestamp }}</span>
            </div>
            <span v-if="evt.sessionId" class="text-xs text-(--ui-text-muted)">{{ evt.sessionId }}</span>
          </div>
          <pre class="mt-2 text-xs overflow-x-auto whitespace-pre-wrap text-(--ui-text-muted)">{{ JSON.stringify(evt, null, 2) }}</pre>
        </UCard>
      </div>
    </div>
  </UDashboardPanel>
</template>
