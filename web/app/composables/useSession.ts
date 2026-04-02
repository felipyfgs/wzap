import { createSharedComposable } from '@vueuse/core'
import type { Session } from '~/types'

const _useSession = () => {
  const { api, isAuthenticated } = useWzap()

  const sessions = useState<Session[]>('session_list', () => [])
  const current = useState<Session | null>('session_current', () => null)
  const loadingSessions = useState<boolean>('session_list_loading', () => false)

  async function refreshSessions() {
    if (!isAuthenticated.value) return
    loadingSessions.value = true
    try {
      const res: any = await api('/sessions')
      sessions.value = res.data || []
    } catch {
      sessions.value = []
    } finally {
      loadingSessions.value = false
    }
  }

  async function refreshCurrent(id: string) {
    try {
      const res: any = await api(`/sessions/${id}`)
      current.value = res.data
    } catch {
      current.value = null
    }
  }

  return { sessions, current, loadingSessions, refreshSessions, refreshCurrent }
}

export const useSession = createSharedComposable(_useSession)
