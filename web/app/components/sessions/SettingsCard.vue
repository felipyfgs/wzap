<script setup lang="ts">
import type { Session } from '~/types'

defineProps<{ session: Session }>()

const flags = [
  { key: 'alwaysOnline', label: 'Always Online', icon: 'i-lucide-wifi' },
  { key: 'readMessages', label: 'Auto Read Messages', icon: 'i-lucide-check-check' },
  { key: 'rejectCall', label: 'Reject Calls', icon: 'i-lucide-phone-off' },
  { key: 'ignoreGroups', label: 'Ignore Groups', icon: 'i-lucide-users-2' },
  { key: 'ignoreStatus', label: 'Ignore Status', icon: 'i-lucide-circle-dashed' }
] as const
</script>

<template>
  <UCard>
    <template #header>
      <p class="font-semibold text-highlighted">Settings</p>
    </template>

    <div v-if="session.settings" class="space-y-3">
      <div
        v-for="flag in flags"
        :key="flag.key"
        class="flex items-center justify-between"
      >
        <div class="flex items-center gap-2 text-sm">
          <UIcon :name="flag.icon" class="size-4 text-muted" />
          <span>{{ flag.label }}</span>
        </div>
        <UToggle
          :model-value="!!session.settings![flag.key]"
          disabled
          size="sm"
        />
      </div>

      <div v-if="session.settings.rejectCall && session.settings.msgRejectCall" class="pt-1">
        <p class="text-xs text-muted mb-1">Reject call message</p>
        <p class="text-xs text-highlighted italic">{{ session.settings.msgRejectCall }}</p>
      </div>
    </div>

    <div v-else class="text-sm text-muted py-2">
      No settings configured.
    </div>
  </UCard>
</template>
