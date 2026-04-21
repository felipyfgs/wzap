<script setup lang="ts">
import * as z from 'zod'
import type { FormSubmitEvent } from '@nuxt/ui'
import { MESSAGE_TYPE_OPTIONS, useMessageSender } from '~/composables/useMessageSender'
import type { MessageType, ButtonItem, ListSection } from '~/composables/useMessageSender'

const props = defineProps<{ sessionId: string }>()

const { sendMessage } = useMessageSender()
const toast = useToast()
const open = ref(false)
const loading = ref(false)
const msgType = ref<MessageType>('text')

const textSchema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number or JID'),
  body: z.string().min(1, 'Message cannot be empty')
})

const contactSchema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number or JID'),
  name: z.string().min(1, 'Contact name is required'),
  vcard: z.string().min(1, 'vCard data is required')
})

const locationSchema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number or JID'),
  latitude: z.coerce.number().min(-90).max(90, 'Latitude must be between -90 and 90'),
  longitude: z.coerce.number().min(-180).max(180, 'Longitude must be between -180 and 180'),
  name: z.string().optional(),
  address: z.string().optional()
})

const linkSchema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number or JID'),
  url: z.string().min(1, 'URL is required'),
  title: z.string().optional(),
  description: z.string().optional()
})

const mediaSchema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number or JID'),
  mimeType: z.string().min(1, 'MIME type is required'),
  base64: z.string().optional(),
  url: z.string().optional(),
  caption: z.string().optional(),
  fileName: z.string().optional()
}).refine(data => data.base64 || data.url, {
  message: 'Provide a file or a URL',
  path: ['base64']
})

const stickerSchema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number or JID'),
  mimeType: z.string().min(1, 'MIME type is required'),
  base64: z.string().min(1, 'Sticker file is required')
})

const pollSchema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number or JID'),
  name: z.string().min(1, 'Poll question is required'),
  options: z.array(z.string().min(1, 'Option cannot be empty')).min(2, 'At least 2 options required'),
  selectableCount: z.coerce.number().int().min(1).optional()
})

const buttonSchema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number or JID'),
  body: z.string().min(1, 'Body text is required'),
  footer: z.string().optional(),
  buttons: z.array(z.object({
    id: z.string().min(1, 'Button ID is required'),
    text: z.string().min(1, 'Button text is required')
  })).min(1, 'At least 1 button required')
})

const listSchema = z.object({
  phone: z.string().min(6, 'Enter a valid phone number or JID'),
  title: z.string().min(1, 'Title is required'),
  body: z.string().min(1, 'Body is required'),
  footer: z.string().optional(),
  buttonText: z.string().min(1, 'Button text is required'),
  sections: z.array(z.object({
    title: z.string().min(1, 'Section title is required'),
    rows: z.array(z.object({
      id: z.string().min(1, 'Row ID is required'),
      title: z.string().min(1, 'Row title is required'),
      description: z.string().optional()
    })).min(1, 'At least 1 row per section')
  })).min(1, 'At least 1 section required')
})

const schemas: Record<MessageType, z.ZodTypeAny> = {
  text: textSchema,
  image: mediaSchema,
  video: mediaSchema,
  document: mediaSchema,
  audio: mediaSchema,
  contact: contactSchema,
  location: locationSchema,
  poll: pollSchema,
  sticker: stickerSchema,
  link: linkSchema,
  button: buttonSchema,
  list: listSchema
}

const activeSchema = computed(() => schemas[msgType.value])

// eslint-disable-next-line @typescript-eslint/no-explicit-any
type FormState = Record<string, any>
const state = reactive<FormState>({ phone: '', body: '' })

const mediaFileName = ref('')
const mediaBase64 = ref('')
const mediaMimeType = ref('')

const pollOptions = ref(['', ''])

const buttons = ref<ButtonItem[]>([{ id: '1', text: '' }])

const sections = ref<ListSection[]>([{ title: '', rows: [{ id: '1', title: '', description: '' }] }])

function resetForm(preservePhone = false) {
  const saved = preservePhone ? state.phone : ''
  for (const key of Object.keys(state) as (keyof typeof state)[]) {
    state[key] = undefined
  }
  state.phone = saved
  mediaFileName.value = ''
  mediaBase64.value = ''
  mediaMimeType.value = ''
  pollOptions.value = ['', '']
  buttons.value = [{ id: '1', text: '' }]
  sections.value = [{ title: '', rows: [{ id: '1', title: '', description: '' }] }]
}

watch(open, (val) => {
  if (val) {
    resetForm(false)
  }
})

function handleFileSelect(event: Event) {
  const input = event.target as HTMLInputElement
  const file = input.files?.[0]
  if (!file) return

  mediaFileName.value = file.name
  mediaMimeType.value = file.type

  const reader = new FileReader()
  reader.onload = () => {
    const result = reader.result as string
    mediaBase64.value = result.split(',')[1] || ''
    state.mimeType = mediaMimeType.value
    state.base64 = mediaBase64.value
    state.fileName = mediaFileName.value
  }
  reader.readAsDataURL(file)
}

function addPollOption() {
  pollOptions.value.push('')
}

function removePollOption(index: number) {
  if (pollOptions.value.length > 2) {
    pollOptions.value.splice(index, 1)
  }
}

function addButton() {
  const nextId = String(buttons.value.length + 1)
  buttons.value.push({ id: nextId, text: '' })
}

function removeButton(index: number) {
  if (buttons.value.length > 1) {
    buttons.value.splice(index, 1)
  }
}

function addSection() {
  sections.value.push({ title: '', rows: [{ id: '1', title: '', description: '' }] })
}

function removeSection(index: number) {
  if (sections.value.length > 1) {
    sections.value.splice(index, 1)
  }
}

function addRow(sectionIndex: number) {
  const section = sections.value[sectionIndex]
  if (!section) return
  const nextId = String(section.rows.length + 1)
  section.rows.push({ id: nextId, title: '', description: '' })
}

function removeRow(sectionIndex: number, rowIndex: number) {
  const section = sections.value[sectionIndex]
  if (!section || section.rows.length <= 1) return
  section.rows.splice(rowIndex, 1)
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
function buildPayload(type: MessageType, data: Record<string, any>) {
  const phone = data.phone?.includes('@')
    ? data.phone
    : (data.phone || '').replace(/\D/g, '')

  const base = { phone } as Record<string, unknown>

  switch (type) {
    case 'text':
      return { ...base, body: data.body }
    case 'image':
    case 'video':
    case 'document':
    case 'audio':
      return {
        ...base,
        mimeType: state.mimeType || mediaMimeType.value || data.mimeType,
        base64: mediaBase64.value || data.base64 || undefined,
        url: (!mediaBase64.value && data.url) ? data.url : undefined,
        caption: data.caption || undefined,
        fileName: mediaFileName.value || data.fileName || undefined
      }
    case 'sticker':
      return {
        ...base,
        mimeType: mediaMimeType.value || data.mimeType,
        base64: mediaBase64.value || data.base64
      }
    case 'contact':
      return { ...base, name: data.name, vcard: data.vcard }
    case 'location':
      return {
        ...base,
        latitude: Number(data.latitude),
        longitude: Number(data.longitude),
        name: data.name || undefined,
        address: data.address || undefined
      }
    case 'poll':
      return {
        ...base,
        name: data.name,
        options: pollOptions.value.filter(o => o.trim() !== ''),
        selectableCount: data.selectableCount ? Number(data.selectableCount) : undefined
      }
    case 'link':
      return {
        ...base,
        url: data.url,
        title: data.title || undefined,
        description: data.description || undefined
      }
    case 'button':
      return {
        ...base,
        body: data.body,
        footer: data.footer || undefined,
        buttons: buttons.value.filter(b => b.text.trim() !== '')
      }
    case 'list':
      return {
        ...base,
        title: data.title,
        body: data.body,
        footer: data.footer || undefined,
        buttonText: data.buttonText,
        sections: sections.value.map(s => ({
          title: s.title,
          rows: s.rows.filter(r => r.title.trim() !== '')
        }))
      }
    default:
      return base
  }
}

// eslint-disable-next-line @typescript-eslint/no-explicit-any
async function onSubmit(_event: FormSubmitEvent<any>) {
  loading.value = true
  try {
    const payload = buildPayload(msgType.value, state)
    // eslint-disable-next-line @typescript-eslint/no-explicit-any
    await sendMessage(props.sessionId, msgType.value, payload as any)
    toast.add({ title: 'Message sent', color: 'success' })
    resetForm(false)
    state.phone = ''
    open.value = false
  } catch {
    toast.add({ title: 'Failed to send message', color: 'error' })
  }
  loading.value = false
}

function getModalDescription(): string {
  const descriptions: Record<MessageType, string> = {
    text: 'Send a text message',
    image: 'Send an image',
    video: 'Send a video',
    document: 'Send a document',
    audio: 'Send an audio',
    contact: 'Send a contact',
    location: 'Send a location',
    poll: 'Send a poll',
    sticker: 'Send a sticker',
    link: 'Send a link preview',
    button: 'Send buttons message',
    list: 'Send a list message'
  }
  return descriptions[msgType.value]
}

function show() {
  open.value = true
}
defineExpose({ show })
</script>

<template>
  <UModal v-model:open="open" :title="'Send Message'" :description="getModalDescription()">
    <template #body>
      <UForm
        :schema="activeSchema"
        :state="state"
        class="space-y-4"
        @submit="onSubmit"
      >
        <UFormField label="Type" name="type">
          <USelect
            v-model="msgType"
            :items="MESSAGE_TYPE_OPTIONS"
            value-key="value"
            class="w-full"
          />
        </UFormField>

        <USeparator />

        <UFormField
          label="Phone / JID"
          name="phone"
          description="E.g. 5511999999999 or full JID"
        >
          <UInput v-model="state.phone" placeholder="5511999999999" class="w-full" />
        </UFormField>
        <!-- Text -->
        <SessionsMessageFormsTextForm v-if="msgType === 'text'" :state="state" />

        <!-- Contact -->
        <SessionsMessageFormsContactForm v-if="msgType === 'contact'" :state="state" />

        <!-- Location -->
        <SessionsMessageFormsLocationForm v-if="msgType === 'location'" :state="state" />

        <!-- Link -->
        <SessionsMessageFormsLinkForm v-if="msgType === 'link'" :state="state" />

        <!-- Media: image, video, document, audio -->
        <SessionsMessageFormsMediaForm
          v-if="['image', 'video', 'document', 'audio'].includes(msgType)"
          :state="state"
          :msg-type="msgType"
          :media-file-name="mediaFileName"
          :media-base64="mediaBase64"
          :media-mime-type="mediaMimeType"
          @file-select="handleFileSelect"
        />

        <!-- Sticker -->
        <SessionsMessageFormsStickerForm
          v-if="msgType === 'sticker'"
          :media-file-name="mediaFileName"
          @file-select="handleFileSelect"
        />

        <!-- Poll -->
        <SessionsMessageFormsPollForm
          v-if="msgType === 'poll'"
          :state="state"
          :poll-options="pollOptions"
          @add-option="addPollOption"
          @remove-option="removePollOption"
        />

        <!-- Button -->
        <SessionsMessageFormsButtonForm
          v-if="msgType === 'button'"
          :state="state"
          :buttons="buttons"
          @add-button="addButton"
          @remove-button="removeButton"
        />

        <!-- List -->
        <SessionsMessageFormsListForm
          v-if="msgType === 'list'"
          :state="state"
          :sections="sections"
          @add-section="addSection"
          @remove-section="removeSection"
          @add-row="addRow"
          @remove-row="removeRow"
        />

        <div class="flex justify-end gap-2">
          <UButton
            label="Cancel"
            color="neutral"
            variant="subtle"
            @click="open = false"
          />
          <UButton
            label="Send"
            color="primary"
            type="submit"
            :loading="loading"
            :disabled="loading"
            icon="i-lucide-send"
          />
        </div>
      </UForm>
    </template>
  </UModal>
</template>
