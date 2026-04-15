<script setup lang="ts">
import type { ListSection } from '~/composables/useMessageSender'

defineProps<{
  state: Record<string, unknown>
  sections: ListSection[]
}>()

const emit = defineEmits<{
  addSection: []
  removeSection: [index: number]
  addRow: [sectionIndex: number]
  removeRow: [sectionIndex: number, rowIndex: number]
}>()
</script>

<template>
  <!-- eslint-disable vue/no-mutating-props -->
  <UFormField label="Title" name="title">
    <UInput v-model="state.title" placeholder="List title" class="w-full" />
  </UFormField>
  <UFormField label="Body" name="body">
    <UTextarea
      v-model="state.body"
      :rows="2"
      autoresize
      placeholder="List body text"
      class="w-full"
    />
  </UFormField>
  <div class="grid grid-cols-2 gap-3">
    <UFormField label="Footer" name="footer">
      <UInput v-model="state.footer" placeholder="Footer (optional)" class="w-full" />
    </UFormField>
    <UFormField label="Button Text" name="buttonText">
      <UInput v-model="state.buttonText" placeholder="Menu" class="w-full" />
    </UFormField>
  </div>

  <USeparator />

  <div class="space-y-4">
    <div class="flex items-center justify-between">
      <p class="text-sm font-medium text-highlighted">
        Sections
      </p>
      <UButton
        label="Add Section"
        icon="i-lucide-plus"
        color="neutral"
        variant="subtle"
        size="sm"
        @click="emit('addSection')"
      />
    </div>

    <div v-for="(section, si) in sections" :key="si" class="space-y-2 rounded-md border border-default p-3">
      <div class="flex items-center gap-2">
        <UInput v-model="section.title" placeholder="Section title" class="flex-1" />
        <UButton
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          size="sm"
          :disabled="sections.length <= 1"
          @click="emit('removeSection', si)"
        />
      </div>

      <div v-for="(row, ri) in section.rows" :key="ri" class="flex items-center gap-2 pl-4">
        <UInput v-model="row.id" placeholder="ID" class="w-20" />
        <UInput v-model="row.title" placeholder="Row title" class="flex-1" />
        <UInput v-model="row.description" placeholder="Description" class="flex-1" />
        <UButton
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          size="sm"
          :disabled="section.rows.length <= 1"
          @click="emit('removeRow', si, ri)"
        />
      </div>
      <div class="pl-4">
        <UButton
          label="Add Row"
          icon="i-lucide-plus"
          color="neutral"
          variant="ghost"
          size="sm"
          @click="emit('addRow', si)"
        />
      </div>
    </div>
  </div>
  <!-- eslint-enable vue/no-mutating-props -->
</template>
