<script setup lang="ts">
defineProps<{
  state: Record<string, unknown>
  pollOptions: string[]
}>()

const emit = defineEmits<{
  addOption: []
  removeOption: [index: number]
}>()
</script>

<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <UFormField label="Question" name="name">
    <UInput v-model="state.name" placeholder="What do you prefer?" class="w-full" />
  </UFormField>
  <div class="space-y-2">
    <p class="text-sm font-medium text-highlighted">
      Options
    </p>
    <div v-for="(_, i) in pollOptions" :key="i" class="flex items-center gap-2">
      <UInput v-model="pollOptions[i]" :placeholder="`Option ${i + 1}`" class="flex-1" />
      <UButton
        icon="i-lucide-x"
        color="neutral"
        variant="ghost"
        size="sm"
        :disabled="pollOptions.length <= 2"
        @click="emit('removeOption', i)"
      />
    </div>
    <UButton
      label="Add Option"
      icon="i-lucide-plus"
      color="neutral"
      variant="subtle"
      size="sm"
      @click="emit('addOption')"
    />
  </div>
  <UFormField label="Selectable Count" name="selectableCount" hint="Leave blank for default">
    <UInput
      v-model="state.selectableCount"
      type="number"
      min="1"
      placeholder="1"
      class="w-full"
    />
  </UFormField>
  <!-- eslint-enable vue/no-mutating-props -->
</template>
