<script setup lang="ts">
defineProps<{ collapsed?: boolean }>()
</script>

<template>
  <UButton
    to="/"
    color="neutral"
    variant="ghost"
    block
    :square="collapsed"
    class="py-2"
    :ui="{ label: 'font-semibold text-highlighted' }"
  >
    <template #leading>
      <span class="flex size-5 items-center justify-center rounded-sm bg-primary text-xs font-bold text-white shrink-0">
        W
      </span>
    </template>
    <template v-if="!collapsed" #default>
      wzap
    </template>
  </UButton>
</template>
