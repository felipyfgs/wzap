<script setup lang="ts">
interface LiveEvent {
  id: number
  timestamp: string
  event?: string
  sessionId?: string
  [key: string]: unknown
}

const props = defineProps<{
  sessionIdFilter?: string
}>()

const { token } = useWzap()

const open = ref(false)
const events = ref<LiveEvent[]>([])
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
      events.value.unshift({
        id: Date.now() + Math.random(),
        timestamp: new Date().toLocaleTimeString(),
        ...data
      })
      if (events.value.length > maxEvents) {
        events.value = events.value.slice(0, maxEvents)
      }
    } catch { /* ignore */ }
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

const filtered = computed(() => {
  if (!props.sessionIdFilter) return events.value
  return events.value.filter(e => e.sessionId === props.sessionIdFilter)
})

function show() {
  open.value = true
  if (!connected.value) connect()
}

watch(open, (val) => {
  if (!val) disconnect()
})

onBeforeUnmount(() => disconnect())

defineExpose({ show })
</script>

<template>
  <USlideover
    v-model:open="open"
    :title="sessionIdFilter ? `Live Events — ${sessionIdFilter}` : 'Live Events'"
    :description="connected ? 'Streaming live from /ws' : 'Disconnected'"
    side="right"
    :ui="{ content: 'sm:max-w-xl' }"
  >
    <template #body>
      <div class="flex items-center gap-2 mb-3">
        <UBadge :color="connected ? 'success' : 'neutral'" variant="subtle" size="sm">
          <UIcon :name="connected ? 'i-lucide-radio' : 'i-lucide-radio-receiver'" class="size-3" />
          {{ connected ? 'live' : 'offline' }}
        </UBadge>
        <UButton
          v-if="!connected"
          label="Connect"
          icon="i-lucide-play"
          size="xs"
          color="primary"
          @click="connect"
        />
        <UButton
          v-else
          label="Stop"
          icon="i-lucide-square"
          size="xs"
          color="error"
          variant="soft"
          @click="disconnect"
        />
        <UButton
          v-if="events.length"
          label="Clear"
          icon="i-lucide-trash-2"
          size="xs"
          color="neutral"
          variant="ghost"
          @click="clearEvents"
        />
        <span class="ml-auto text-xs text-muted">
          {{ filtered.length }}{{ sessionIdFilter ? ` / ${events.length}` : '' }}
        </span>
      </div>

      <div v-if="filtered.length === 0" class="flex flex-col items-center py-16 gap-2 text-muted">
        <UIcon name="i-lucide-radio" class="size-8" />
        <p class="text-sm">
          Waiting for events…
        </p>
      </div>

      <div v-else class="space-y-2">
        <div
          v-for="evt in filtered"
          :key="evt.id"
          class="rounded-md border border-default bg-elevated/40 p-2.5 text-xs"
        >
          <div class="flex items-center justify-between mb-1">
            <div class="flex items-center gap-1.5">
              <UBadge color="primary" variant="subtle" size="xs">
                {{ evt.event || 'unknown' }}
              </UBadge>
              <span v-if="evt.sessionId && !sessionIdFilter" class="text-[10px] text-muted font-mono">
                {{ evt.sessionId }}
              </span>
            </div>
            <span class="text-[10px] text-muted">{{ evt.timestamp }}</span>
          </div>
          <pre class="overflow-x-auto whitespace-pre-wrap text-[11px] text-muted leading-snug">{{ JSON.stringify(evt, null, 2) }}</pre>
        </div>
      </div>
    </template>
  </USlideover>
</template>
