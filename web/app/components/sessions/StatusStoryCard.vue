<script setup lang="ts">
import type { Status } from '~/composables/useStatus'

const props = defineProps<{
  sessionId: string
  senderJid: string
  senderName?: string
  latestStatus: Status
  hasUnviewed?: boolean
}>()

defineEmits<{
  click: []
}>()

const { fetchAvatar } = useAvatarCache()
const avatarUrl = ref<string | null>(null)

onMounted(async () => {
  avatarUrl.value = await fetchAvatar(props.sessionId, props.senderJid)
})

const displayName = computed(() => props.senderName || props.senderJid?.split('@')[0] || 'Unknown')

function formatTime(ts?: string): string {
  if (!ts) return ''
  const date = new Date(ts)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffHours = diffMs / (1000 * 60 * 60)
  if (diffHours < 24) return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  if (diffHours < 48) return 'Yesterday'
  return date.toLocaleDateString([], { month: 'short', day: 'numeric' })
}
</script>

<template>
  <button
    class="flex items-center gap-3 p-3 rounded-lg hover:bg-elevated/50 transition-colors w-full text-left"
    @click="$emit('click')"
  >
    <div class="relative flex-shrink-0">
      <div
        class="rounded-full p-[2.5px]"
        :style="hasUnviewed ? 'background: linear-gradient(135deg, #25D366, #128C7E)' : 'background: transparent'"
      >
        <div class="w-12 h-12 rounded-full bg-muted flex items-center justify-center text-sm font-medium text-highlighted ring-2 ring-default overflow-hidden">
          <img
            v-if="avatarUrl"
            :src="avatarUrl"
            :alt="displayName"
            class="w-full h-full object-cover"
            @error="avatarUrl = null"
          >
          <span v-else>{{ displayName.charAt(0).toUpperCase() }}</span>
        </div>
      </div>
    </div>
    <div class="flex-1 min-w-0">
      <p class="text-sm font-medium text-highlighted truncate">
        {{ displayName }}
      </p>
      <p class="text-xs text-muted truncate">
        {{ latestStatus?.statusType === 'text' ? latestStatus.body : latestStatus?.statusType }}
      </p>
    </div>
    <div class="text-xs text-muted">
      {{ formatTime(latestStatus?.timestamp) }}
    </div>
  </button>
</template>