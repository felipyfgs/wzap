<script setup lang="ts">
const props = defineProps<{ sessionId: string, sessionName: string }>()
const emit = defineEmits<{ connected: [] }>()

const { api } = useWzap()
const open = ref(false)
const qrImage = ref('')
const loadingQR = ref(false)
let intervalId: ReturnType<typeof setInterval> | null = null

async function pollQR() {
  try {
    const res: { data: unknown } = await api(`/sessions/${props.sessionId}/qr`)
    const qrData = res.data as { image?: string } | null
    qrImage.value = qrData?.image || ''
  } catch {
    /* QR not ready yet or already expired — keep polling */
  }
}

async function checkStatus() {
  try {
    const res: { data: unknown } = await api(`/sessions/${props.sessionId}`)
    if ((res.data as { status?: string } | null)?.status === 'connected') {
      emit('connected')
      close()
    }
  } catch { /* ignore — keep polling */ }
}

function startPolling() {
  loadingQR.value = true
  pollQR().then(() => {
    loadingQR.value = false
  })
  intervalId = setInterval(async () => {
    await pollQR()
    await checkStatus()
  }, 3000)
}

function stopPolling() {
  if (intervalId) {
    clearInterval(intervalId)
    intervalId = null
  }
}

function close() {
  open.value = false
}

watch(open, (val) => {
  if (!val) {
    stopPolling()
    qrImage.value = ''
  }
})

function show() {
  open.value = true
  startPolling()
}

defineExpose({ show })
</script>

<template>
  <UModal
    v-model:open="open"
    :title="`Scan QR — ${sessionName}`"
    description="Open WhatsApp on your phone and scan the QR code below."
    :ui="{ body: 'flex flex-col items-center gap-4 py-6' }"
  >
    <template #body>
      <div v-if="loadingQR || !qrImage" class="flex flex-col items-center gap-3 py-8">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
        <p class="text-sm text-muted">
          Waiting for QR code…
        </p>
      </div>

      <img
        v-else
        :src="qrImage"
        alt="QR Code"
        class="size-64 rounded-lg"
      >

      <p class="text-xs text-muted text-center max-w-xs">
        The QR code refreshes automatically every 30 seconds. Keep this window open until the session connects.
      </p>

      <UButton
        label="Close"
        color="neutral"
        variant="subtle"
        @click="close"
      />
    </template>
  </UModal>
</template>
