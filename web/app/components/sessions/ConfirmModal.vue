<script setup lang="ts">
withDefaults(defineProps<{
  title?: string
  description?: string
  confirmLabel?: string
  confirmColor?: 'error' | 'primary' | 'warning' | 'success' | 'neutral'
  icon?: string
}>(), {
  title: 'Confirm',
  description: 'Are you sure you want to proceed?',
  confirmLabel: 'Confirm',
  confirmColor: 'error',
  icon: 'i-lucide-alert-triangle'
})

const emit = defineEmits<{
  confirm: []
  cancel: []
}>()

const open = defineModel<boolean>('open', { default: false })
const loading = ref(false)

watch(open, (value) => {
  if (!value) {
    loading.value = false
  }
})

async function onConfirm() {
  loading.value = true
  emit('confirm')
}

function onCancel() {
  open.value = false
  emit('cancel')
}

function done() {
  loading.value = false
  open.value = false
}

defineExpose({ done })
</script>

<template>
  <UModal
    v-model:open="open"
    :title="title"
    :description="description"
    :icon="icon"
  >
    <slot />

    <template #body>
      <div class="flex justify-end gap-2">
        <UButton
          label="Cancel"
          color="neutral"
          variant="subtle"
          @click="onCancel"
        />
        <UButton
          :label="confirmLabel"
          :color="confirmColor"
          :loading="loading"
          @click="onConfirm"
        />
      </div>
    </template>
  </UModal>
</template>
