<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'
import type { Session } from '~/types'

defineProps<{
  collapsed?: boolean
}>()

const { sessions, current, refreshSessions } = useSession()
const route = useRoute()

const currentSessionId = computed(() => route.params.id as string || '')

const selectedSession = computed<Session | null>(() =>
  sessions.value.find(s => s.id === currentSessionId.value) ?? current.value
)

function statusColor(status: string) {
  const map: Record<string, string> = {
    connected: 'bg-success',
    connecting: 'bg-warning',
    pairing: 'bg-info',
    disconnected: 'bg-muted',
    error: 'bg-error'
  }
  return map[status?.toLowerCase()] ?? 'bg-muted'
}

const sessionInitials = computed(() => {
  const name = selectedSession.value?.name || '?'
  return name.slice(0, 2).toUpperCase()
})

const items = computed<DropdownMenuItem[][]>(() => {
  const sessionItems: DropdownMenuItem[] = sessions.value.map(s => ({
    label: s.name,
    avatar: { text: s.name.slice(0, 2).toUpperCase(), alt: s.name },
    onSelect() {
      navigateTo(`/sessions/${s.id}`)
    }
  }))

  return [
    sessionItems,
    [{
      label: 'All Sessions',
      icon: 'i-lucide-layout-grid',
      onSelect() { navigateTo('/sessions') }
    }]
  ]
})

onMounted(() => {
  if (sessions.value.length === 0) refreshSessions()
})
</script>

<template>
  <UDropdownMenu
    :items="items"
    :content="{ align: 'center', collisionPadding: 12 }"
    :ui="{ content: collapsed ? 'w-40' : 'w-(--reka-dropdown-menu-trigger-width)' }"
  >
    <UButton
      color="neutral"
      variant="ghost"
      block
      :square="collapsed"
      class="data-[state=open]:bg-elevated"
      :class="[!collapsed && 'py-2']"
      :ui="{ trailingIcon: 'text-dimmed' }"
      v-bind="{
        label: collapsed ? undefined : (selectedSession?.name || 'Select session'),
        trailingIcon: collapsed ? undefined : 'i-lucide-chevrons-up-down'
      }"
    >
      <template #leading>
        <span class="relative flex size-5 shrink-0 items-center justify-center rounded-sm bg-elevated text-xs font-semibold ring-1 ring-default">
          {{ sessionInitials }}
          <span
            class="absolute -bottom-0.5 -right-0.5 size-2 rounded-full ring-1 ring-default"
            :class="statusColor(selectedSession?.status || '')"
          />
        </span>
      </template>
    </UButton>
  </UDropdownMenu>
</template>
