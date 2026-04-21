<script setup lang="ts">
const props = defineProps<{ sessionId: string, sessionName: string, token?: string }>()
const emit = defineEmits<{ rotated: [newToken: string] }>()

const { api } = useWzap()
const toast = useToast()

const open = ref(false)
const revealed = ref(false)
const rotating = ref(false)
const currentToken = ref<string>('')

function show() {
  open.value = true
  revealed.value = false
  currentToken.value = props.token || ''
}

watch(() => props.token, (t) => {
  currentToken.value = t || ''
})

async function copy() {
  if (!currentToken.value) return
  await navigator.clipboard.writeText(currentToken.value)
  toast.add({ title: 'Token copied', color: 'success' })
}

async function rotate() {
  rotating.value = true
  try {
    const res: { data: unknown } = await api(`/sessions/${props.sessionId}`, {
      method: 'PUT',
      body: { token: '' }
    })
    const newToken = (res.data as { token?: string } | null)?.token || ''
    currentToken.value = newToken
    revealed.value = true
    emit('rotated', newToken)
    toast.add({ title: 'Token rotated', color: 'success' })
  } catch {
    toast.add({ title: 'Failed to rotate token', color: 'error' })
  }
  rotating.value = false
}

const maskedToken = computed(() => {
  if (!currentToken.value) return '—'
  if (revealed.value) return currentToken.value
  return currentToken.value.slice(0, 6) + '•'.repeat(Math.max(0, currentToken.value.length - 10)) + currentToken.value.slice(-4)
})

const curlSample = computed(() => {
  if (!currentToken.value) return ''
  return `curl -H "Authorization: ${currentToken.value}" \\
  http://localhost:8080/sessions/${props.sessionId}`
})

defineExpose({ show })
</script>

<template>
  <UModal
    v-model:open="open"
    :title="`API Token — ${sessionName}`"
    description="Session-scoped token. Use it in the Authorization header."
    :ui="{ content: 'sm:max-w-xl' }"
  >
    <template #body>
      <div class="space-y-4">
        <div class="rounded-lg border border-default bg-elevated/40 p-3 space-y-2">
          <p class="text-xs text-muted uppercase font-medium">
            Token
          </p>
          <div class="flex items-center gap-2">
            <code class="flex-1 text-xs font-mono text-highlighted break-all">{{ maskedToken }}</code>
            <UTooltip :text="revealed ? 'Hide' : 'Reveal'">
              <UButton
                :icon="revealed ? 'i-lucide-eye-off' : 'i-lucide-eye'"
                size="xs"
                color="neutral"
                variant="ghost"
                :disabled="!currentToken"
                @click="revealed = !revealed"
              />
            </UTooltip>
            <UTooltip text="Copy">
              <UButton
                icon="i-lucide-copy"
                size="xs"
                color="neutral"
                variant="ghost"
                :disabled="!currentToken"
                @click="copy"
              />
            </UTooltip>
            <UTooltip text="Rotate">
              <UButton
                icon="i-lucide-refresh-cw"
                size="xs"
                color="warning"
                variant="ghost"
                :loading="rotating"
                @click="rotate"
              />
            </UTooltip>
          </div>
        </div>

        <div v-if="currentToken" class="space-y-2">
          <p class="text-xs text-muted uppercase font-medium">
            Example request
          </p>
          <pre class="rounded-lg bg-elevated/60 border border-default p-3 text-xs font-mono text-highlighted overflow-x-auto">{{ curlSample }}</pre>
        </div>

        <p class="text-xs text-muted">
          Rotating invalidates the previous token. Update any client that uses it.
        </p>
      </div>
    </template>
  </UModal>
</template>
