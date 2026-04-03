<script setup lang="ts">
import type { TabsItem } from '@nuxt/ui'
import type { GroupDetail, JoinRequest } from './group/types'

const props = defineProps<{
  sessionId: string
  groupJid: string
}>()

const open = defineModel<boolean>('open', { default: false })
const emit = defineEmits<{ left: [] }>()

const { api } = useWzap()
const toast = useToast()

const group = ref<GroupDetail | null>(null)
const loading = ref(false)
const leavingGroup = ref(false)
const joinRequests = ref<JoinRequest[]>([])

const subgroups = computed(() => {
  return group.value?.subgroups || []
})

const members = computed(() => {
  if (!group.value) return []
  return group.value.participants
})

const tabItems = computed<TabsItem[]>(() => {
  if (!group.value) return []
  const items: TabsItem[] = [
    { label: 'Info', icon: 'i-lucide-info', value: 'info', slot: 'info' as const },
    { label: 'Participants', icon: 'i-lucide-users-2', value: 'participants', slot: 'participants' as const, badge: members.value.length }
  ]
  if (group.value.isParent) {
    items.push({ label: 'Subgroups', icon: 'i-lucide-network', value: 'community', slot: 'community' as const, badge: subgroups.value.length })
  }
  if (group.value.isAdmin) {
    items.push({ label: 'Settings', icon: 'i-lucide-settings-2', value: 'settings', slot: 'settings' as const })
  }
  return items
})

async function fetchDetail() {
  if (!props.groupJid) return
  loading.value = true
  try {
    const res: any = await api(`/sessions/${props.sessionId}/groups/info`, {
      method: 'POST',
      body: { groupJid: props.groupJid }
    })
    group.value = res.data
    if (res.data?.isAdmin) fetchJoinRequests()
  } catch {
    group.value = null
    toast.add({ title: 'Failed to load group info', color: 'error' })
  }
  loading.value = false
}

async function fetchJoinRequests() {
  try {
    const res: any = await api(`/sessions/${props.sessionId}/groups/requests`, {
      method: 'POST',
      body: { groupJid: props.groupJid }
    })
    joinRequests.value = res.data || []
  } catch {
    joinRequests.value = []
  }
}

watch(open, (val) => {
  if (val) fetchDetail()
})

async function leaveGroup() {
  leavingGroup.value = true
  try {
    await api(`/sessions/${props.sessionId}/groups/leave`, {
      method: 'POST',
      body: { groupJid: props.groupJid }
    })
    toast.add({ title: 'Left group', color: 'success' })
    open.value = false
    emit('left')
  } catch {
    toast.add({ title: 'Failed to leave group', color: 'error' })
  }
  leavingGroup.value = false
}

const settingsTab = useTemplateRef('settingsTab') as Ref<{ copyInviteLink: () => void } | null>
</script>

<template>
  <USlideover
    v-model:open="open"
    :title="group?.name || 'Group'"
    description="Manage group settings and participants"
    side="right"
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-16">
        <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
      </div>

      <div v-else-if="group">
        <UTabs :items="tabItems" default-value="info" variant="link" size="sm" class="w-full">
          <template #info>
            <SessionsGroupInfoTab :session-id="sessionId" :group="group" @updated="fetchDetail" />
          </template>

          <template #participants>
            <SessionsGroupParticipantsTab
              :session-id="sessionId"
              :group="group"
              :members="members"
              :join-requests="joinRequests"
              @updated="fetchDetail"
            />
          </template>

          <template #community>
            <SessionsGroupSubgroupsTab
              :session-id="sessionId"
              :group="group"
              :subgroups="subgroups"
              @updated="fetchDetail"
            />
          </template>

          <template #settings>
            <SessionsGroupSettingsTab
              ref="settingsTab"
              :session-id="sessionId"
              :group="group"
              @updated="fetchDetail"
            />
          </template>
        </UTabs>
      </div>
    </template>

    <template #footer>
      <div class="flex items-center gap-2 w-full">
        <UButton
          icon="i-lucide-link"
          label="Copy invite link"
          color="neutral"
          variant="outline"
          size="sm"
          @click="settingsTab?.copyInviteLink()"
        />
        <UButton
          icon="i-lucide-log-out"
          label="Leave group"
          color="error"
          variant="soft"
          size="sm"
          :loading="leavingGroup"
          class="ms-auto"
          @click="leaveGroup"
        />
      </div>
    </template>
  </USlideover>
</template>
