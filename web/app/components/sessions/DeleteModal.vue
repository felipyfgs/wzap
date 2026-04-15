<script setup lang="ts">
const props = defineProps<{ count?: number }>()
const emit = defineEmits<{ confirmed: [] }>()

const open = ref(false)
const loading = ref(false)

async function onConfirm() {
  loading.value = true
  emit('confirmed')
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
    title="Delete Sessions"
    :description="`Are you sure you want to delete ${count} session(s)?`"
  >
    <slot />

    <template #body>
      <div class="flex justify-end gap-2">
        <UButton
          label="Cancel"
          color="neutral"
          variant="subtle"
          @click="open = false"
        />
        <UButton
          label="Delete"
          color="error"
          :loading="loading"
          @click="onConfirm"
        />
      </div>
    </template>
  </UModal>
</template>