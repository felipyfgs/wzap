<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const props = defineProps<{ sessionId: string, chatwootEnabled?: boolean }>()
const emit = defineEmits<{ updated: [] }>()

const { api } = useWzap()
const toast = useToast()

const loading = ref(true)
const saving = ref(false)
const removing = ref(false)
const hasConfig = ref(false)
const editing = ref(false)

interface ChatwootConfig {
  url: string
  accountId: number
  inboxId: number
  inboxName: string
  signMsg: boolean
  signDelimiter: string
  reopenConversation: boolean
  mergeBrContacts: boolean
  ignoreGroups: boolean
  webhookUrl: string
}

const config = ref<ChatwootConfig | null>(null)

const schema = z.object({
  url: z.string().url('Must be a valid URL'),
  accountId: z.coerce.number().int().min(1, 'Required'),
  token: z.string().min(1, 'Required'),
  inboxId: z.coerce.number().int().min(0).optional(),
  inboxName: z.string().optional(),
  signMsg: z.boolean(),
  signDelimiter: z.string().optional(),
  reopenConversation: z.boolean(),
  mergeBrContacts: z.boolean(),
  ignoreGroups: z.boolean(),
  autoCreateInbox: z.boolean()
})

type Schema = z.output<typeof schema>

const state = reactive<Partial<Schema>>({
  url: '',
  accountId: undefined,
  token: '',
  inboxId: undefined,
  inboxName: '',
  signMsg: false,
  signDelimiter: '',
  reopenConversation: false,
  mergeBrContacts: false,
  ignoreGroups: false,
  autoCreateInbox: false
})

async function fetchConfig() {
  loading.value = true
  try {
    const res: any = await api(`/sessions/${props.sessionId}/integrations/chatwoot`)
    if (res.data?.url) {
      config.value = res.data
      hasConfig.value = true
    } else {
      hasConfig.value = false
    }
  } catch {
    hasConfig.value = false
  }
  loading.value = false
}

async function onSubmit(event: FormSubmitEvent<Schema>) {
  saving.value = true
  try {
    await api(`/sessions/${props.sessionId}/integrations/chatwoot`, {
      method: 'PUT',
      body: event.data
    })
    toast.add({ title: 'Chatwoot configured', color: 'success' })
    editing.value = false
    await fetchConfig()
    emit('updated')
  } catch {
    toast.add({ title: 'Failed to save Chatwoot config', color: 'error' })
  }
  saving.value = false
}

async function removeConfig() {
  removing.value = true
  try {
    await api(`/sessions/${props.sessionId}/integrations/chatwoot`, { method: 'DELETE' })
    toast.add({ title: 'Chatwoot integration removed', color: 'success' })
    config.value = null
    hasConfig.value = false
    emit('updated')
  } catch {
    toast.add({ title: 'Failed to remove integration', color: 'error' })
  }
  removing.value = false
}

function startEditing() {
  state.url = config.value?.url || ''
  state.accountId = config.value?.accountId
  state.token = ''
  state.inboxId = config.value?.inboxId || undefined
  state.inboxName = config.value?.inboxName || ''
  state.signMsg = config.value?.signMsg ?? false
  state.signDelimiter = config.value?.signDelimiter || ''
  state.reopenConversation = config.value?.reopenConversation ?? false
  state.mergeBrContacts = config.value?.mergeBrContacts ?? false
  state.ignoreGroups = config.value?.ignoreGroups ?? false
  state.autoCreateInbox = false
  editing.value = true
}

watch(() => props.sessionId, fetchConfig, { immediate: true })
</script>

<template>
  <UPageCard
    title="Chatwoot Integration"
    description="Connect this session to a Chatwoot inbox."
    variant="naked"
    orientation="horizontal"
    class="mt-6 mb-0"
  />

  <UPageCard variant="subtle">
    <div v-if="loading" class="flex items-center justify-center py-8">
      <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
    </div>

    <!-- Configured state -->
    <template v-else-if="hasConfig && config && !editing">
      <div class="space-y-3 text-sm">
        <div class="flex items-center justify-between">
          <span class="text-muted">URL</span>
          <span class="font-medium text-highlighted">{{ config.url }}</span>
        </div>
        <USeparator />
        <div class="flex items-center justify-between">
          <span class="text-muted">Account ID</span>
          <span class="font-medium text-highlighted">{{ config.accountId }}</span>
        </div>
        <USeparator />
        <div class="flex items-center justify-between">
          <span class="text-muted">Inbox</span>
          <span class="font-medium text-highlighted">{{ config.inboxName || `#${config.inboxId}` }}</span>
        </div>
        <USeparator />
        <div class="flex items-center justify-between">
          <span class="text-muted">Sign Messages</span>
          <UBadge :color="config.signMsg ? 'success' : 'neutral'" variant="subtle" size="xs">
            {{ config.signMsg ? 'Yes' : 'No' }}
          </UBadge>
        </div>
        <USeparator />
        <div class="flex items-center justify-between">
          <span class="text-muted">Reopen Conversations</span>
          <UBadge :color="config.reopenConversation ? 'success' : 'neutral'" variant="subtle" size="xs">
            {{ config.reopenConversation ? 'Yes' : 'No' }}
          </UBadge>
        </div>
        <USeparator />
        <div class="flex items-center justify-between">
          <span class="text-muted">Webhook URL</span>
          <span class="font-mono text-xs text-highlighted truncate max-w-xs">{{ config.webhookUrl }}</span>
        </div>
      </div>
      <div class="flex items-center gap-2 mt-4">
        <UButton
          label="Edit"
          icon="i-lucide-pencil"
          size="sm"
          color="neutral"
          variant="outline"
          @click="startEditing"
        />
        <UButton
          label="Remove Integration"
          icon="i-lucide-trash"
          size="sm"
          color="error"
          variant="soft"
          :loading="removing"
          @click="removeConfig"
        />
      </div>
    </template>

    <!-- Configuration form -->
    <template v-else>
      <UForm
        :schema="schema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <div class="grid grid-cols-2 gap-3">
          <UFormField label="Chatwoot URL" name="url" required>
            <UInput v-model="state.url" placeholder="https://chatwoot.example.com" class="w-full" />
          </UFormField>
          <UFormField label="Account ID" name="accountId" required>
            <UInput
              v-model.number="state.accountId"
              type="number"
              placeholder="1"
              class="w-full"
            />
          </UFormField>
        </div>
        <UFormField label="API Token" name="token" required>
          <UInput
            v-model="state.token"
            type="password"
            placeholder="Token"
            class="w-full"
          />
        </UFormField>
        <div class="grid grid-cols-2 gap-3">
          <UFormField
            label="Inbox ID"
            name="inboxId"
            :hint="state.inboxId ? 'Uses existing inbox' : 'Leave empty to auto-create'"
          >
            <UInput
              v-model.number="state.inboxId"
              type="number"
              placeholder="Optional"
              class="w-full"
            />
          </UFormField>
          <UFormField label="Inbox Name" name="inboxName" :hint="!state.inboxId ? 'Used when creating a new inbox' : undefined">
            <UInput v-model="state.inboxName" placeholder="WhatsApp" class="w-full" :disabled="!!state.inboxId" />
          </UFormField>
        </div>

        <USeparator />

        <div class="grid grid-cols-2 gap-x-6 gap-y-2">
          <UFormField label="Sign Messages" name="signMsg">
            <USwitch v-model="state.signMsg" />
          </UFormField>
          <UFormField label="Reopen Conversations" name="reopenConversation">
            <USwitch v-model="state.reopenConversation" />
          </UFormField>
          <UFormField label="Merge BR Contacts" name="mergeBrContacts">
            <USwitch v-model="state.mergeBrContacts" />
          </UFormField>
          <UFormField label="Ignore Groups" name="ignoreGroups">
            <USwitch v-model="state.ignoreGroups" />
          </UFormField>
          <UFormField label="Auto Create Inbox" name="autoCreateInbox">
            <USwitch v-model="state.autoCreateInbox" />
          </UFormField>
        </div>

        <UFormField v-if="state.signMsg" label="Sign Delimiter" name="signDelimiter">
          <UInput v-model="state.signDelimiter" placeholder="-" class="w-full max-w-xs" />
        </UFormField>

        <div class="flex justify-end gap-2 pt-1">
          <UButton
            v-if="hasConfig"
            label="Cancel"
            color="neutral"
            variant="subtle"
            @click="editing = false"
          />
          <UButton
            label="Save"
            color="primary"
            type="submit"
            :loading="saving"
          />
        </div>
      </UForm>
    </template>
  </UPageCard>
</template>
