<script setup lang="ts">
const props = defineProps<{ sessionId: string }>()

const { api } = useWzap()
const toast = useToast()

const loading = ref(true)
const saving = ref(false)

interface PrivacySetting {
  name: string
  value: string
}

const settings = ref<PrivacySetting[]>([])

const settingLabels: Record<string, string> = {
  groupadd: 'Who can add me to groups',
  last: 'Last seen',
  status: 'Status visibility',
  profile: 'Profile photo visibility',
  readreceipts: 'Read receipts',
  online: 'Online visibility',
  calladd: 'Who can add me to calls'
}

const valueOptions = [
  { label: 'Everyone', value: 'all' },
  { label: 'Contacts', value: 'contacts' },
  { label: 'Nobody', value: 'none' }
]

async function fetchPrivacy() {
  loading.value = true
  try {
    const res: { data: unknown } = await api(`/sessions/${props.sessionId}/contacts/privacy`)
    const data = res.data || {}
    settings.value = Object.entries(data).map(([name, value]) => ({
      name,
      value: String(value)
    }))
  } catch {
    settings.value = []
  }
  loading.value = false
}

async function updateSetting(name: string, value: string) {
  saving.value = true
  try {
    await api(`/sessions/${props.sessionId}/contacts/privacy`, {
      method: 'POST',
      body: { setting: name, value }
    })
    toast.add({ title: 'Privacy setting updated', color: 'success' })
    await fetchPrivacy()
  } catch {
    toast.add({ title: 'Failed to update privacy setting', color: 'error' })
  }
  saving.value = false
}

watch(() => props.sessionId, fetchPrivacy, { immediate: true })
</script>

<template>
  <UPageCard
    title="Privacy"
    description="Manage your WhatsApp privacy settings."
    variant="naked"
    orientation="horizontal"
    class="mt-6 mb-0"
  />

  <UPageCard variant="subtle">
    <div v-if="loading" class="flex items-center justify-center py-8">
      <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
    </div>

    <template v-else-if="settings.length">
      <template v-for="(s, i) in settings" :key="s.name">
        <div class="flex max-sm:flex-col justify-between sm:items-center gap-4">
          <span class="text-sm">{{ settingLabels[s.name] || s.name }}</span>
          <USelect
            :model-value="s.value"
            :items="valueOptions"
            value-key="value"
            size="sm"
            class="w-40"
            :disabled="saving"
            @update:model-value="(v: string) => updateSetting(s.name, v)"
          />
        </div>
        <USeparator v-if="i < settings.length - 1" />
      </template>
    </template>

    <div v-else class="flex flex-col items-center justify-center py-8 gap-3 text-muted">
      <UIcon name="i-lucide-shield" class="size-6" />
      <p class="text-sm">
        No privacy settings available.
      </p>
    </div>
  </UPageCard>
</template>
