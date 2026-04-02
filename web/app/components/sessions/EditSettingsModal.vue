<script setup lang="ts">
import type { Session } from '~/types'

const props = defineProps<{ session: Session }>()
const emit = defineEmits<{ saved: [] }>()

const { api } = useWzap()
const toast = useToast()
const open = ref(false)
const loading = ref(false)

const state = reactive({
  alwaysOnline: false,
  readMessages: false,
  rejectCall: false,
  msgRejectCall: '',
  ignoreGroups: false,
  ignoreStatus: false
})

watch(open, (val) => {
  if (val && props.session.settings) {
    state.alwaysOnline = props.session.settings.alwaysOnline
    state.readMessages = props.session.settings.readMessages
    state.rejectCall = props.session.settings.rejectCall
    state.msgRejectCall = props.session.settings.msgRejectCall || ''
    state.ignoreGroups = props.session.settings.ignoreGroups
    state.ignoreStatus = props.session.settings.ignoreStatus
  }
})

async function save() {
  loading.value = true
  try {
    await api(`/sessions/${props.session.id}`, {
      method: 'PUT',
      body: {
        settings: {
          alwaysOnline: state.alwaysOnline,
          readMessages: state.readMessages,
          rejectCall: state.rejectCall,
          msgRejectCall: state.rejectCall ? state.msgRejectCall : '',
          ignoreGroups: state.ignoreGroups,
          ignoreStatus: state.ignoreStatus
        }
      }
    })
    toast.add({ title: 'Settings saved', color: 'success' })
    open.value = false
    emit('saved')
  } catch {
    toast.add({ title: 'Failed to save settings', color: 'error' })
  }
  loading.value = false
}

defineExpose({ open: () => { open.value = true } })
</script>

<template>
  <UModal v-model:open="open" title="Edit Settings" description="Configure session behavior">
    <template #body>
      <div class="space-y-4">
        <div class="space-y-3">
          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 text-sm">
              <UIcon name="i-lucide-wifi" class="size-4 text-muted" />
              <span>Always Online</span>
            </div>
            <UToggle v-model="state.alwaysOnline" />
          </div>

          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 text-sm">
              <UIcon name="i-lucide-check-check" class="size-4 text-muted" />
              <span>Auto Read Messages</span>
            </div>
            <UToggle v-model="state.readMessages" />
          </div>

          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 text-sm">
              <UIcon name="i-lucide-phone-off" class="size-4 text-muted" />
              <span>Reject Calls</span>
            </div>
            <UToggle v-model="state.rejectCall" />
          </div>

          <div v-if="state.rejectCall" class="pl-6">
            <UFormField label="Reject message (optional)" name="msgRejectCall">
              <UInput
                v-model="state.msgRejectCall"
                placeholder="Sorry, I can't answer calls."
                class="w-full"
              />
            </UFormField>
          </div>

          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 text-sm">
              <UIcon name="i-lucide-users-2" class="size-4 text-muted" />
              <span>Ignore Groups</span>
            </div>
            <UToggle v-model="state.ignoreGroups" />
          </div>

          <div class="flex items-center justify-between">
            <div class="flex items-center gap-2 text-sm">
              <UIcon name="i-lucide-circle-dashed" class="size-4 text-muted" />
              <span>Ignore Status</span>
            </div>
            <UToggle v-model="state.ignoreStatus" />
          </div>
        </div>

        <div class="flex justify-end gap-2 pt-2">
          <UButton label="Cancel" color="neutral" variant="subtle" @click="open = false" />
          <UButton label="Save" color="primary" :loading="loading" @click="save" />
        </div>
      </div>
    </template>
  </UModal>
</template>
