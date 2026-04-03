<script setup lang="ts">
import type { Session } from '~/types'

const props = defineProps<{ session: Session }>()

const flags = [
  { key: 'alwaysOnline', label: 'Always Online', icon: 'i-lucide-wifi' },
  { key: 'readMessages', label: 'Auto Read', icon: 'i-lucide-check-check' },
  { key: 'rejectCall', label: 'Reject Calls', icon: 'i-lucide-phone-off' },
  { key: 'ignoreGroups', label: 'Ignore Groups', icon: 'i-lucide-users-2' },
  { key: 'ignoreStatus', label: 'Ignore Status', icon: 'i-lucide-circle-dashed' }
] as const

function isEnabled(key: string): boolean {
  return !!(props.session.settings as any)?.[key]
}
</script>

<template>
  <UCard>
    <template #header>
      <UTooltip text="Edit settings">
        <ULink raw :to="`/sessions/${session.id}/settings`" class="flex items-center justify-between group w-full">
          <p class="font-semibold text-highlighted">Settings</p>
          <UIcon name="i-lucide-chevron-right" class="size-4 text-muted group-hover:text-highlighted transition-colors" />
        </ULink>
      </UTooltip>
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
        <UBadge
          :label="isEnabled(flag.key) ? 'On' : 'Off'"
          :color="isEnabled(flag.key) ? 'success' : 'neutral'"
          variant="subtle"
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
