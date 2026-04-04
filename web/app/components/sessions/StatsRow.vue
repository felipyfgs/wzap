<script setup lang="ts">
import type { Session, SessionProfile } from '~/types'

const props = defineProps<{ session: Session, profile?: SessionProfile | null }>()

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
  <!-- Profile hero (when connected with WhatsApp profile data) -->
  <UCard v-if="profile?.pushName || profile?.pictureUrl" class="mb-4">
    <div class="flex items-start gap-4 flex-wrap">
      <!-- Avatar -->
      <div class="relative shrink-0">
        <img
          v-if="profile.pictureUrl"
          :src="profile.pictureUrl"
          :alt="profile.pushName"
          class="size-16 rounded-full object-cover ring-2 ring-default"
        >
        <div
          v-else
          class="size-16 rounded-full bg-primary/10 ring-2 ring-default flex items-center justify-center text-2xl font-bold text-primary"
        >
          {{ session.name.slice(0, 2).toUpperCase() }}
        </div>
        <span
          class="absolute bottom-0 right-0 size-4 rounded-full ring-2 ring-default"
          :class="{
            'bg-success': session.status === 'connected',
            'bg-warning': session.status === 'connecting',
            'bg-info': session.status === 'pairing',
            'bg-muted': session.status === 'disconnected',
            'bg-error': session.status === 'error'
          }"
        />
      </div>

      <!-- Info -->
      <div class="flex-1 min-w-0 space-y-1">
        <div class="flex items-center gap-2 flex-wrap">
          <p class="text-lg font-semibold text-highlighted leading-none">
            {{ profile.pushName || session.name }}
          </p>
          <UBadge
            v-if="profile.businessName"
            size="xs"
            color="info"
            variant="subtle"
            icon="i-lucide-briefcase"
          >
            {{ profile.businessName }}
          </UBadge>
          <UBadge
            v-if="profile.platform"
            size="xs"
            color="neutral"
            variant="subtle"
          >
            {{ profile.platform }}
          </UBadge>
        </div>
        <p class="text-sm text-muted font-mono">
          {{ phone }}
        </p>
        <p v-if="profile.status" class="text-sm text-muted italic">
          {{ profile.status }}
        </p>
      </div>

      <!-- Meta -->
      <div class="text-right text-xs text-muted space-y-1 shrink-0">
        <p>Since {{ createdLabel }}</p>
        <p>Updated {{ updatedLabel }}</p>
        <UBadge :color="sessionStatusColor(session.status)" variant="subtle" class="capitalize">
          {{ session.status }}
        </UBadge>
      </div>
    </div>
  </UCard>

  <!-- 4 stats cards always visible -->
  <UPageGrid class="lg:grid-cols-4 gap-4 sm:gap-6 lg:gap-px">
    <UPageCard
      icon="i-lucide-activity"
      title="Status"
      variant="subtle"
      :ui="cardUi"
      class="lg:rounded-none first:rounded-l-lg last:rounded-r-lg"
    >
      <UBadge
        :color="sessionStatusColor(session.status)"
        variant="subtle"
        size="lg"
        class="capitalize"
      >
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
