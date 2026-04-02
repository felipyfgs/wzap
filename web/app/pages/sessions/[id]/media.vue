<script setup lang="ts">
const route = useRoute()
const sessionId = computed(() => route.params.id as string)
</script>

<template>
  <UDashboardPanel id="session-media">
    <template #header>
      <UDashboardNavbar title="Media">
        <template #leading>
          <UDashboardSidebarCollapse />
        </template>
      </UDashboardNavbar>
    </template>

    <template #body>
      <div class="flex flex-col items-center justify-center py-24 gap-3 text-muted">
        <UIcon name="i-lucide-image" class="size-10" />
        <p class="text-sm">Media retrieval is available via API.</p>
        <UButton
          label="View API Docs"
          icon="i-lucide-book-open"
          variant="soft"
          to="/swagger"
          target="_blank"
        />
      </div>
    </template>
  </UDashboardPanel>
</template>
