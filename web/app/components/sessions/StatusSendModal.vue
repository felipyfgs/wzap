<script setup lang="ts">
const props = defineProps<{
  sessionId: string
}>()

const emit = defineEmits<{
  sent: []
}>()

const { sendStatusText, sendStatusImage, sendStatusVideo } = useStatus()
const toast = useToast()

const open = ref(false)
const loading = ref(false)
const mode = ref<'text' | 'image' | 'video'>('text')
const textBody = ref('')
const backgroundColor = ref('')
const font = ref<number | undefined>(undefined)
const caption = ref('')
const mediaFileName = ref('')
const mediaBase64 = ref('')
const mediaMimeType = ref('')

const bgColors = [
  { label: 'Default', value: '' },
  { label: 'Green', value: '#25D366' },
  { label: 'Blue', value: '#34B7F1' },
  { label: 'Purple', value: '#8E44AD' },
  { label: 'Red', value: '#E74C3C' },
  { label: 'Orange', value: '#F39C12' },
  { label: 'Dark', value: '#1B2631' }
]

const fonts = [
  { label: 'Sans Serif', value: 0 },
  { label: 'Serif', value: 1 },
  { label: 'Norican', value: 2 },
  { label: 'Bryndan', value: 3 },
  { label: 'Bebas Neue', value: 4 }
]

function resetForm() {
  textBody.value = ''
  backgroundColor.value = ''
  font.value = undefined
  caption.value = ''
  mediaFileName.value = ''
  mediaBase64.value = ''
  mediaMimeType.value = ''
}

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
  }
  reader.readAsDataURL(file)
}

async function onSubmit() {
  loading.value = true
  try {
    if (mode.value === 'text') {
      await sendStatusText(props.sessionId, textBody.value, backgroundColor.value || undefined, font.value)
    } else if (mode.value === 'image') {
      await sendStatusImage(props.sessionId, {
        mimeType: mediaMimeType.value,
        base64: mediaBase64.value || undefined,
        caption: caption.value || undefined
      })
    } else if (mode.value === 'video') {
      await sendStatusVideo(props.sessionId, {
        mimeType: mediaMimeType.value,
        base64: mediaBase64.value || undefined,
        caption: caption.value || undefined
      })
    }
    toast.add({ title: 'Status sent', color: 'success' })
    resetForm()
    open.value = false
    emit('sent')
  } catch {
    toast.add({ title: 'Failed to send status', color: 'error' })
  } finally {
    loading.value = false
  }
}

watch(open, (val) => {
  if (val) resetForm()
})
</script>

<template>
  <UModal v-model:open="open" title="Post Status">
    <UButton label="Post Status" icon="i-lucide-circle-dot" color="primary" />

    <template #body>
      <div class="space-y-4">
        <UFormField label="Type">
          <USelectMenu
            v-model="mode"
            :items="[
              { label: 'Text', value: 'text' },
              { label: 'Image', value: 'image' },
              { label: 'Video', value: 'video' }
            ]"
            value-key="value"
            class="w-full"
          />
        </UFormField>

        <USeparator />

        <!-- Text form -->
        <template v-if="mode === 'text'">
          <UFormField label="Status Text" name="body">
            <UTextarea
              v-model="textBody"
              :rows="4"
              autoresize
              placeholder="What's on your mind?"
              class="w-full"
            />
          </UFormField>
          <div class="grid grid-cols-2 gap-3">
            <UFormField label="Background Color" name="backgroundColor">
              <USelectMenu
                v-model="backgroundColor"
                :items="bgColors"
                value-key="value"
                placeholder="Select color"
                class="w-full"
              />
            </UFormField>
            <UFormField label="Font" name="font">
              <USelectMenu
                v-model="font"
                :items="fonts"
                value-key="value"
                placeholder="Select font"
                class="w-full"
              />
            </UFormField>
          </div>
        </template>

        <!-- Image form -->
        <template v-if="mode === 'image'">
          <UFormField label="Image File">
            <input
              type="file"
              accept="image/*"
              class="block w-full text-sm text-muted
                file:mr-2 file:py-1.5 file:px-3 file:rounded-md
                file:border-0 file:text-sm file:font-medium
                file:bg-elevated file:text-highlighted
                hover:file:bg-accentened file:cursor-pointer"
              @change="handleFileSelect"
            >
            <template v-if="mediaFileName" #hint>
              <span class="text-xs text-muted">{{ mediaFileName }}</span>
            </template>
          </UFormField>
          <USeparator label="or" />
          <UFormField label="URL" name="url">
            <UInput v-model="caption" placeholder="Image URL (or upload above)" class="w-full" />
          </UFormField>
          <UFormField label="Caption" name="caption">
            <UTextarea v-model="caption" :rows="2" autoresize placeholder="Caption (optional)" class="w-full" />
          </UFormField>
        </template>

        <!-- Video form -->
        <template v-if="mode === 'video'">
          <UFormField label="Video File">
            <input
              type="file"
              accept="video/*"
              class="block w-full text-sm text-muted
                file:mr-2 file:py-1.5 file:px-3 file:rounded-md
                file:border-0 file:text-sm file:font-medium
                file:bg-elevated file:text-highlighted
                hover:file:bg-accentened file:cursor-pointer"
              @change="handleFileSelect"
            >
            <template v-if="mediaFileName" #hint>
              <span class="text-xs text-muted">{{ mediaFileName }}</span>
            </template>
          </UFormField>
          <USeparator label="or" />
          <UFormField label="URL" name="url">
            <UInput v-model="caption" placeholder="Video URL (or upload above)" class="w-full" />
          </UFormField>
          <UFormField label="Caption" name="caption">
            <UTextarea v-model="caption" :rows="2" autoresize placeholder="Caption (optional)" class="w-full" />
          </UFormField>
        </template>

        <div class="flex justify-end gap-2">
          <UButton label="Cancel" color="neutral" variant="subtle" @click="open = false" />
          <UButton
            label="Send"
            color="primary"
            icon="i-lucide-send"
            :loading="loading"
            :disabled="loading"
            @click="onSubmit"
          />
        </div>
      </div>
    </template>
  </UModal>
</template>