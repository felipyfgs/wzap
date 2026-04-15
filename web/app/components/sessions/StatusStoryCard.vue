<script setup lang="ts">
import type { Status } from '~/composables/useStatus'

const props = defineProps<{
  senderJid: string
  senderName?: string
  latestStatus: Status
  hasUnviewed?: boolean
}>()

defineEmits<{
  click: []
}>()

const displayName = computed(() => props.senderName || props.senderJid?.split('@')[0] || 'Unknown')

function formatTime(ts?: string): string {
  if (!ts) return ''
  const date = new Date(ts)
  const now = new Date()
  const diffMs = now.getTime() - date.getTime()
  const diffHours = diffMs / (1000 * 60 * 60)

  if (diffHours < 24) {
    return date.toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
  }
  if (diffHours < 48) {
    return 'Yesterday'
  }
  return date.toLocaleDateString([], { month: 'short', day: 'numeric' })
}
</script>

<template>
  <button
    class="flex items-center gap-3 p-3 rounded-lg hover:bg-elevated/50 transition-colors w-full text-left"
    @click="$emit('click')"
  >
    <div
      class="relative flex-shrink-0"
      :class="hasUnviewed ? 'ring-2 ring-blue-500 rounded-full p-[2px]' : ''"
    >
      <div class="w-12 h-12 rounded-full bg-muted flex items-center justify-center text-sm font-medium text-highlighted">
        {{ displayName.charAt(0).toUpperCase() }}
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