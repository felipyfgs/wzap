<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const { api, apiBase, setApiBase, token, setToken } = useWzap()
const toast = useToast()

const schema = z.object({
  apiBase: z.string().url('Must be a valid URL'),
  token: z.string().min(1, 'Token is required')
})

type Schema = z.output<typeof schema>

const state = reactive<Schema>({
  apiBase: apiBase.value,
  token: token.value
})

const testing = ref(false)
const healthResult = ref<{ status: string; services: Record<string, boolean> } | null>(null)

async function onSubmit(event: FormSubmitEvent<Schema>) {
  setApiBase(event.data.apiBase)
  setToken(event.data.token)
  toast.add({ title: 'Settings saved', color: 'success' })
}

async function testConnection() {
  testing.value = true
  healthResult.value = null
  try {
    setApiBase(state.apiBase)
    setToken(state.token)
    const res: any = await api('/health')
    healthResult.value = res.data
  } catch {
    healthResult.value = { status: 'UNREACHABLE', services: {} }
  }
  testing.value = false
}
</script>

<template>
  <UForm id="settings-general" :schema="schema" :state="state" @submit="onSubmit">
    <UPageCard
      title="API Connection"
      description="Configure the wzap API endpoint and authentication token."
      variant="naked"
      orientation="horizontal"
      class="mb-4"
    >
      <UButton
        form="settings-general"
        label="Save"
        color="neutral"
        type="submit"
        class="w-fit lg:ms-auto"
      />
    </UPageCard>

    <UPageCard variant="subtle">
      <UFormField
        name="apiBase"
        label="API Base URL"
        description="The base URL of your wzap instance, e.g. http://localhost:8080"
        required
        class="flex max-sm:flex-col justify-between items-start gap-4"
      >
        <UInput v-model="state.apiBase" placeholder="http://localhost:8080" class="w-full max-w-xs" />
      </UFormField>

      <USeparator />

      <UFormField
        name="token"
        label="Admin Token"
        description="The admin token set in your wzap configuration."
        required
        class="flex max-sm:flex-col justify-between items-start gap-4"
      >
        <UInput v-model="state.token" type="password" placeholder="your-admin-token" autocomplete="off" class="w-full max-w-xs" />
      </UFormField>

      <USeparator />

      <div class="flex max-sm:flex-col justify-between sm:items-center gap-4">
        <div>
          <p class="text-sm font-medium">Connection Status</p>
          <p class="text-sm text-muted">Test the connection to your wzap API.</p>
        </div>
        <div class="flex items-center gap-3">
          <template v-if="healthResult">
            <UBadge
              :color="healthResult.status === 'UP' ? 'success' : healthResult.status === 'DEGRADED' ? 'warning' : 'error'"
              variant="subtle"
              size="lg"
            >
              {{ healthResult.status }}
            </UBadge>
            <div v-if="healthResult.services" class="flex gap-2 text-xs text-muted">
              <span v-for="(ok, svc) in healthResult.services" :key="svc" class="flex items-center gap-1">
                <UIcon :name="ok ? 'i-lucide-check-circle' : 'i-lucide-x-circle'" :class="ok ? 'text-success' : 'text-error'" class="size-3" />
                {{ svc }}
              </span>
            </div>
          </template>
          <UButton
            label="Test Connection"
            icon="i-lucide-plug"
            color="neutral"
            variant="outline"
            :loading="testing"
            @click="testConnection"
          />
        </div>
      </div>
    </UPageCard>
  </UForm>
</template>
