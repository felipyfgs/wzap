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

const STORY_DURATION = 6000

const carouselRef = useTemplateRef<{ emblaApi: { scrollTo: (i: number) => void } | null }>('carousel')
const currentIndex = ref(props.initialIndex ?? 0)
const progress = ref(0)
const paused = ref(false)
let rafId: number | null = null
let startTime = 0
let elapsed = 0

const currentStatus = computed(() => props.statuses[currentIndex.value])

function mediaProxyUrl(url: string): string {
  if (!url) return ''
  try {
    const parsed = new URL(url)
    if (!parsed.hostname.includes('.') || parsed.hostname === 'localhost' || parsed.hostname === '127.0.0.1') {
      return `/api/media-proxy?url=${encodeURIComponent(url)}`
    }
  } catch {}
  return url
}

function formatTime(ts?: string) {
  if (!ts) return ''
  return new Date(ts).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}

function tick(now: number) {
  if (paused.value) {
    startTime = now - elapsed
    rafId = requestAnimationFrame(tick)
    return
  }
  elapsed = now - startTime
  progress.value = Math.min((elapsed / STORY_DURATION) * 100, 100)
  if (elapsed >= STORY_DURATION) {
    goNext()
    return
  }
  rafId = requestAnimationFrame(tick)
}

function startTimer() {
  if (rafId) cancelAnimationFrame(rafId)
  elapsed = 0
  progress.value = 0
  startTime = performance.now()
  rafId = requestAnimationFrame(tick)
}

function stopTimer() {
  if (rafId) {
    cancelAnimationFrame(rafId)
    rafId = null
  }
}

function goTo(index: number) {
  if (index < 0) return
  if (index >= props.statuses.length) { emit('close'); return }
  currentIndex.value = index
  carouselRef.value?.emblaApi?.scrollTo(index)
  startTimer()
}

function goNext() { goTo(currentIndex.value + 1) }
function goPrev() { goTo(currentIndex.value - 1) }

function handleKeydown(e: KeyboardEvent) {
  if (e.key === 'Escape') emit('close')
  if (e.key === 'ArrowRight') goNext()
  if (e.key === 'ArrowLeft') goPrev()
}

onMounted(() => {
  window.addEventListener('keydown', handleKeydown)
  startTimer()
})
onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  stopTimer()
})
</script>

<template>
  <UModal
    :open="true"
    :overlay="true"
    :transition="true"
    :ui="{ content: 'bg-black w-full max-w-sm mx-auto rounded-2xl overflow-hidden shadow-2xl' }"
    @update:open="$emit('close')"
  >

    <template #content>
      <div
        class="relative w-full bg-black select-none"
        style="aspect-ratio: 9/16; max-height: 90vh;"
        @mousedown="paused = true"
        @mouseup="paused = false"
        @touchstart.passive="paused = true"
        @touchend.passive="paused = false"
      >
        <!-- Tap zones -->
        <div class="absolute inset-y-0 left-0 w-2/5 z-20 cursor-pointer" @click.stop="goPrev" />
        <div class="absolute inset-y-0 right-0 w-2/5 z-20 cursor-pointer" @click.stop="goNext" />

        <!-- Media slide -->
        <Transition name="story-fade" mode="out-in">
          <div :key="currentIndex" class="absolute inset-0 flex items-center justify-center overflow-hidden">
            <!-- Image -->
            <img
              v-if="currentStatus?.mediaUrl && currentStatus.statusType === 'image'"
              :src="mediaProxyUrl(currentStatus.mediaUrl)"
              :alt="currentStatus.body || 'Status'"
              class="w-full h-full object-cover"
              draggable="false"
            >
            <!-- Video -->
            <video
              v-else-if="currentStatus?.mediaUrl && currentStatus.statusType === 'video'"
              :src="mediaProxyUrl(currentStatus.mediaUrl)"
              class="w-full h-full object-cover"
              autoplay
              muted
              playsinline
              @ended="goNext"
            />
            <!-- Text -->
            <div
              v-else-if="currentStatus?.statusType === 'text'"
              class="w-full h-full flex items-center justify-center p-8 text-white text-xl font-semibold text-center whitespace-pre-wrap"
              style="background: linear-gradient(135deg,#1a1a3e,#0f2027);"
            >
              {{ currentStatus.body }}
            </div>
            <!-- Fallback -->
            <div v-else class="flex flex-col items-center gap-3 text-white/40">
              <UIcon name="i-lucide-image-off" class="text-5xl" />
              <span class="text-sm">Mídia não disponível</span>
            </div>
          </div>
        </Transition>

        <!-- Top gradient overlay -->
        <div class="absolute inset-x-0 top-0 z-10 pointer-events-none" style="background:linear-gradient(to bottom,rgba(0,0,0,.6) 0%,transparent 100%);height:96px" />

        <!-- Progress bars -->
        <div class="absolute top-0 inset-x-0 z-10 flex gap-1 px-3 pt-3">
          <div
            v-for="(_, i) in statuses"
            :key="i"
            class="h-0.5 flex-1 rounded-full bg-white/30 overflow-hidden"
          >
            <div
              class="h-full bg-white rounded-full"
              :style="{
                width: i < currentIndex ? '100%' : i === currentIndex ? `${progress}%` : '0%',
                transition: i === currentIndex ? 'none' : undefined
              }"
            />
          </div>
        </div>

        <!-- Header: avatar + name + time + close -->
        <div class="absolute top-5 inset-x-0 z-10 flex items-center justify-between px-3">
          <div class="flex items-center gap-2">
            <div class="w-8 h-8 rounded-full bg-white/20 border border-white/30 flex items-center justify-center text-xs font-bold text-white">
              {{ (senderName || 'U').charAt(0).toUpperCase() }}
            </div>
            <div class="leading-tight">
              <p class="text-white text-sm font-semibold leading-none">{{ senderName || 'Unknown' }}</p>
              <p class="text-white/60 text-xs mt-0.5">{{ formatTime(currentStatus?.timestamp) }}</p>
            </div>
          </div>
          <UButton
            icon="i-lucide-x"
            color="neutral"
            variant="ghost"
            size="xs"
            class="text-white hover:bg-white/20"
            @click="$emit('close')"
          />
        </div>

        <!-- Bottom gradient + caption -->
        <div
          v-if="currentStatus?.body && currentStatus.statusType !== 'text'"
          class="absolute inset-x-0 bottom-0 z-10 pointer-events-none px-4 pb-5 pt-12"
          style="background:linear-gradient(to top,rgba(0,0,0,.65) 0%,transparent 100%)"
        >
          <p class="text-white text-sm leading-snug">{{ currentStatus.body }}</p>
        </div>
      </div>
    </template>
  </UModal>
</template>

<style scoped>
.story-fade-enter-active,
.story-fade-leave-active {
  transition: opacity 0.25s ease;
}
.story-fade-enter-from,
.story-fade-leave-to {
  opacity: 0;
}
</style>