<script setup lang="ts">
import type { NavigationMenuItem } from '@nuxt/ui'

const route = useRoute()
const toast = useToast()
const { sessions, refreshSessions } = useSession()

const open = ref(false)

// ── Session sidebar ─────────────────────────────────────────
const isSessionRoute = computed(() =>
  route.path.startsWith('/sessions/') && !!route.params.id
)

const currentSessionId = computed(() => route.params.id as string || '')

const sessionNavLinks = computed(() => {
  const id = currentSessionId.value
  if (!id) return []
  return [{
    label: 'Overview',
    icon: 'i-lucide-layout-dashboard',
    to: `/sessions/${id}`,
    exact: true
  }, {
    label: 'Messages',
    icon: 'i-lucide-message-square',
    to: `/sessions/${id}/messages`
  }, {
    label: 'Contacts',
    icon: 'i-lucide-users',
    to: `/sessions/${id}/contacts`
  }, {
    label: 'Groups',
    icon: 'i-lucide-users-2',
    to: `/sessions/${id}/groups`
  }, {
    label: 'Webhooks',
    icon: 'i-lucide-webhook',
    to: `/sessions/${id}/webhooks`
  }, {
    label: 'Media',
    icon: 'i-lucide-image',
    to: `/sessions/${id}/media`
  }, {
    label: 'Settings',
    icon: 'i-lucide-settings-2',
    to: `/sessions/${id}/settings`
  }]
})

watch(isSessionRoute, (val) => {
  if (val && sessions.value.length === 0) refreshSessions()
}, { immediate: true })

const links = [[{
  label: 'Dashboard',
  icon: 'i-lucide-house',
  to: '/',
  onSelect: () => {
    open.value = false
  }
}, {
  label: 'Sessions',
  icon: 'i-lucide-smartphone',
  to: '/sessions',
  onSelect: () => {
    open.value = false
  }
}, {
  label: 'Webhooks',
  icon: 'i-lucide-webhook',
  to: '/webhooks',
  onSelect: () => {
    open.value = false
  }
}, {
  label: 'Live Events',
  icon: 'i-lucide-activity',
  to: '/logs',
  onSelect: () => {
    open.value = false
  }
}, {
  label: 'Settings',
  to: '/settings',
  icon: 'i-lucide-settings',
  defaultOpen: true,
  type: 'trigger',
  children: [{
    label: 'General',
    to: '/settings',
    exact: true,
    onSelect: () => {
      open.value = false
    }
  }, {
    label: 'Members',
    to: '/settings/members',
    onSelect: () => {
      open.value = false
    }
  }, {
    label: 'Notifications',
    to: '/settings/notifications',
    onSelect: () => {
      open.value = false
    }
  }, {
    label: 'Security',
    to: '/settings/security',
    onSelect: () => {
      open.value = false
    }
  }]
}], [{
  label: 'Documentation',
  icon: 'i-lucide-book-open',
  to: 'https://github.com/wzap/wzap',
  target: '_blank'
}, {
  label: 'Help & Support',
  icon: 'i-lucide-info',
  to: 'https://github.com/wzap/wzap/issues',
  target: '_blank'
}]] satisfies NavigationMenuItem[][]

const groups = computed(() => [{
  id: 'links',
  label: 'Go to',
  items: links.flat()
}, {
  id: 'code',
  label: 'Code',
  items: [{
    id: 'source',
    label: 'View source on GitHub',
    icon: 'i-simple-icons-github',
    to: 'https://github.com/wzap/wzap',
    target: '_blank'
  }]
}])

onMounted(async () => {
  const cookie = useCookie('cookie-consent')
  if (cookie.value === 'accepted') {
    return
  }

  toast.add({
    title: 'We use first-party cookies to enhance your experience on our website.',
    duration: 0,
    close: false,
    actions: [{
      label: 'Accept',
      color: 'neutral',
      variant: 'outline',
      onClick: () => {
        cookie.value = 'accepted'
      }
    }, {
      label: 'Opt out',
      color: 'neutral',
      variant: 'ghost'
    }]
  })
})
</script>

<template>
  <UDashboardGroup unit="rem">
    <!-- Sidebar raiz — visível fora de sessão -->
    <UDashboardSidebar
      v-if="!isSessionRoute"
      id="default"
      v-model:open="open"
      collapsible
      resizable
      class="bg-elevated/25"
      :ui="{ footer: 'lg:border-t lg:border-default' }"
    >
      <template #header="{ collapsed }">
        <TeamsMenu :collapsed="collapsed" />
      </template>

      <template #default="{ collapsed }">
        <UDashboardSearchButton :collapsed="collapsed" class="bg-transparent ring-default" />

        <UNavigationMenu
          :collapsed="collapsed"
          :items="links[0]"
          orientation="vertical"
          tooltip
          popover
        />

        <UNavigationMenu
          :collapsed="collapsed"
          :items="links[1]"
          orientation="vertical"
          tooltip
          class="mt-auto"
        />
      </template>

      <template #footer="{ collapsed }">
        <UserMenu :collapsed="collapsed" />
      </template>
    </UDashboardSidebar>

    <!-- Sidebar de sessão — visível dentro de /sessions/:id -->
    <UDashboardSidebar
      v-if="isSessionRoute"
      id="session"
      collapsible
      resizable
      class="bg-elevated/25"
      :ui="{ footer: 'lg:border-t lg:border-default' }"
    >
      <template #header="{ collapsed }">
        <SessionsMenu :collapsed="collapsed" />
      </template>

      <template #default="{ collapsed }">
        <UNavigationMenu
          :collapsed="collapsed"
          :items="sessionNavLinks"
          orientation="vertical"
          tooltip
          popover
        />
      </template>

      <template #footer>
        <UserMenu />
      </template>
    </UDashboardSidebar>

    <UDashboardSearch :groups="groups" />

    <slot />

    <NotificationsSlideover />
  </UDashboardGroup>
</template>
