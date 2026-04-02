<script setup lang="ts">
import type { Session } from '~/types'

defineProps<{ session: Session }>()

const toast = useToast()

async function copy(value: string, label: string) {
  await navigator.clipboard.writeText(value)
  toast.add({ title: `${label} copied`, color: 'success' })
}
</script>

<template>
  <UCard>
    <template #header>
      <p class="font-semibold text-highlighted">Connection</p>
    </template>

    <div class="space-y-3 text-sm">
      <div class="flex items-center justify-between gap-3">
        <span class="text-muted shrink-0">Session ID</span>
        <div class="flex items-center gap-1 min-w-0">
          <span class="font-mono text-xs truncate text-highlighted">{{ session.id }}</span>
          <UButton icon="i-lucide-copy" size="xs" color="neutral" variant="ghost" @click="copy(session.id, 'Session ID')" />
        </div>
      </div>

      <USeparator />

      <div class="flex items-center justify-between gap-3">
        <span class="text-muted shrink-0">JID</span>
        <div class="flex items-center gap-1 min-w-0">
          <span class="font-mono text-xs text-highlighted">{{ session.jid || 'Not paired' }}</span>
          <UButton v-if="session.jid" icon="i-lucide-copy" size="xs" color="neutral" variant="ghost" @click="copy(session.jid!, 'JID')" />
        </div>
      </div>

      <template v-if="session.apiKey">
        <USeparator />
        <div class="flex items-center justify-between gap-3">
          <span class="text-muted shrink-0">API Key</span>
          <div class="flex items-center gap-1">
            <span class="font-mono text-xs text-highlighted">{{ session.apiKey.slice(0, 8) }}••••••••</span>
            <UButton icon="i-lucide-copy" size="xs" color="neutral" variant="ghost" @click="copy(session.apiKey!, 'API Key')" />
          </div>
        </div>
      </template>

      <USeparator />

      <div class="flex items-center justify-between gap-3">
        <span class="text-muted shrink-0">Created</span>
        <span class="text-highlighted text-xs">{{ session.createdAt ? new Date(session.createdAt).toLocaleString() : '—' }}</span>
      </div>

      <USeparator />

      <div class="flex items-center justify-between gap-3">
        <span class="text-muted shrink-0">Updated</span>
        <span class="text-highlighted text-xs">{{ session.updatedAt ? new Date(session.updatedAt).toLocaleString() : '—' }}</span>
      </div>
    </div>
  </UCard>
</template>
