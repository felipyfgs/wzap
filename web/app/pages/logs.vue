<script setup lang="ts">
const { token } = useWzap()

const events = ref<Record<string, unknown>[]>([])
const connected = ref(false)
const maxEvents = 200
let ws: WebSocket | null = null

function connect() {
  if (ws) ws.close()
  const proto = window.location.protocol === 'https:' ? 'wss:' : 'ws:'
  const wsUrl = `${proto}//${window.location.host}/ws?token=${token.value}`
  ws = new WebSocket(wsUrl)
  ws.onopen = () => {
    connected.value = true
  }
  ws.onmessage = (e) => {
    try {
      const data = JSON.parse(e.data)
      events.value.unshift({ id: Date.now(), timestamp: new Date().toLocaleTimeString(), ...data })
      if (events.value.length > maxEvents) events.value = events.value.slice(0, maxEvents)
    } catch { /* ignore parse errors */ }
  }
  ws.onclose = () => {
    connected.value = false
  }
  ws.onerror = () => {
    connected.value = false
  }
}

function disconnect() {
  ws?.close()
  ws = null
}

function clearEvents() {
  events.value = []
}

onMounted(() => {
  connect()
})

onUnmounted(() => disconnect())
</script>

<template>
  <UDashboardPanel id="logs">
    <template #header>
      <UDashboardNavbar title="Live Events">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>

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
              v-if="events.length"
              icon="i-lucide-trash-2"
              label="Clear"
              color="neutral"
              variant="ghost"
              @click="clearEvents"
            />
          </div>
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div v-if="events.length === 0" class="flex flex-col items-center justify-center py-24 gap-3 text-muted">
        <UIcon name="i-lucide-radio" class="size-10" />
        <p class="text-sm">
          No events yet. Click <strong>Connect</strong> to start receiving live events.
        </p>
      </div>

      <div v-else class="space-y-2">
        <UCard
          v-for="evt in events"
          :key="evt.id"
          class="text-sm"
          :ui="{ body: 'p-3' }"
        >
          <div class="flex items-center justify-between mb-2">
            <div class="flex items-center gap-2">
              <UBadge color="primary" variant="subtle">
                {{ evt.event || 'unknown' }}
              </UBadge>
              <span v-if="evt.sessionId" class="text-xs text-muted">{{ evt.sessionId }}</span>
            </div>
            <span class="text-xs text-muted">{{ evt.timestamp }}</span>
          </div>
          <pre class="text-xs overflow-x-auto whitespace-pre-wrap text-muted">{{ JSON.stringify(evt, null, 2) }}</pre>
        </UCard>
      </div>
    </template>
  </UDashboardPanel>
</template>
