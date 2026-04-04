<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const props = defineProps<{ sessionId: string }>()
const emit = defineEmits<{ created: [] }>()

const { createNewsletter } = useNewsletter(props.sessionId)
const toast = useToast()
const open = ref(false)
const loading = ref(false)

const schema = z.object({
  name: z.string().min(1, 'Required'),
  description: z.string().optional()
})

type Schema = z.output<typeof schema>

const state = reactive<Partial<Schema>>({ name: '', description: '' })

async function onSubmit(event: FormSubmitEvent<Schema>) {
  loading.value = true
  try {
    await createNewsletter({ name: event.data.name, description: event.data.description })
    toast.add({ title: 'Newsletter created', color: 'success' })
    state.name = ''
    state.description = ''
    open.value = false
    emit('created')
  } catch {
    toast.add({ title: 'Failed to create newsletter', color: 'error' })
  }
  loading.value = false
}
</script>

<template>
  <UModal v-model:open="open" title="Create Newsletter" description="Create a new WhatsApp channel">
    <UButton label="Create Newsletter" icon="i-lucide-plus" color="primary" />

    <template #body>
      <UForm
        :schema="schema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <UFormField label="Name" name="name" required>
          <UInput v-model="state.name" placeholder="My Channel" class="w-full" />
        </UFormField>
        <UFormField label="Description" name="description">
          <UTextarea v-model="state.description" placeholder="What is this channel about?" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton
            label="Cancel"
            color="neutral"
            variant="subtle"
            @click="open = false"
          />
          <UButton
            label="Create"
            color="primary"
            type="submit"
            :loading="loading"
          />
        </div>
      </UForm>
    </template>
  </UModal>
</template>
