<script setup lang="ts">
import { useVirtualizer } from '@tanstack/vue-virtual'

const route = useRoute()
const { api } = useWzap()

const sessionId = computed(() => route.params.id as string)

interface MediaMessage {
  id: string
  chatJid: string
  msgType: string
  body: string
  timestamp: string
  mediaType?: string
}

interface MediaPage {
  items: MediaMessage[]
  nextCursor: string | null
  total: number
}

const items = ref<MediaMessage[]>([])
const totalCount = ref(0)
const loading = ref(false)
const loadingMore = ref(false)
const nextCursor = ref<string | null>(null)
const failedCursor = ref<string | null>(null)
const typeFilter = ref('all')
const searchQuery = ref('')
const searchDebounced = ref('')
const dateSince = ref<string | undefined>(undefined)
const dateUntil = ref<string | undefined>(undefined)
const sortOrder = ref<'desc' | 'asc'>('desc')
const viewMode = ref<'grid' | 'list'>('grid')

let activeRequestId = 0

let searchTimer: ReturnType<typeof setTimeout> | null = null
watch(searchQuery, (val) => {
  if (searchTimer) clearTimeout(searchTimer)
  searchTimer = setTimeout(() => {
    searchDebounced.value = val
  }, 400)
})
onBeforeUnmount(() => {
  if (searchTimer) clearTimeout(searchTimer)
})
const previewOpen = ref(false)
const previewStartIndex = ref(0)

const typeOptions = [
  { label: 'All', value: 'all' },
  { label: 'Images', value: 'image' },
  { label: 'Videos', value: 'video' },
  { label: 'Documents', value: 'document' },
  { label: 'Audio', value: 'audio' }
]

async function fetchPage(cursor: string | null = null): Promise<MediaPage | null> {
  const params = new URLSearchParams()
  params.set('limit', '100')
  if (cursor) params.set('cursor', cursor)
  if (typeFilter.value !== 'all') params.set('type', typeFilter.value)
  if (searchDebounced.value) params.set('search', searchDebounced.value)
  if (dateSince.value) params.set('since', dateSince.value)
  if (dateUntil.value) params.set('until', dateUntil.value)
  if (sortOrder.value) params.set('sort', sortOrder.value)

  const res: { data: unknown } = await api(`/sessions/${sessionId.value}/media?${params}`)
  const data = res.data as { items: MediaMessage[], nextCursor: string | null, total: number } | null
  return {
    items: data?.items || [],
    nextCursor: data?.nextCursor || null,
    total: data?.total || 0
  }
}

async function loadInitial() {
  const requestId = ++activeRequestId
  loading.value = true
  loadingMore.value = false
  items.value = []
  totalCount.value = 0
  nextCursor.value = null
  failedCursor.value = null
  try {
    const page = await fetchPage()
    if (requestId !== activeRequestId) return
    if (page) {
      items.value = page.items
      nextCursor.value = page.nextCursor
      totalCount.value = page.total
    }
  } catch {
    if (requestId !== activeRequestId) return
    items.value = []
    totalCount.value = 0
  } finally {
    if (requestId === activeRequestId) {
      loading.value = false
    }
  }
}

async function loadMore() {
  const cursor = nextCursor.value
  const requestId = activeRequestId
  if (loadingMore.value || !cursor || failedCursor.value === cursor) return
  loadingMore.value = true
  try {
    const page = await fetchPage(cursor)
    if (requestId !== activeRequestId) return
    if (page) {
      items.value.push(...page.items)
      nextCursor.value = page.nextCursor
      totalCount.value = page.total
      failedCursor.value = null
    }
  } catch {
    if (requestId === activeRequestId) {
      failedCursor.value = cursor
    }
  } finally {
    if (requestId === activeRequestId) {
      loadingMore.value = false
    }
  }
}

watch([typeFilter, searchDebounced, dateSince, dateUntil, sortOrder], loadInitial)
onMounted(loadInitial)
watch(sessionId, loadInitial)

// Virtual scroll setup
const parentRef = ref<HTMLElement | null>(null)
const CARD_SIZE = 200 // approximate height including gap
const LIST_ITEM_SIZE = 80 // approximate height for list view

// Detect responsive column count from the scroll container width
const columnCount = ref(5)
function updateColumnCount() {
  if (!parentRef.value || viewMode.value !== 'grid') {
    columnCount.value = 1
    return
  }
  const w = parentRef.value.clientWidth
  if (w >= 1280) columnCount.value = 5 // xl
  else if (w >= 1024) columnCount.value = 4 // lg
  else if (w >= 640) columnCount.value = 3 // sm
  else columnCount.value = 2
}

const resizeObserver = ref<ResizeObserver | null>(null)
onMounted(() => {
  resizeObserver.value = new ResizeObserver(() => updateColumnCount())
})

watch([parentRef, viewMode], ([el, mode]) => {
  resizeObserver.value?.disconnect()
  if (el && mode === 'grid') {
    resizeObserver.value?.observe(el)
    updateColumnCount()
  } else {
    columnCount.value = 1
  }
})
onBeforeUnmount(() => resizeObserver.value?.disconnect())

const rowVirtualizerOptions = computed(() => ({
  count: viewMode.value === 'grid' ? Math.ceil(items.value.length / columnCount.value) : items.value.length,
  getScrollElement: () => parentRef.value,
  estimateSize: () => viewMode.value === 'grid' ? CARD_SIZE : LIST_ITEM_SIZE,
  overscan: 5
}))

const rowVirtualizer = useVirtualizer(rowVirtualizerOptions)

const virtualRows = computed(() => rowVirtualizer.value.getVirtualItems())
const totalSize = computed(() => rowVirtualizer.value.getTotalSize())

// Infinite scroll trigger
watchEffect(() => {
  const [lastItem] = [...virtualRows.value].reverse()
  if (!lastItem) return
  const threshold = viewMode.value === 'grid' ? Math.ceil(items.value.length / columnCount.value) - 3 : items.value.length - 3
  if (lastItem.index >= threshold && nextCursor.value && !loadingMore.value) {
    loadMore()
  }
})

// Thumbnails
const thumbnails = ref<Record<string, string>>({})
const thumbLoading = ref<Set<string>>(new Set())

async function fetchMediaBlob(messageId: string): Promise<Blob> {
  const res: { data: unknown } = await api(`/sessions/${sessionId.value}/media/${messageId}`)
  const presignedUrl = (res.data as { url?: string } | null)?.url
  if (!presignedUrl) throw new Error('No media URL returned')
  return await $fetch<Blob>('/api/media-proxy', {
    query: { url: presignedUrl },
    responseType: 'blob'
  })
}

async function loadThumbIfNeeded(msgId: string) {
  if (thumbnails.value[msgId] || thumbLoading.value.has(msgId)) return
  thumbLoading.value.add(msgId)
  try {
    const blob = await fetchMediaBlob(msgId)
    thumbnails.value[msgId] = URL.createObjectURL(blob)
  } catch {
    // skip
  }
  thumbLoading.value.delete(msgId)
}

// Revoke blob URLs for items no longer in the list
watch(items, (newItems, oldItems) => {
  if (!oldItems) return
  const newIds = new Set(newItems.map(i => i.id))
  for (const old of oldItems) {
    const thumbUrl = thumbnails.value[old.id]
    if (!newIds.has(old.id) && thumbUrl) {
      URL.revokeObjectURL(thumbUrl)
      Reflect.deleteProperty(thumbnails.value, old.id)
    }
  }
})

onBeforeUnmount(() => {
  for (const url of Object.values(thumbnails.value)) {
    URL.revokeObjectURL(url)
  }
  thumbnails.value = {}
})

const observer = ref<IntersectionObserver | null>(null)
const observedElements = new WeakMap<HTMLElement, string>()

onMounted(() => {
  observer.value = new IntersectionObserver(
    (entries) => {
      for (const entry of entries) {
        if (entry.isIntersecting) {
          const el = entry.target as HTMLElement
          const msgId = el.dataset.msgId
          const msgType = el.dataset.msgType
          if (msgId && msgType && ['image', 'sticker'].includes(msgType)) {
            loadThumbIfNeeded(msgId)
          }
          // Unobserve after triggering load — no need to keep watching
          observer.value?.unobserve(el)
        }
      }
    },
    { rootMargin: '200px' }
  )
})

onBeforeUnmount(() => {
  observer.value?.disconnect()
})

function observeCard(el: Element | ComponentPublicInstance | null, msg: MediaMessage) {
  if (el && observer.value) {
    const htmlEl = (el as { $el?: Element })?.$el ?? el
    if (htmlEl instanceof HTMLElement) {
      htmlEl.dataset.msgId = msg.id
      htmlEl.dataset.msgType = msg.msgType
      // Unobserve previous element for this message if any
      const prev = observedElements.get(htmlEl)
      if (prev === msg.id) return // already observing for this msg
      observedElements.set(htmlEl, msg.id)
      observer.value.observe(htmlEl)
    }
  }
}

function mediaIcon(type: string): string {
  const map: Record<string, string> = {
    image: 'i-lucide-image',
    video: 'i-lucide-video',
    document: 'i-lucide-file-text',
    audio: 'i-lucide-music',
    sticker: 'i-lucide-sticker'
  }
  return map[type] || 'i-lucide-file'
}

const imageItems = computed(() => items.value.filter(m => ['image', 'sticker'].includes(m.msgType)))

async function downloadMedia(msg: MediaMessage) {
  try {
    const blob = await fetchMediaBlob(msg.id)
    const url = URL.createObjectURL(blob)
    const link = document.createElement('a')
    link.href = url
    link.download = msg.body || msg.id
    link.click()
    setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch {
    useToast().add({ title: 'Failed to download media', color: 'error' })
  }
}

function previewImage(msg: MediaMessage) {
  const idx = imageItems.value.findIndex(m => m.id === msg.id)
  previewStartIndex.value = idx >= 0 ? idx : 0
  previewOpen.value = true
}

function getRowItems(rowIndex: number): MediaMessage[] {
  const start = rowIndex * columnCount.value
  const end = start + columnCount.value
  return items.value.slice(start, end)
}

function getListItem(rowIndex: number): MediaMessage | null {
  return items.value[rowIndex] ?? null
}

const gridClass = computed(() => {
  const colsMap: Record<number, string> = {
    2: 'grid-cols-2',
    3: 'grid-cols-3',
    4: 'grid-cols-4',
    5: 'grid-cols-5',
  }
  return `grid gap-3 px-4 ${colsMap[columnCount.value] ?? 'grid-cols-2'}`
})
</script>

<template>
  <UDashboardPanel id="session-media">
    <template #header>
      <UDashboardNavbar title="Media">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
        <template #right>
          <UButton
            icon="i-lucide-refresh-cw"
            color="neutral"
            variant="ghost"
            size="sm"
            @click="loadInitial"
          />
        </template>
      </UDashboardNavbar>

      <UDashboardToolbar>
        <template #left>
          <div class="flex items-center gap-2 flex-wrap">
            <USelectMenu
              v-model="typeFilter"
              :items="typeOptions"
              value-key="value"
              :search-input="false"
              size="sm"
              color="neutral"
              class="w-36"
            />
            <UInput
              v-model="searchQuery"
              placeholder="Search..."
              icon="i-lucide-search"
              size="sm"
              color="neutral"
              class="w-48"
            />
            <UInput
              v-model="dateSince"
              type="date"
              placeholder="Since"
              size="sm"
              color="neutral"
              class="w-36"
            />
            <UInput
              v-model="dateUntil"
              type="date"
              placeholder="Until"
              size="sm"
              color="neutral"
              class="w-36"
            />
            <USelectMenu
              v-model="sortOrder"
              :items="[{ label: 'Newest first', value: 'desc' }, { label: 'Oldest first', value: 'asc' }]"
              value-key="value"
              :search-input="false"
              size="sm"
              color="neutral"
              class="w-36"
            />
            <UButton
              :icon="viewMode === 'grid' ? 'i-lucide-list' : 'i-lucide-grid'"
              size="sm"
              color="neutral"
              variant="ghost"
              @click="viewMode = viewMode === 'grid' ? 'list' : 'grid'"
            />
          </div>
        </template>
        <template #right>
          <span class="text-sm text-muted">{{ totalCount }} item(s)</span>
        </template>
      </UDashboardToolbar>
    </template>

    <template #body>
      <div v-if="loading" class="flex items-center justify-center py-24">
        <UIcon name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
      </div>

      <div
        v-else-if="items.length === 0"
        class="flex flex-col items-center justify-center py-24 gap-3 text-muted"
      >
        <UIcon name="i-lucide-image" class="size-10" />
        <p class="text-sm">
          No media found.
        </p>
      </div>

      <div
        v-else
        ref="parentRef"
        class="h-[calc(100vh-8rem)] overflow-auto"
      >
        <div :style="{ height: `${totalSize}px`, width: '100%', position: 'relative' }">
          <div
            v-for="virtualRow in virtualRows"
            :key="String(virtualRow.key)"
            :style="{
              position: 'absolute',
              top: 0,
              left: 0,
              width: '100%',
              height: `${virtualRow.size}px`,
              transform: `translateY(${virtualRow.start}px)`
            }"
            :class="viewMode === 'grid' ? gridClass : 'flex flex-col gap-2 px-4'"
          >
            <template v-if="viewMode === 'grid'">
              <template v-for="msg in getRowItems(virtualRow.index)" :key="msg.id">
                <UCard
                  :ref="(el: Element | ComponentPublicInstance | null) => observeCard(el, msg)"
                  class="cursor-pointer hover:ring-1 hover:ring-primary transition-all"
                  :ui="{ body: '!p-3' }"
                  @click="msg.msgType === 'image' ? previewImage(msg) : downloadMedia(msg)"
                >
                  <div class="flex flex-col items-center gap-2">
                    <div class="flex size-24 items-center justify-center rounded-lg bg-elevated overflow-hidden">
                      <img
                        v-if="thumbnails[msg.id]"
                        :src="thumbnails[msg.id]"
                        class="size-full object-cover"
                        alt=""
                      >
                      <UIcon v-else :name="mediaIcon(msg.msgType)" class="size-8 text-muted" />
                    </div>
                    <div class="w-full text-center space-y-0.5">
                      <p class="text-xs font-medium truncate">
                        {{ msg.body || msg.msgType }}
                      </p>
                      <p class="text-[10px] text-muted truncate">
                        {{ msg.chatJid.split('@')[0] }}
                      </p>
                      <p class="text-[10px] text-dimmed">
                        {{ new Date(msg.timestamp).toLocaleString() }}
                      </p>
                    </div>
                  </div>
                  <template #footer>
                    <div class="flex items-center justify-between">
                      <UBadge
                        :label="msg.msgType"
                        variant="subtle"
                        size="xs"
                        class="capitalize"
                      />
                      <UButton
                        icon="i-lucide-download"
                        size="xs"
                        color="neutral"
                        variant="ghost"
                        @click.stop="downloadMedia(msg)"
                      />
                    </div>
                  </template>
                </UCard>
              </template>
            </template>
            <template v-else>
              <UCard
                v-if="getListItem(virtualRow.index)"
                :ref="(el: Element | ComponentPublicInstance | null) => observeCard(el, getListItem(virtualRow.index)!)"
                class="cursor-pointer hover:ring-1 hover:ring-primary transition-all"
                :ui="{ body: '!p-3' }"
                @click="getListItem(virtualRow.index)!.msgType === 'image' ? previewImage(getListItem(virtualRow.index)!) : downloadMedia(getListItem(virtualRow.index)!)"
              >
                <div class="flex items-center gap-3">
                  <div class="flex size-12 items-center justify-center rounded-lg bg-elevated overflow-hidden flex-shrink-0">
                    <img
                      v-if="thumbnails[getListItem(virtualRow.index)!.id]"
                      :src="thumbnails[getListItem(virtualRow.index)!.id]"
                      class="size-full object-cover"
                      alt=""
                    >
                    <UIcon v-else :name="mediaIcon(getListItem(virtualRow.index)!.msgType)" class="size-6 text-muted" />
                  </div>
                  <div class="flex-1 min-w-0 space-y-0.5">
                    <p class="text-sm font-medium truncate">
                      {{ getListItem(virtualRow.index)!.body || getListItem(virtualRow.index)!.msgType }}
                    </p>
                    <p class="text-xs text-muted truncate">
                      {{ getListItem(virtualRow.index)!.chatJid.split('@')[0] }}
                    </p>
                    <p class="text-[10px] text-dimmed">
                      {{ new Date(getListItem(virtualRow.index)!.timestamp).toLocaleString() }}
                    </p>
                  </div>
                </div>
                <template #footer>
                  <div class="flex items-center justify-between">
                    <UBadge
                      :label="getListItem(virtualRow.index)!.msgType"
                      variant="subtle"
                      size="xs"
                      class="capitalize"
                    />
                    <UButton
                      icon="i-lucide-download"
                      size="xs"
                      color="neutral"
                      variant="ghost"
                      @click.stop="downloadMedia(getListItem(virtualRow.index)!)"
                    />
                  </div>
                </template>
              </UCard>
            </template>
          </div>
        </div>
        <div v-if="loadingMore" class="flex items-center justify-center py-4">
          <UIcon name="i-lucide-loader-2" class="size-6 animate-spin text-muted" />
        </div>
      </div>

      <UModal
        v-model:open="previewOpen"
        title="Image Preview"
        description="Media file preview"
        :ui="{ title: 'sr-only', description: 'sr-only', body: '!p-2' }"
      >
        <template #body>
          <UCarousel
            v-if="imageItems.length"
            :items="imageItems"
            :start-index="previewStartIndex"
            arrows
            :ui="{ item: 'basis-full' }"
          >
            <template #default="{ item }">
              <div class="flex items-center justify-center w-full min-h-64">
                <img
                  v-if="thumbnails[item.id]"
                  :src="thumbnails[item.id]"
                  class="max-h-[70vh] w-auto mx-auto rounded-lg object-contain"
                  :alt="item.body || 'Image'"
                >
                <UIcon v-else name="i-lucide-loader-2" class="size-8 animate-spin text-muted" />
              </div>
              <p class="text-xs text-center text-muted mt-2 truncate">
                {{ item.body || item.msgType }} — {{ new Date(item.timestamp).toLocaleString() }}
              </p>
            </template>
          </UCarousel>
        </template>
      </UModal>
    </template>
  </UDashboardPanel>
</template>
