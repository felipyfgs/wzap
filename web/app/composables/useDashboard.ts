import { createSharedComposable } from '@vueuse/core'

const _useDashboard = () => {
  const router = useRouter()

  defineShortcuts({
    'g-h': () => router.push('/'),
    'g-s': () => router.push('/sessions'),
    'g-w': () => router.push('/webhooks')
  })

  return {}
}

export const useDashboard = createSharedComposable(_useDashboard)
