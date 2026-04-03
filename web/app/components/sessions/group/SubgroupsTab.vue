<script setup lang="ts">
import type { GroupDetail, Subgroup } from './types'

const props = defineProps<{
  sessionId: string
  group: GroupDetail
  subgroups: Subgroup[]
}>()
const emit = defineEmits<{ updated: [] }>()

const { api } = useWzap()
const toast = useToast()

const addSubgroupJid = ref('')
const addingSubgroup = ref(false)

async function addSubgroup() {
  if (!addSubgroupJid.value.trim()) return
  addingSubgroup.value = true
  try {
    await api(`/sessions/${props.sessionId}/community/participant/add`, {
      method: 'POST',
      body: { communityJid: props.group.jid, participants: [addSubgroupJid.value.trim()] }
    })
    toast.add({ title: 'Subgroup added', color: 'success' })
    addSubgroupJid.value = ''
    emit('updated')
  } catch {
    toast.add({ title: 'Failed to add subgroup', color: 'error' })
  }
  addingSubgroup.value = false
}

async function removeSubgroup(jid: string) {
  try {
    await api(`/sessions/${props.sessionId}/community/participant/remove`, {
      method: 'POST',
      body: { communityJid: props.group.jid, participants: [jid] }
    })
    toast.add({ title: 'Subgroup removed', color: 'success' })
    emit('updated')
  } catch {
    toast.add({ title: 'Failed to remove subgroup', color: 'error' })
  }
}
</script>

<template>
  <div class="space-y-3 pt-4">
    <div v-if="group.isAdmin" class="flex gap-2">
      <UInput v-model="addSubgroupJid" placeholder="120363...@g.us" size="sm" class="flex-1" />
      <UButton icon="i-lucide-plus" size="sm" color="neutral" :loading="addingSubgroup" @click="addSubgroup" />
    </div>

    <div v-for="sg in subgroups" :key="sg.jid" class="flex items-center justify-between gap-2 py-2 border-b border-default last:border-b-0">
      <div class="min-w-0">
        <p class="text-sm font-medium text-highlighted truncate">{{ sg.name || 'Unnamed' }}</p>
        <p class="text-xs text-muted font-mono truncate">{{ sg.jid }}</p>
      </div>
      <UButton v-if="group.isAdmin" icon="i-lucide-x" size="xs" color="error" variant="ghost" @click="removeSubgroup(sg.jid)" />
    </div>

    <div v-if="!subgroups.length" class="text-sm text-muted py-4 text-center">
      No subgroups linked.
    </div>
  </div>
</template>
