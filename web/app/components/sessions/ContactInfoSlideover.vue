<script setup lang="ts">
const props = defineProps<{
  sessionId: string
  contactJid: string
  contactName?: string
}>()

const open = defineModel<boolean>('open', { default: false })
const { api } = useWzap()
const toast = useToast()

interface UserInfo {
  jid: string
  status: string
  picture: string
  devices: string[]
}

const info = ref<UserInfo | null>(null)
const loading = ref(false)

async function fetchInfo() {
  if (!props.contactJid) return
  loading.value = true
  info.value = null
  try {
    const phone = props.contactJid.split('@')[0]
    const res: { data: unknown } = await api(`/sessions/${props.sessionId}/contacts/info`, {
      method: 'POST',
      body: { phones: [phone] }
    })
    const data = res.data || {}
    const key = Object.keys(data)[0]
    info.value = key ? data[key] : null
  } catch {
    toast.add({ title: 'Failed to fetch contact info', color: 'error' })
  }
  loading.value = false
}

function copyToClipboard(text: string) {
  navigator.clipboard.writeText(text)
  toast.add({ title: 'Copied', color: 'success' })
}

watch(open, (val) => {
  if (val) fetchInfo()
})
</script>

<template>
  <USlideover
    v-model:open="open"
    title="Contact Info"
    description="Detailed contact information"
    side="right"
    :ui="{ content: 'sm:max-w-md' }"
  >
    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-16">
        <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
      </div>

      <div v-else-if="info" class="space-y-4">
        <div v-if="info.picture" class="flex justify-center">
          <img :src="info.picture" alt="Profile" class="size-20 rounded-full object-cover">
        </div>

        <UCard variant="subtle">
          <div class="space-y-3">
            <div>
              <p class="text-xs text-muted mb-1">
                JID
              </p>
              <div class="flex items-center gap-2">
                <code class="text-sm flex-1 break-all">{{ info.jid }}</code>
                <UButton
                  icon="i-lucide-copy"
                  size="xs"
                  color="neutral"
                  variant="ghost"
                  @click="copyToClipboard(info.jid)"
                />
              </div>
            </div>

            <USeparator />

            <div>
              <p class="text-xs text-muted mb-1">
                Status
              </p>
              <p class="text-sm">
                {{ info.status || '—' }}
              </p>
            </div>

            <USeparator />

            <div>
              <p class="text-xs text-muted mb-1">
                Linked Devices
              </p>
              <p v-if="!info.devices?.length" class="text-sm text-muted">
                No devices found
              </p>
              <div v-else class="space-y-1">
                <div v-for="device in info.devices" :key="device" class="flex items-center gap-2">
                  <UIcon name="i-lucide-smartphone" class="size-3.5 text-muted" />
                  <code class="text-xs break-all">{{ device }}</code>
                  <UButton
                    icon="i-lucide-copy"
                    size="xs"
                    color="neutral"
                    variant="ghost"
                    @click="copyToClipboard(device)"
                  />
                </div>
              </div>
            </div>
          </div>
        </UCard>
      </div>

      <div v-else class="flex flex-col items-center justify-center py-16 gap-3 text-muted">
        <UIcon name="i-lucide-user-x" class="size-8" />
        <p class="text-sm">
          No info available.
        </p>
      </div>
    </template>
  </USlideover>
</template>
