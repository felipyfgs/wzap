<script setup lang="ts">
import type { GroupDetail } from './types'

const props = defineProps<{ sessionId: string; group: GroupDetail }>()
const emit = defineEmits<{ updated: [] }>()

const { api } = useWzap()
const toast = useToast()

const editName = ref(props.group.name ?? '')
const editTopic = ref(props.group.topic ?? '')
const savingName = ref(false)
const savingTopic = ref(false)
const uploadingPhoto = ref(false)
const removingPhoto = ref(false)
const photoInputRef = ref<HTMLInputElement | null>(null)

watch(() => props.group, (g) => {
  editName.value = g.name ?? ''
  editTopic.value = g.topic ?? ''
})

async function saveName() {
  savingName.value = true
  try {
    await api(`/sessions/${props.sessionId}/groups/name`, {
      method: 'POST',
      body: { groupJid: props.group.jid, text: editName.value }
    })
    toast.add({ title: 'Name updated', color: 'success' })
    emit('updated')
  } catch {
    toast.add({ title: 'Failed to update name', color: 'error' })
  }
  savingName.value = false
}

async function saveTopic() {
  savingTopic.value = true
  try {
    await api(`/sessions/${props.sessionId}/groups/description`, {
      method: 'POST',
      body: { groupJid: props.group.jid, text: editTopic.value }
    })
    toast.add({ title: 'Description updated', color: 'success' })
    emit('updated')
  } catch {
    toast.add({ title: 'Failed to update description', color: 'error' })
  }
  savingTopic.value = false
}

function toBase64(file: File): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader()
    reader.onload = () => {
      const result = reader.result as string
      resolve(result.split(',')[1] || '')
    }
    reader.onerror = reject
    reader.readAsDataURL(file)
  })
}

async function uploadPhoto(event: Event) {
  const file = (event.target as HTMLInputElement).files?.[0]
  if (!file) return
  uploadingPhoto.value = true
  try {
    const b64 = await toBase64(file)
    await api(`/sessions/${props.sessionId}/groups/photo`, {
      method: 'POST',
      body: { groupJid: props.group.jid, image: b64 }
    })
    toast.add({ title: 'Photo updated', color: 'success' })
  } catch {
    toast.add({ title: 'Failed to update photo', color: 'error' })
  }
  uploadingPhoto.value = false
  if (photoInputRef.value) photoInputRef.value.value = ''
}

async function removePhoto() {
  removingPhoto.value = true
  try {
    await api(`/sessions/${props.sessionId}/groups/photo/remove`, {
      method: 'POST',
      body: { groupJid: props.group.jid }
    })
    toast.add({ title: 'Photo removed', color: 'success' })
  } catch {
    toast.add({ title: 'Failed to remove photo', color: 'error' })
  }
  removingPhoto.value = false
}
</script>

<template>
  <div class="space-y-4 pt-4">
    <div class="flex items-center gap-2">
      <UBadge :label="group.isAdmin ? 'Admin' : 'Member'" :color="group.isAdmin ? 'success' : 'neutral'" variant="subtle" size="sm" />
      <UBadge v-if="group.isParent" label="Community" color="info" variant="subtle" size="sm" />
      <span class="text-xs font-mono text-muted truncate">{{ group.jid }}</span>
    </div>

    <div v-if="group.createdAt" class="text-xs text-muted">
      Created {{ new Date(group.createdAt).toLocaleDateString() }}
    </div>

    <div v-if="group.isAdmin" class="flex items-center gap-3">
      <input ref="photoInputRef" type="file" accept="image/*" class="hidden" @change="uploadPhoto">
      <UButton icon="i-lucide-camera" label="Change photo" size="xs" color="neutral" variant="outline" :loading="uploadingPhoto" @click="photoInputRef?.click()" />
      <UButton icon="i-lucide-trash-2" label="Remove" size="xs" color="error" variant="ghost" :loading="removingPhoto" @click="removePhoto" />
    </div>

    <USeparator />

    <div class="space-y-1.5">
      <p class="text-sm font-medium">Name</p>
      <div v-if="group.isAdmin" class="flex gap-2">
        <UInput v-model="editName" class="flex-1" size="sm" />
        <UButton icon="i-lucide-check" size="sm" color="neutral" :loading="savingName" @click="saveName" />
      </div>
      <p v-else class="text-sm text-muted">{{ group.name }}</p>
    </div>

    <div class="space-y-1.5">
      <p class="text-sm font-medium">Description</p>
      <div v-if="group.isAdmin" class="space-y-2">
        <UTextarea v-model="editTopic" :rows="3" size="sm" />
        <UButton icon="i-lucide-check" label="Save" size="xs" color="neutral" :loading="savingTopic" @click="saveTopic" />
      </div>
      <p v-else class="text-sm text-muted">{{ group.topic || 'No description' }}</p>
    </div>
  </div>
</template>
