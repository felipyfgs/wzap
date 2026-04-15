<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const props = defineProps<{ sessionId: string }>()
const emit = defineEmits<{ paired: [] }>()

const { api } = useWzap()
const toast = useToast()
const open = ref(false)
const loading = ref(false)
const pairingCode = ref('')

const schema = z.object({
  phone: z.string().min(8, 'Enter a valid phone number')
})

type Schema = z.output<typeof schema>

const state = reactive<Partial<Schema>>({ phone: '' })

async function onSubmit(event: FormSubmitEvent<Schema>) {
  loading.value = true
  pairingCode.value = ''
  try {
    const res: { data: unknown } = await api(`/sessions/${props.sessionId}/pair`, {
      method: 'POST',
      body: { phone: event.data.phone.replace(/\D/g, '') }
    })
    pairingCode.value = res.data?.pairingCode || ''
    if (pairingCode.value) {
      toast.add({ title: 'Pairing code generated', color: 'success' })
    }
    emit('paired')
  } catch {
    toast.add({ title: 'Failed to generate pairing code', color: 'error' })
  }
  loading.value = false
}

function show() {
  pairingCode.value = ''
  state.phone = ''
  open.value = true
}

defineExpose({ show })
</script>

<template>
  <UModal v-model:open="open" title="Pair by Phone" description="Enter your phone number to get a pairing code">
    <template #body>
      <div v-if="pairingCode" class="flex flex-col items-center gap-4 py-4">
        <p class="text-sm text-muted">
          Enter this code on your WhatsApp device:
        </p>
        <div class="flex items-center gap-2">
          <span class="text-3xl font-mono font-bold tracking-widest text-highlighted">{{ pairingCode }}</span>
          <UButton
            icon="i-lucide-copy"
            color="neutral"
            variant="ghost"
            size="sm"
            @click="navigator.clipboard.writeText(pairingCode); toast.add({ title: 'Code copied', color: 'success' })"
          />
        </div>
        <UButton
          label="Close"
          color="neutral"
          variant="subtle"
          @click="open = false"
        />
      </div>

      <UForm
        v-else
        :schema="schema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <UFormField label="Phone Number" name="phone" required>
          <UInput v-model="state.phone" placeholder="5511999999999" class="w-full" />
        </UFormField>
        <div class="flex justify-end gap-2">
          <UButton
            label="Cancel"
            color="neutral"
            variant="subtle"
            @click="open = false"
          />
          <UButton
            label="Get Code"
            color="primary"
            type="submit"
            :loading="loading"
          />
        </div>
      </UForm>
    </template>
  </UModal>
</template>
