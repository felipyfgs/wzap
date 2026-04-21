<script setup lang="ts">
defineProps<{ sessionId: string, sessionName: string }>()
const emit = defineEmits<{ updated: [] }>()

const open = ref(false)
function show() {
  open.value = true
}
defineExpose({ show })
</script>

<template>
  <UModal
    v-model:open="open"
    :title="`Chatwoot — ${sessionName}`"
    description="Configure the Chatwoot inbox integration for this session."
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <SessionsChatwootConfigCard
        :session-id="sessionId"
        @updated="emit('updated')"
      />
    </template>
  </UModal>
</template>
