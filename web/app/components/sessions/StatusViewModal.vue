<script setup lang="ts">
import type { Status } from '~/composables/useStatus'

const props = defineProps<{
  statuses: Status[]
  initialIndex?: number
  senderName?: string
}>()

const emit = defineEmits<{
  close: []
}>()

const currentIndex = ref(props.initialIndex ?? 0)
const currentStatus = computed(() => props.statuses[currentIndex.value])

function mediaProxyUrl(url: string): string {
  if (!url) return ''
  try {
    const parsed = new URL(url)
    if (parsed.hostname === 'localhost' || parsed.hostname === '127.0.0.1' || !parsed.hostname.includes('.')) {
      return `/api/media-proxy?url=${encodeURIComponent(url)}`
    }
  } catch {}
  return url
}

function next() {
  if (currentIndex.value < props.statuses.length - 1) {
    currentIndex.value++
  }
}

function prev() {
  if (currentIndex.value > 0) {
    currentIndex.value--
  }
}

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
  if (e.key === 'ArrowRight') next()
  if (e.key === 'ArrowLeft') prev()
}

onMounted(() => window.addEventListener('keydown', handleKeydown))
onUnmounted(() => window.removeEventListener('keydown', handleKeydown))
</script>

<template>
  <UModal
    :open="true"
    @update:open="$emit('close')"
  >
    <template #header>
      <div class="flex items-center justify-between w-full">
        <div class="flex items-center gap-2">
          <div class="w-8 h-8 rounded-full bg-muted flex items-center justify-center text-xs font-medium text-highlighted">
            {{ (senderName || 'U').charAt(0).toUpperCase() }}
          </div>
          <span class="text-sm font-medium text-highlighted">{{ senderName || 'Unknown' }}</span>
        </div>
        <UButton
          icon="i-lucide-x"
          color="neutral"
          variant="ghost"
          size="sm"
          @click="$emit('close')"
        />
      </div>
    </template>

    <template #body>
      <!-- Progress bars -->
      <div class="flex gap-1 mb-4">
        <div
          v-for="(_, i) in statuses"
          :key="i"
          class="h-1 flex-1 rounded-full"
          :class="i <= currentIndex ? 'bg-blue-500' : 'bg-elevated'"
        />
      </div>

      <!-- Navigation -->
      <div class="relative">
        <div
          v-if="currentIndex > 0"
          class="absolute left-0 top-0 bottom-0 w-1/3 cursor-pointer z-10"
          @click="prev"
        />
        <div
          v-if="currentIndex < statuses.length - 1"
          class="absolute right-0 top-0 bottom-0 w-1/3 cursor-pointer z-10"
          @click="next"
        />

        <!-- Content -->
        <div class="min-h-[300px] flex items-center justify-center rounded-lg bg-elevated p-6">
          <template v-if="currentStatus">
            <!-- Text status -->
            <div
              v-if="currentStatus.statusType === 'text'"
              class="w-full text-center"
            >
              <p class="text-lg text-highlighted whitespace-pre-wrap">
                {{ currentStatus.body }}
              </p>
            </div>

            <!-- Image status -->
            <img
              v-else-if="currentStatus.mediaUrl && currentStatus.statusType === 'image'"
              :src="mediaProxyUrl(currentStatus.mediaUrl)"
              :alt="currentStatus.body || 'Status image'"
              class="max-h-[60vh] max-w-full rounded-lg object-contain"
            >

            <!-- Video status -->
            <video
              v-else-if="currentStatus.mediaUrl && currentStatus.statusType === 'video'"
              :src="mediaProxyUrl(currentStatus.mediaUrl)"
              controls
              class="max-h-[60vh] max-w-full rounded-lg"
            />

            <!-- Fallback -->
            <p v-else class="text-muted">
              {{ currentStatus.statusType }} status
            </p>
          </template>
        </div>

        <!-- Caption -->
        <p
          v-if="currentStatus?.body && currentStatus.statusType !== 'text'"
          class="text-sm text-muted mt-2 text-center"
        >
          {{ currentStatus.body }}
        </p>
      </div>

      <!-- Navigation buttons -->
      <div class="flex justify-between mt-4">
        <UButton
          label="Previous"
          icon="i-lucide-chevron-left"
          color="neutral"
          variant="outline"
          :disabled="currentIndex === 0"
          @click="prev"
        />
        <UButton
          label="Next"
          icon="i-lucide-chevron-right"
          color="primary"
          :disabled="currentIndex >= statuses.length - 1"
          @click="next"
        />
      </div>

      <p class="text-xs text-muted text-center mt-2">
        {{ currentIndex + 1 }} / {{ statuses.length }}
      </p>
    </template>
  </UModal>
</template>