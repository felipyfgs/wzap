<script setup lang="ts">
import type { DropdownMenuItem } from '@nuxt/ui'

const route = useRoute()
const { api } = useWzap()
const toast = useToast()

const sessionId = computed(() => route.params.id as string)
const { archiveChat, unarchiveChat, muteChat, unmuteChat, pinChat, unpinChat, deleteChat, markRead, markUnread } = useChatOperations(sessionId)

const labelModalOpen = ref(false)
const labelMode = ref<'add-chat' | 'remove-chat'>('add-chat')
const labelTargetJid = ref('')

interface Group {
  jid: string
  name: string
  participants: number
  subgroups?: number
  isAdmin: boolean
  isParent: boolean
}

const groups = ref<Group[]>([])
const loading = ref(true)
const search = ref('')
const typeFilter = ref('all')
const viewMode = ref<'grid' | 'list'>('list')

const typeFilterOptions = [
  { label: 'All', value: 'all' },
  { label: 'Groups', value: 'groups' },
  { label: 'Communities', value: 'communities' }
]

const slideoverGroupJid = ref('')
const slideoverOpen = ref(false)

const createOpen = ref(false)
const createName = ref('')
const createParticipants = ref('')
const creating = ref(false)

const joinOpen = ref(false)
const joinCode = ref('')
const joining = ref(false)
const previewing = ref(false)
const joinPreview = ref<{ name: string, topic?: string, participants: number } | null>(null)

const communityOpen = ref(false)
const communityName = ref('')
const communityDesc = ref('')
const creatingCommunity = ref(false)

const avatarCache = ref<Record<string, string | null>>({})
const avatarPending = new Set<string>()
const avatarQueue: string[] = []
let avatarActive = 0
const MAX_AVATAR_CONCURRENT = 3

async function processAvatarQueue() {
  while (avatarQueue.length > 0 && avatarActive < MAX_AVATAR_CONCURRENT) {
    const jid = avatarQueue.shift()!
    if (avatarCache.value[jid] !== undefined) continue
    avatarActive++
    try {
      const res: any = await api(`/sessions/${sessionId.value}/contacts/avatar`, {
        method: 'POST',
        body: { phone: jid }
      })
      avatarCache.value[jid] = res.data?.url || null
    } catch {
      avatarCache.value[jid] = null
    }
    avatarActive--
    processAvatarQueue()
  }
}

function fetchAvatarsForVisibleItems(virtualizer: any) {
  const items = virtualizer.getVirtualItems?.() || []
  const list = filtered.value
  for (const vItem of items) {
    const group = list[vItem.index]
    if (group && avatarCache.value[group.jid] === undefined && !avatarPending.has(group.jid)) {
      avatarPending.add(group.jid)
      avatarQueue.push(group.jid)
    }
  }
  processAvatarQueue()
}

async function fetchGroups() {
  loading.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}/groups`)
    groups.value = res.data || []
  } catch {
    groups.value = []
  }
  loading.value = false
}

const filtered = computed(() => {
  let result = groups.value
  if (typeFilter.value === 'groups') result = result.filter(g => !g.isParent)
  else if (typeFilter.value === 'communities') result = result.filter(g => g.isParent)
  const q = search.value.toLowerCase()
  if (q) result = result.filter(g => g.name?.toLowerCase().includes(q) || g.jid.toLowerCase().includes(q))
  return result
})

function openGroup(group: Group) {
  slideoverGroupJid.value = group.jid
  slideoverOpen.value = true
}

async function createGroup() {
  if (!createName.value.trim()) return
  creating.value = true
  try {
    const participants = createParticipants.value
      .split(/[,\n]/)
      .map(p => p.trim())
      .filter(Boolean)
    await api(`/sessions/${sessionId.value}/groups/create`, {
      method: 'POST',
      body: { name: createName.value.trim(), participants }
    })
    toast.add({ title: 'Group created', color: 'success' })
    createOpen.value = false
    createName.value = ''
    createParticipants.value = ''
    await fetchGroups()
  } catch {
    toast.add({ title: 'Failed to create group', color: 'error' })
  }
  creating.value = false
}

async function previewInvite() {
  if (!joinCode.value.trim()) return
  previewing.value = true
  try {
    const res: any = await api(`/sessions/${sessionId.value}/groups/invite-info`, {
      method: 'POST',
      body: { inviteCode: joinCode.value.trim() }
    })
    const d = res.data
    joinPreview.value = {
      name: d?.Name || d?.name || 'Unknown',
      topic: d?.Topic || d?.topic,
      participants: d?.Participants?.length || d?.participants?.length || 0
    }
  } catch {
    toast.add({ title: 'Failed to preview group', color: 'error' })
    joinPreview.value = null
  }
  previewing.value = false
}

async function joinGroup() {
  if (!joinCode.value.trim()) return
  joining.value = true
  try {
    await api(`/sessions/${sessionId.value}/groups/join`, {
      method: 'POST',
      body: { inviteCode: joinCode.value.trim() }
    })
    toast.add({ title: 'Joined group', color: 'success' })
    joinOpen.value = false
    joinCode.value = ''
    joinPreview.value = null
    await fetchGroups()
  } catch {
    toast.add({ title: 'Failed to join group', color: 'error' })
  }
  joining.value = false
}

async function createCommunity() {
  if (!communityName.value.trim()) return
  creatingCommunity.value = true
  try {
    await api(`/sessions/${sessionId.value}/community/create`, {
      method: 'POST',
      body: { name: communityName.value.trim(), description: communityDesc.value.trim() }
    })
    toast.add({ title: 'Community created', color: 'success' })
    communityOpen.value = false
    communityName.value = ''
    communityDesc.value = ''
    await fetchGroups()
  } catch {
    toast.add({ title: 'Failed to create community', color: 'error' })
  }
  creatingCommunity.value = false
}

async function wrapAction(fn: () => Promise<void>, label: string) {
  try {
    await fn()
    toast.add({ title: label, color: 'success' })
  } catch {
    toast.add({ title: `Failed: ${label}`, color: 'error' })
  }
}

function getGroupActions(g: Group): DropdownMenuItem[][] {
  return [
    [
      { label: 'Archive chat', icon: 'i-lucide-archive', onSelect: () => wrapAction(() => archiveChat(g.jid), 'Chat archived') },
      { label: 'Mute chat', icon: 'i-lucide-bell-off', onSelect: () => wrapAction(() => muteChat(g.jid), 'Chat muted') },
      { label: 'Pin chat', icon: 'i-lucide-pin', onSelect: () => wrapAction(() => pinChat(g.jid), 'Chat pinned') },
      { label: 'Mark unread', icon: 'i-lucide-eye-off', onSelect: () => wrapAction(() => markUnread(g.jid), 'Marked unread') }
    ],
    [
      { label: 'Add label', icon: 'i-lucide-tag', onSelect: () => { labelMode.value = 'add-chat'; labelTargetJid.value = g.jid; labelModalOpen.value = true } },
      { label: 'Remove label', icon: 'i-lucide-tag-off', onSelect: () => { labelMode.value = 'remove-chat'; labelTargetJid.value = g.jid; labelModalOpen.value = true } }
    ],
    [
      { label: 'Delete chat', icon: 'i-lucide-trash', color: 'error' as const, onSelect: () => wrapAction(() => deleteChat(g.jid), 'Chat deleted') }
    ]
  ]
}

watch(sessionId, fetchGroups, { immediate: true })
</script>

<template>
  <UDashboardPanel id="session-groups">
    <template #header>
      <UDashboardNavbar title="Groups">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            size="sm"
            :loading="loading"
            @click="fetchGroups"
          />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <template #left>
          <UInput
            v-model="search"
            icon="i-lucide-search"
            placeholder="Search groups…"
            class="max-w-xs"
          />
          <USelect
            v-model="typeFilter"
            :items="typeFilterOptions"
            value-key="value"
            size="sm"
            class="w-36"
          />
        </template>
        <template #right>
          <div class="flex items-center border border-default rounded-md">
            <UButton
              icon="i-lucide-list"
              size="xs"
              :color="viewMode === 'list' ? 'primary' : 'neutral'"
              :variant="viewMode === 'list' ? 'solid' : 'ghost'"
              class="rounded-r-none"
              @click="viewMode = 'list'"
            />
            <UButton
              icon="i-lucide-layout-grid"
              size="xs"
              :color="viewMode === 'grid' ? 'primary' : 'neutral'"
              :variant="viewMode === 'grid' ? 'solid' : 'ghost'"
              class="rounded-l-none"
              @click="viewMode = 'grid'"
            />
          </div>
          <UButton
            icon="i-lucide-link"
            label="Join"
            size="sm"
            color="neutral"
            variant="outline"
            @click="joinOpen = true"
          />
          <UButton
            icon="i-lucide-network"
            label="New Community"
            size="sm"
            color="neutral"
            variant="outline"
            @click="communityOpen = true"
          />
          <UButton
            icon="i-lucide-plus"
            label="New Group"
            size="sm"
            color="primary"
            @click="createOpen = true"
          />
        </template>
      </UDashboardToolbar>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <UEmpty
        v-else-if="!filtered.length"
        icon="i-lucide-users-2"
        title="No groups found"
        :description="search ? 'Try adjusting your search or filters.' : 'This session has no groups yet.'"
      />

      <template v-else>
        <p class="text-xs text-muted mb-3">
          {{ filtered.length }} group(s)
        </p>

        <!-- List view: virtualized ScrollArea -->
        <UScrollArea
          v-if="viewMode === 'list'"
          :items="filtered"
          :virtualize="{ estimateSize: 56, skipMeasurement: true, overscan: 10, onChange: fetchAvatarsForVisibleItems }"
          class="h-[calc(100vh-220px)]"
        >
          <template #default="{ item: group }">
            <div
              class="flex items-center gap-3 px-3 py-2.5 cursor-pointer hover:bg-elevated/50 transition-colors border-b border-default"
              @click="openGroup(group)"
            >
              <img
                v-if="avatarCache[group.jid]"
                :src="avatarCache[group.jid]!"
                :alt="group.name"
                class="size-9 shrink-0 rounded-full object-cover"
              >
              <div
                v-else
                class="size-9 shrink-0 rounded-full bg-primary/10 flex items-center justify-center text-xs font-bold text-primary"
              >
                {{ (group.name || '??').slice(0, 2).toUpperCase() }}
              </div>

              <div class="flex-1 min-w-0">
                <p class="text-sm font-medium text-highlighted truncate">
                  {{ group.name || 'Unnamed' }}
                </p>
                <div class="flex items-center gap-2 text-xs text-muted">
                  <span class="flex items-center gap-0.5">
                    <UIcon name="i-lucide-users-2" class="size-3" />
                    {{ group.participants }}
                  </span>
                  <span v-if="group.isParent && group.subgroups" class="flex items-center gap-0.5">
                    <UIcon name="i-lucide-network" class="size-3" />
                    {{ group.subgroups }}
                  </span>
                </div>
              </div>

              <div class="flex items-center gap-1 shrink-0">
                <UBadge
                  v-if="group.isParent"
                  label="Community"
                  color="info"
                  variant="subtle"
                  size="xs"
                />
                <UBadge
                  :label="group.isAdmin ? 'Admin' : 'Member'"
                  :color="group.isAdmin ? 'success' : 'neutral'"
                  variant="subtle"
                  size="xs"
                />
              </div>

              <UDropdownMenu :items="getGroupActions(group)" @click.stop>
                <UButton
                  icon="i-lucide-ellipsis-vertical"
                  color="neutral"
                  variant="ghost"
                  size="xs"
                  @click.stop
                />
              </UDropdownMenu>
              <UIcon name="i-lucide-chevron-right" class="size-4 text-muted shrink-0" />
            </div>
          </template>
        </UScrollArea>

        <!-- Grid view: virtualized with lanes -->
        <UScrollArea
          v-else
          :items="filtered"
          :virtualize="{ estimateSize: 100, lanes: 3, gap: 12, skipMeasurement: false, overscan: 6, onChange: fetchAvatarsForVisibleItems }"
          class="h-[calc(100vh-220px)]"
        >
          <template #default="{ item: group }">
            <div
              class="rounded-lg border border-default p-3 cursor-pointer hover:ring-1 hover:ring-primary/50 transition-all"
              @click="openGroup(group)"
            >
              <div class="flex items-center gap-2 mb-2">
                <img
                  v-if="avatarCache[group.jid]"
                  :src="avatarCache[group.jid]!"
                  :alt="group.name"
                  class="size-8 shrink-0 rounded-full object-cover"
                >
                <div
                  v-else
                  class="size-8 shrink-0 rounded-full bg-primary/10 flex items-center justify-center text-xs font-bold text-primary"
                >
                  {{ (group.name || '??').slice(0, 2).toUpperCase() }}
                </div>
                <p class="font-medium text-sm text-highlighted truncate flex-1">
                  {{ group.name || 'Unnamed' }}
                </p>
                <UDropdownMenu :items="getGroupActions(group)" @click.stop>
                  <UButton
                    icon="i-lucide-ellipsis-vertical"
                    color="neutral"
                    variant="ghost"
                    size="xs"
                    @click.stop
                  />
                </UDropdownMenu>
              </div>
              <div class="flex items-center justify-between">
                <div class="flex items-center gap-2 text-xs text-muted">
                  <span class="flex items-center gap-0.5">
                    <UIcon name="i-lucide-users-2" class="size-3" />
                    {{ group.participants }}
                  </span>
                  <span v-if="group.isParent && group.subgroups" class="flex items-center gap-0.5">
                    <UIcon name="i-lucide-network" class="size-3" />
                    {{ group.subgroups }}
                  </span>
                </div>
                <div class="flex items-center gap-1 shrink-0">
                  <UBadge
                    v-if="group.isParent"
                    label="Community"
                    color="info"
                    variant="subtle"
                    size="xs"
                  />
                  <UBadge
                    :label="group.isAdmin ? 'Admin' : 'Member'"
                    :color="group.isAdmin ? 'success' : 'neutral'"
                    variant="subtle"
                    size="xs"
                  />
                </div>
              </div>
            </div>
          </template>
        </UScrollArea>
      </template>

      <SessionsGroupSlideover
        v-model:open="slideoverOpen"
        :session-id="sessionId"
        :group-jid="slideoverGroupJid"
        @left="fetchGroups"
      />

      <!-- Create Group Modal -->
      <UModal v-model:open="createOpen" title="New Group" description="Create a new WhatsApp group.">
        <template #body>
          <div class="space-y-4">
            <UFormField label="Group name" required>
              <UInput v-model="createName" placeholder="My Group" class="w-full" />
            </UFormField>
            <UFormField label="Participants" description="Phone numbers separated by commas or new lines.">
              <UTextarea
                v-model="createParticipants"
                placeholder="5511999999999&#10;5511888888888"
                :rows="3"
                class="w-full"
              />
            </UFormField>
          </div>
        </template>
        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton
              label="Cancel"
              color="neutral"
              variant="subtle"
              @click="createOpen = false"
            />
            <UButton
              label="Create"
              color="primary"
              :loading="creating"
              @click="createGroup"
            />
          </div>
        </template>
      </UModal>

      <!-- Join Group Modal -->
      <UModal v-model:open="joinOpen" title="Join Group" description="Enter an invite code to preview and join a group.">
        <template #body>
          <div class="space-y-4">
            <UFormField label="Invite code" required>
              <div class="flex gap-2">
                <UInput v-model="joinCode" placeholder="AbCdEfGhIjK" class="flex-1" />
                <UButton
                  label="Preview"
                  size="sm"
                  color="neutral"
                  variant="outline"
                  :loading="previewing"
                  @click="previewInvite"
                />
              </div>
            </UFormField>

            <div v-if="joinPreview" class="rounded-lg border border-default p-3 space-y-1.5">
              <p class="font-medium text-highlighted">
                {{ joinPreview.name }}
              </p>
              <p v-if="joinPreview.topic" class="text-sm text-muted">
                {{ joinPreview.topic }}
              </p>
              <div class="flex items-center gap-1 text-xs text-muted">
                <UIcon name="i-lucide-users-2" class="size-3.5" />
                {{ joinPreview.participants }} participants
              </div>
            </div>
          </div>
        </template>
        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton
              label="Cancel"
              color="neutral"
              variant="subtle"
              @click="joinOpen = false; joinPreview = null"
            />
            <UButton
              label="Join"
              color="primary"
              :loading="joining"
              @click="joinGroup"
            />
          </div>
        </template>
      </UModal>

      <!-- New Community Modal -->
      <UModal v-model:open="communityOpen" title="New Community" description="Create a new WhatsApp community.">
        <template #body>
          <div class="space-y-4">
            <UFormField label="Community name" required>
              <UInput v-model="communityName" placeholder="My Community" class="w-full" />
            </UFormField>
            <UFormField label="Description">
              <UTextarea
                v-model="communityDesc"
                placeholder="What is this community about?"
                :rows="3"
                class="w-full"
              />
            </UFormField>
          </div>
        </template>
        <template #footer>
          <div class="flex justify-end gap-2">
            <UButton
              label="Cancel"
              color="neutral"
              variant="subtle"
              @click="communityOpen = false"
            />
            <UButton
              label="Create"
              color="primary"
              :loading="creatingCommunity"
              @click="createCommunity"
            />
          </div>
        </template>
      </UModal>

      <SessionsLabelActionModal
        v-model:open="labelModalOpen"
        :session-id="sessionId"
        :jid="labelTargetJid"
        :mode="labelMode"
      />
    </template>
  </UDashboardPanel>
</template>
