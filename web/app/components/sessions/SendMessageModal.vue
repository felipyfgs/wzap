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
  body: z.string().min(1, 'Body text is required'),
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

const mediaFileInput = ref<HTMLInputElement | null>(null)
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
</script>

<template>
  <UModal v-model:open="open" :title="'Send Message'" :description="getModalDescription()">
    <UButton label="Send Message" icon="i-lucide-send" color="primary" />

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

        <UFormField label="Phone / JID" name="phone" description="E.g. 5511999999999 or full JID">
          <UInput v-model="state.phone" placeholder="5511999999999" class="w-full" />
        </UFormField>

        <!-- Text -->
        <template v-if="msgType === 'text'">
          <UFormField label="Message" name="body">
            <UTextarea
              v-model="state.body"
              :rows="4"
              autoresize
              placeholder="Type your message…"
              class="w-full"
            />
          </UFormField>
        </template>

        <!-- Contact -->
        <template v-if="msgType === 'contact'">
          <UFormField label="Contact Name" name="name">
            <UInput v-model="state.name" placeholder="John Doe" class="w-full" />
          </UFormField>
          <UFormField label="vCard" name="vcard">
            <UTextarea
              v-model="state.vcard"
              :rows="6"
              autoresize
              placeholder="BEGIN:VCARD&#10;VERSION:3.0&#10;FN:John Doe&#10;TEL:+5511999999999&#10;END:VCARD"
              class="w-full"
            />
          </UFormField>
        </template>

        <!-- Location -->
        <template v-if="msgType === 'location'">
          <div class="grid grid-cols-2 gap-3">
            <UFormField label="Latitude" name="latitude">
              <UInput
                v-model="state.latitude"
                type="number"
                step="any"
                placeholder="-23.5505"
                class="w-full"
              />
            </UFormField>
            <UFormField label="Longitude" name="longitude">
              <UInput
                v-model="state.longitude"
                type="number"
                step="any"
                placeholder="-46.6333"
                class="w-full"
              />
            </UFormField>
          </div>
          <UFormField label="Name" name="name">
            <UInput v-model="state.name" placeholder="Location name" class="w-full" />
          </UFormField>
          <UFormField label="Address" name="address">
            <UInput v-model="state.address" placeholder="Street, City…" class="w-full" />
          </UFormField>
        </template>

        <!-- Link -->
        <template v-if="msgType === 'link'">
          <UFormField label="URL" name="url">
            <UInput v-model="state.url" placeholder="https://example.com" class="w-full" />
          </UFormField>
          <UFormField label="Title" name="title">
            <UInput v-model="state.title" placeholder="Link title" class="w-full" />
          </UFormField>
          <UFormField label="Description" name="description">
            <UInput v-model="state.description" placeholder="Link description" class="w-full" />
          </UFormField>
        </template>

        <!-- Media: image, video, document, audio -->
        <template v-if="['image', 'video', 'document', 'audio'].includes(msgType)">
          <UFormField label="File">
            <div class="flex items-center gap-2 w-full">
              <input
                ref="mediaFileInput"
                type="file"
                class="block w-full text-sm text-muted
                  file:mr-2 file:py-1.5 file:px-3 file:rounded-md
                  file:border-0 file:text-sm file:font-medium
                  file:bg-elevated file:text-highlighted
                  hover:file:bg-accentened file:cursor-pointer"
                :accept="msgType === 'image' ? 'image/*' : msgType === 'video' ? 'video/*' : msgType === 'audio' ? 'audio/*' : '*/*'"
                @change="handleFileSelect"
              >
            </div>
            <template v-if="mediaFileName" #hint>
              <span class="text-xs text-muted">{{ mediaFileName }}</span>
            </template>
          </UFormField>
          <USeparator label="or" />
          <UFormField label="URL" name="url">
            <UInput v-model="state.url" placeholder="https://example.com/file.jpg" class="w-full" />
          </UFormField>
          <UFormField label="Caption" name="caption">
            <UTextarea
              v-model="state.caption"
              :rows="2"
              autoresize
              placeholder="Caption (optional)"
              class="w-full"
            />
          </UFormField>
          <template v-if="msgType === 'document'">
            <UFormField label="File Name" name="fileName">
              <UInput v-model="state.fileName" placeholder="document.pdf" class="w-full" />
            </UFormField>
          </template>
        </template>

        <!-- Sticker -->
        <template v-if="msgType === 'sticker'">
          <UFormField label="Sticker File">
            <input
              type="file"
              class="block w-full text-sm text-muted
                file:mr-2 file:py-1.5 file:px-3 file:rounded-md
                file:border-0 file:text-sm file:font-medium
                file:bg-elevated file:text-highlighted
                hover:file:bg-accentened file:cursor-pointer"
              accept="image/webp,image/png"
              @change="handleFileSelect"
            >
            <template v-if="mediaFileName" #hint>
              <span class="text-xs text-muted">{{ mediaFileName }}</span>
            </template>
          </UFormField>
        </template>

        <!-- Poll -->
        <template v-if="msgType === 'poll'">
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
                @click="removePollOption(i)"
              />
            </div>
            <UButton
              label="Add Option"
              icon="i-lucide-plus"
              color="neutral"
              variant="subtle"
              size="sm"
              @click="addPollOption"
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
        </template>

        <!-- Button -->
        <template v-if="msgType === 'button'">
          <UFormField label="Body" name="body">
            <UTextarea
              v-model="state.body"
              :rows="3"
              autoresize
              placeholder="Message body"
              class="w-full"
            />
          </UFormField>
          <UFormField label="Footer" name="footer">
            <UInput v-model="state.footer" placeholder="Footer text (optional)" class="w-full" />
          </UFormField>
          <div class="space-y-2">
            <p class="text-sm font-medium text-highlighted">
              Buttons
            </p>
            <div v-for="(btn, i) in buttons" :key="i" class="flex items-center gap-2">
              <UInput v-model="btn.id" placeholder="ID" class="w-24" />
              <UInput v-model="btn.text" placeholder="Button text" class="flex-1" />
              <UButton
                icon="i-lucide-x"
                color="neutral"
                variant="ghost"
                size="sm"
                :disabled="buttons.length <= 1"
                @click="removeButton(i)"
              />
            </div>
            <UButton
              label="Add Button"
              icon="i-lucide-plus"
              color="neutral"
              variant="subtle"
              size="sm"
              @click="addButton"
            />
          </div>
        </template>

        <!-- List -->
        <template v-if="msgType === 'list'">
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
                @click="addSection"
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
                  @click="removeSection(si)"
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
                  @click="removeRow(si, ri)"
                />
              </div>
              <div class="pl-4">
                <UButton
                  label="Add Row"
                  icon="i-lucide-plus"
                  color="neutral"
                  variant="ghost"
                  size="sm"
                  @click="addRow(si)"
                />
              </div>
            </div>
          </div>
        </template>

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
