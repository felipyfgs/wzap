<script setup lang="ts">
definePageMeta({ layout: false })

const { setToken, setApiBase, apiBase } = useWzap()
const tokenInput = ref('')
const apiBaseInput = ref(apiBase.value)
const error = ref('')
const loading = ref(false)

async function handleLogin() {
  loading.value = true
  error.value = ''
  setApiBase(apiBaseInput.value)
  setToken(tokenInput.value)

  try {
    await $fetch(`${apiBaseInput.value}/sessions`, {
      headers: { Authorization: tokenInput.value }
    })
    navigateTo('/')
  } catch {
    error.value = 'Invalid token or API unreachable'
    setToken('')
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-(--ui-bg)">
    <div class="w-full max-w-sm">
      <div class="text-center mb-8">
        <h1 class="text-3xl font-bold text-(--ui-text-highlighted)">wzap</h1>
        <p class="text-sm text-(--ui-text-muted) mt-1">WhatsApp API Manager</p>
      </div>

      <UCard>
        <form @submit.prevent="handleLogin" class="space-y-4">
          <UFormField label="API URL">
            <UInput v-model="apiBaseInput" placeholder="http://localhost:8080" class="w-full" />
          </UFormField>

          <UFormField label="Admin Token">
            <UInput v-model="tokenInput" type="password" placeholder="Your API key" class="w-full" />
          </UFormField>

          <p v-if="error" class="text-sm text-red-500">{{ error }}</p>

          <UButton type="submit" block :loading="loading" color="primary">
            Login
          </UButton>
        </form>
      </UCard>
    </div>
  </div>
</template>
