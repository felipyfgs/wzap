<script setup lang="ts">
defineProps<{ state: Record<string, unknown> }>()
</script>

<template>
  <!-- eslint-disable vue/no-mutating-props -->
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
  <!-- eslint-enable vue/no-mutating-props -->
</template>
