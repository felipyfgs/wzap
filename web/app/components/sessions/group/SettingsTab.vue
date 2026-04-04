<script setup lang="ts">
import type { GroupDetail } from './types'

const props = defineProps<{ sessionId: string, group: GroupDetail }>()
const emit = defineEmits<{ updated: [] }>()

const { api } = useWzap()
const toast = useToast()

const announce = ref(props.group.isAnnounce)
const locked = ref(props.group.isLocked)
const joinApproval = ref(props.group.joinApproval)
const ephemeralTimer = ref(String(props.group.ephemeralTimer ?? 0))
const inviteLink = ref('')
const loadingLink = ref(false)

watch(() => props.group, (g) => {
  announce.value = g.isAnnounce
  locked.value = g.isLocked
  joinApproval.value = g.joinApproval
  ephemeralTimer.value = String(g.ephemeralTimer ?? 0)
})

const ephemeralOptions = [
  { label: 'Off', value: '0' },
  { label: '24 hours', value: '86400' },
  { label: '7 days', value: '604800' },
  { label: '90 days', value: '7776000' }
]

async function toggleSetting(r: { value: boolean }, endpoint: string, value: boolean) {
  const prev = r.value
  r.value = value
  try {
    await api(`/sessions/${props.sessionId}/groups/${endpoint}`, {
      method: 'POST',
      body: { groupJid: props.group.jid, enabled: value }
    })
    toast.add({ title: 'Setting updated', color: 'success' })
  } catch {
    refs.value = prev
    toast.add({ title: 'Failed to update setting', color: 'error' })
  }
}

async function setEphemeral(val: string) {
  const prev = ephemeralTimer.value
  ephemeralTimer.value = val
  try {
    await api(`/sessions/${props.sessionId}/groups/ephemeral`, {
      method: 'POST',
      body: { groupJid: props.group.jid, duration: Number(val) }
    })
    toast.add({ title: 'Disappearing messages updated', color: 'success' })
  } catch {
    ephemeralTimer.value = prev
    toast.add({ title: 'Failed to update ephemeral timer', color: 'error' })
  }
}

async function fetchInviteLink(reset = false) {
  loadingLink.value = true
  try {
    const url = reset
      ? `/sessions/${props.sessionId}/groups/invite-link?reset=true`
      : `/sessions/${props.sessionId}/groups/invite-link`
    const res: any = await api(url, {
      method: 'POST',
      body: { groupJid: props.group.jid }
    })
    inviteLink.value = res.data?.link || ''
    if (reset) toast.add({ title: 'Invite link reset', color: 'success' })
  } catch {
    toast.add({ title: 'Failed to get invite link', color: 'error' })
  }
  loadingLink.value = false
}

async function copyInviteLink() {
  if (!inviteLink.value) await fetchInviteLink()
  if (inviteLink.value) {
    try {
      await navigator.clipboard.writeText(inviteLink.value)
      toast.add({ title: 'Invite link copied', color: 'success' })
    } catch {
      toast.add({ title: 'Failed to copy to clipboard', color: 'error' })
    }
  }
}

defineExpose({ copyInviteLink })
</script>

<template>
  <div class="space-y-4 pt-4">
    <div class="space-y-1.5">
      <p class="text-sm font-medium">
        Invite Link
      </p>
      <div class="flex gap-2">
        <UInput
          :model-value="inviteLink"
          readonly
          placeholder="Click copy to generate"
          size="sm"
          class="flex-1"
        />
        <UButton
          icon="i-lucide-copy"
          size="sm"
          color="neutral"
          :loading="loadingLink"
          @click="copyInviteLink"
        />
        <UButton
          icon="i-lucide-refresh-cw"
          size="sm"
          color="neutral"
          variant="ghost"
          :loading="loadingLink"
          @click="fetchInviteLink(true)"
        />
      </div>
    </div>

    <USeparator />

    <div class="space-y-3">
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm">
            Announce
          </p>
          <p class="text-xs text-muted">
            Only admins can send messages
          </p>
        </div>
        <USwitch :model-value="announce" @update:model-value="toggleSetting(announce, 'announce', $event)" />
      </div>
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm">
            Locked
          </p>
          <p class="text-xs text-muted">
            Only admins can edit group info
          </p>
        </div>
        <USwitch :model-value="locked" @update:model-value="toggleSetting(locked, 'locked', $event)" />
      </div>
      <div class="flex items-center justify-between">
        <div>
          <p class="text-sm">
            Join Approval
          </p>
          <p class="text-xs text-muted">
            Require admin approval to join
          </p>
        </div>
        <USwitch :model-value="joinApproval" @update:model-value="toggleSetting(joinApproval, 'join-approval', $event)" />
      </div>
    </div>

    <USeparator />

    <div class="flex items-center justify-between">
      <div>
        <p class="text-sm">
          Disappearing Messages
        </p>
        <p class="text-xs text-muted">
          Auto-delete messages after a period
        </p>
      </div>
      <USelect
        v-model="ephemeralTimer"
        :items="ephemeralOptions"
        value-key="value"
        size="sm"
        class="w-28"
        @update:model-value="setEphemeral"
      />
    </div>
  </div>
</template>
