<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const props = defineProps<{ sessionId: string }>()

const { api } = useWzap()
const toast = useToast()
const open = ref(false)
const loading = ref(false)

interface CheckResult {
  jid: string
  isRegistered: boolean
  numberExists: boolean
}

const result = ref<CheckResult | null>(null)

const schema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number')
})

type Schema = z.output<typeof schema>

const state = reactive<Partial<Schema>>({ phone: '' })

async function onSubmit(event: FormSubmitEvent<Schema>) {
  loading.value = true
  result.value = null
  try {
    const phone = event.data.phone.replace(/\D/g, '')
    const res: { data: unknown } = await api(`/sessions/${props.sessionId}/contacts/check`, {
      method: 'POST',
      body: { phones: [phone] }
    })
    const data = res.data
    result.value = Array.isArray(data) ? data[0] : data
  } catch {
    toast.add({ title: 'Failed to check number', color: 'error' })
  }
  loading.value = false
}

watch(open, (val) => {
  if (!val) {
    state.phone = ''
    result.value = null
  }
})
</script>

<template>
  <UModal v-model:open="open" title="Check Number" description="Verify if a phone number is on WhatsApp">
    <UButton
      label="Check Number"
      icon="i-lucide-search"
      color="neutral"
      variant="outline"
    />

    <template #body>
      <UForm
        :schema="schema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <UFormField label="Phone number" name="phone" description="Digits only, with country code. E.g. 5511999999999">
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
            label="Check"
            color="primary"
            type="submit"
            :loading="loading"
            icon="i-lucide-search"
          />
        </div>

        <div v-if="result" class="rounded-lg border border-default p-4 space-y-2 text-sm">
          <div class="flex items-center justify-between">
            <span class="text-muted">Number</span>
            <span class="font-mono text-highlighted">{{ result.jid?.replace(/@.*$/, '') || state.phone }}</span>
          </div>
          <USeparator />
          <div class="flex items-center justify-between">
            <span class="text-muted">On WhatsApp</span>
            <UBadge
              :color="result.isRegistered || result.numberExists ? 'success' : 'error'"
              variant="subtle"
            >
              {{ result.isRegistered || result.numberExists ? 'Yes' : 'No' }}
            </UBadge>
          </div>
          <template v-if="result.jid">
            <USeparator />
            <div class="flex items-center justify-between">
              <span class="text-muted">JID</span>
              <span class="font-mono text-xs text-highlighted">{{ result.jid }}</span>
            </div>
          </template>
        </div>
      </UForm>
    </template>
  </UModal>
</template>
