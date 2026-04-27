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
    :title="`Elodesk — ${sessionName}`"
    :ui="{ content: 'sm:max-w-2xl' }"
  >
    <template #body>
      <SessionsElodeskConfigCard
        :session-id="sessionId"
        @updated="emit('updated')"
      />
    </template>
  </UModal>
</template>
