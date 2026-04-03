import { createSharedComposable } from '@vueuse/core'
import type { Session, SessionProfile } from '~/types'

const _useSession = () => {
  const { api, isAuthenticated } = useWzap()

  const sessions = useState<Session[]>('session_list', () => [])
  const current = useState<Session | null>('session_current', () => null)
  const profile = useState<SessionProfile | null>('session_profile', () => null)
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

  async function refreshProfile(id: string) {
    try {
      const res: any = await api(`/sessions/${id}/profile`)
      profile.value = res.data
    } catch {
      profile.value = null
    }
  }

  async function refreshCurrent(id: string) {
    try {
      const res: any = await api(`/sessions/${id}`)
      current.value = res.data
      if (res.data?.status === 'connected') {
        await refreshProfile(id)
      } else {
        profile.value = null
      }
    } catch {
      current.value = null
      profile.value = null
    }
  }

  return { sessions, current, profile, loadingSessions, refreshSessions, refreshCurrent }
}

export const useSession = createSharedComposable(_useSession)
