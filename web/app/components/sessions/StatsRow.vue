<script setup lang="ts">
import type { Session } from '~/types'

const props = defineProps<{ session: Session }>()

function statusColor(status: string) {
  const map: Record<string, 'success' | 'warning' | 'error' | 'neutral' | 'info'> = {
    connected: 'success', connecting: 'warning', pairing: 'info',
    disconnected: 'neutral', error: 'error'
  }
  return map[status?.toLowerCase()] ?? 'neutral'
}

const phone = computed(() => {
  if (!props.session.jid) return 'Not paired'
  return props.session.jid.replace(/@.*$/, '').replace(/^(\d+)$/, '+$1')
})

const createdLabel = computed(() =>
  props.session.createdAt
    ? new Date(props.session.createdAt).toLocaleDateString('pt-BR', { day: '2-digit', month: 'short', year: 'numeric' })
    : '—'
)

const updatedLabel = computed(() =>
  props.session.updatedAt
    ? new Date(props.session.updatedAt).toLocaleDateString('pt-BR', { day: '2-digit', month: 'short', year: 'numeric' })
    : '—'
)

const cardUi = {
  container: 'gap-y-1.5',
  wrapper: 'items-start',
  leading: 'p-2.5 rounded-full bg-primary/10 ring ring-inset ring-primary/25',
  title: 'font-normal text-muted text-xs uppercase'
}
</script>

<template>
  <UPageGrid class="lg:grid-cols-4 gap-4 sm:gap-6 lg:gap-px">
    <UPageCard
      icon="i-lucide-activity"
      title="Status"
      variant="subtle"
      :ui="cardUi"
      class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
    >
      <UBadge :color="statusColor(session.status)" variant="subtle" size="lg" class="capitalize">
        {{ session.status }}
      </UBadge>
    </UPageCard>

    <UPageCard
      icon="i-lucide-phone"
      title="Phone"
      variant="subtle"
      :ui="cardUi"
      class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
    >
      <span class="text-lg font-semibold text-highlighted font-mono">{{ phone }}</span>
    </UPageCard>

    <UPageCard
      icon="i-lucide-calendar"
      title="Since"
      variant="subtle"
      :ui="cardUi"
      class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
    >
      <span class="text-lg font-semibold text-highlighted">{{ createdLabel }}</span>
    </UPageCard>

    <UPageCard
      icon="i-lucide-clock"
      title="Updated"
      variant="subtle"
      :ui="cardUi"
      class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
    >
      <span class="text-lg font-semibold text-highlighted">{{ updatedLabel }}</span>
    </UPageCard>
  </UPageGrid>
</template>
