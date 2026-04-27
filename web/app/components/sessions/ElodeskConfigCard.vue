<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'

const props = defineProps<{ sessionId: string }>()
const emit = defineEmits<{ updated: [] }>()

const { api } = useWzap()
const toast = useToast()

const loading = ref(true)
const saving = ref(false)
const removing = ref(false)
const importing = ref(false)
const hasConfig = ref(false)
const editing = ref(false)
const confirmImportOpen = ref(false)
const confirmImportModal = useTemplateRef('confirmImportModal')

interface ElodeskConfig {
  url: string
  inboxIdentifier: string
  hasApiToken: boolean
  hasHmacToken: boolean
  hasUserAccessToken: boolean
  accountId: number
  signMsg: boolean
  signDelimiter: string
  reopenConv: boolean
  mergeBrContacts: boolean
  ignoreGroups: boolean
  webhookUrl: string
}

const config = ref<ElodeskConfig | null>(null)

const isNewConfig = computed(() => !hasConfig.value)

const schema = z.object({
  url: z.string().url('URL inválida'),
  inboxName: z.string().optional(),
  userAccessToken: z.string().optional(),
  inboxIdentifier: z.string().optional(),
  accountId: z.number().int().positive().optional(),
  signMsg: z.boolean(),
  signDelimiter: z.string().optional(),
  reopenConv: z.boolean(),
  mergeBrContacts: z.boolean(),
  ignoreGroups: z.boolean()
})

type Schema = z.output<typeof schema>

const state = reactive<Partial<Schema>>({
  url: '',
  inboxName: '',
  userAccessToken: '',
  inboxIdentifier: '',
  accountId: 1,
  signMsg: false,
  signDelimiter: '',
  reopenConv: true,
  mergeBrContacts: true,
  ignoreGroups: true
})

async function fetchConfig() {
  loading.value = true
  try {
    const res: { data: ElodeskConfig | null } = await api(`/sessions/${props.sessionId}/integrations/elodesk`)
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
    const body: Record<string, unknown> = { ...event.data }
    if (body.userAccessToken === '') delete body.userAccessToken
    if (body.inboxIdentifier === '') delete body.inboxIdentifier
    if (body.inboxName === '') delete body.inboxName

    await api(`/sessions/${props.sessionId}/integrations/elodesk`, {
      method: 'PUT',
      body
    })
    toast.add({ title: 'Elodesk configurado', color: 'success' })
    editing.value = false
    await fetchConfig()
    emit('updated')
  } catch {
    toast.add({ title: 'Falha ao salvar configuração', color: 'error' })
  }
  saving.value = false
}

async function removeConfig() {
  removing.value = true
  try {
    await api(`/sessions/${props.sessionId}/integrations/elodesk`, { method: 'DELETE' })
    toast.add({ title: 'Integração removida', color: 'success' })
    config.value = null
    hasConfig.value = false
    emit('updated')
  } catch {
    toast.add({ title: 'Falha ao remover integração', color: 'error' })
  }
  removing.value = false
}

function startEditing() {
  state.url = config.value?.url || ''
  state.inboxName = ''
  state.userAccessToken = ''
  state.inboxIdentifier = config.value?.inboxIdentifier || ''
  state.accountId = config.value?.accountId || 1
  state.signMsg = config.value?.signMsg ?? false
  state.signDelimiter = config.value?.signDelimiter || ''
  state.reopenConv = config.value?.reopenConv ?? true
  state.mergeBrContacts = config.value?.mergeBrContacts ?? true
  state.ignoreGroups = config.value?.ignoreGroups ?? true
  editing.value = true
}

async function importHistory() {
  importing.value = true
  try {
    await api(`/sessions/${props.sessionId}/integrations/elodesk/import`, {
      method: 'POST',
      body: { period: '7d' }
    })
    toast.add({ title: 'Importação iniciada', description: 'O histórico está sendo importado em background.', color: 'success' })
  } catch {
    toast.add({ title: 'Falha ao iniciar importação', color: 'error' })
  }
  importing.value = false
  confirmImportModal.value?.done()
}

watch(() => props.sessionId, fetchConfig, { immediate: true })
</script>

<template>
  <UPageCard variant="subtle" class="mt-6">
    <div v-if="loading" class="flex items-center justify-center py-8">
      <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
    </div>

    <!-- Configured state -->
    <template v-else-if="hasConfig && config && !editing">
      <UAlert
        v-if="config.inboxIdentifier && !config.hasApiToken"
        title="Inbox sem API token"
        description="Edite e cole um Access Token para reprovisionar."
        color="warning"
        variant="soft"
        icon="i-lucide-triangle-alert"
        class="mb-4"
      />

      <dl class="grid grid-cols-1 gap-y-3 text-sm sm:grid-cols-3">
        <dt class="text-muted">
          URL
        </dt>
        <dd class="font-medium text-highlighted truncate sm:col-span-2">
          {{ config.url }}
        </dd>

        <template v-if="config.inboxIdentifier">
          <dt class="text-muted">
            Inbox ID
          </dt>
          <dd class="font-mono text-xs text-highlighted truncate sm:col-span-2" :title="config.inboxIdentifier">
            {{ config.inboxIdentifier }}
          </dd>
        </template>

        <dt class="text-muted">
          Tokens
        </dt>
        <dd class="flex flex-wrap gap-1.5 sm:col-span-2">
          <UBadge :color="config.hasApiToken ? 'success' : 'error'" variant="subtle" size="xs">
            API {{ config.hasApiToken ? '✓' : '✕' }}
          </UBadge>
          <UBadge :color="config.hasHmacToken ? 'success' : 'neutral'" variant="subtle" size="xs">
            HMAC {{ config.hasHmacToken ? '✓' : '✕' }}
          </UBadge>
          <UBadge :color="config.hasUserAccessToken ? 'success' : 'neutral'" variant="subtle" size="xs">
            Access {{ config.hasUserAccessToken ? '✓' : '✕' }}
          </UBadge>
        </dd>

        <USeparator class="sm:col-span-3 my-1" />

        <dt class="text-muted">
          Webhook
        </dt>
        <dd class="font-mono text-xs text-highlighted truncate sm:col-span-2" :title="config.webhookUrl">
          {{ config.webhookUrl }}
        </dd>
      </dl>

      <div class="mt-5 flex flex-wrap items-center gap-2">
        <UButton
          label="Editar"
          icon="i-lucide-pencil"
          size="sm"
          color="primary"
          @click="startEditing"
        />
        <UButton
          label="Importar histórico"
          icon="i-lucide-download"
          size="sm"
          color="neutral"
          variant="outline"
          :loading="importing"
          @click="confirmImportOpen = true"
        />
        <UButton
          label="Remover"
          icon="i-lucide-trash"
          size="sm"
          color="error"
          variant="soft"
          :loading="removing"
          class="ms-auto"
          @click="removeConfig"
        />
      </div>

      <SessionsConfirmModal
        ref="confirmImportModal"
        v-model:open="confirmImportOpen"
        title="Importar histórico"
        description="Importa todas as conversas do Elodesk. Pode levar alguns minutos dependendo do volume."
        confirm-label="Importar"
        confirm-color="primary"
        icon="i-lucide-download"
        @confirm="importHistory"
      />
    </template>

    <!-- Configuration form -->
    <template v-else>
      <UForm
        :schema="schema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <UFormField label="Nome da inbox" name="inboxName">
          <UInput v-model="state.inboxName" placeholder="wzap" class="w-full" />
        </UFormField>

        <UFormField label="URL do backend" name="url" required>
          <UInput v-model="state.url" placeholder="http://host.docker.internal:3001" class="w-full" />
        </UFormField>

        <UFormField label="API Token do Elodesk" name="userAccessToken" :required="isNewConfig">
          <UInput
            v-model="state.userAccessToken"
            type="password"
            placeholder="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
            class="w-full"
          />
        </UFormField>

        <UFormField label="Inbox ID" name="inboxIdentifier">
          <UInput
            v-model="state.inboxIdentifier"
            placeholder="xxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxxx"
            class="w-full"
          />
        </UFormField>

        <UFormField label="Account ID" name="accountId">
          <UInput
            v-model.number="state.accountId"
            type="number"
            placeholder="1"
            class="w-full max-w-xs"
          />
        </UFormField>

        <USeparator />

        <div class="grid grid-cols-1 gap-x-6 gap-y-2 sm:grid-cols-2">
          <UFormField label="Assinar mensagens" name="signMsg">
            <USwitch v-model="state.signMsg" />
          </UFormField>
          <UFormField label="Reabrir conversas" name="reopenConv">
            <USwitch v-model="state.reopenConv" />
          </UFormField>
          <UFormField label="Mesclar contatos BR" name="mergeBrContacts">
            <USwitch v-model="state.mergeBrContacts" />
          </UFormField>
          <UFormField label="Ignorar grupos" name="ignoreGroups">
            <USwitch v-model="state.ignoreGroups" />
          </UFormField>
        </div>

        <UFormField v-if="state.signMsg" label="Delimitador" name="signDelimiter">
          <UInput v-model="state.signDelimiter" placeholder="-" class="w-full max-w-xs" />
        </UFormField>

        <div class="flex justify-end gap-2 pt-1">
          <UButton
            v-if="hasConfig"
            label="Cancelar"
            color="neutral"
            variant="subtle"
            @click="editing = false"
          />
          <UButton
            :label="isNewConfig ? 'Configurar' : 'Salvar'"
            color="primary"
            type="submit"
            :loading="saving"
          />
        </div>
      </UForm>
    </template>
  </UPageCard>
</template>
