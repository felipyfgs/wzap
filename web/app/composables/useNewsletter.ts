export function useNewsletter(sessionId: Ref<string> | string) {
  const { api } = useWzap()
  const resolveId = () => typeof sessionId === 'string' ? sessionId : sessionId.value

  async function listNewsletters() {
    const res: { data: unknown } = await api(`/sessions/${resolveId()}/newsletter/list`)
    return res.data || []
  }

  async function createNewsletter(body: { name: string, description?: string, picture?: string }) {
    const res: { data: unknown } = await api(`/sessions/${resolveId()}/newsletter/create`, { method: 'POST', body })
    return res.data
  }

  async function getNewsletterInfo(newsletterJid: string) {
    const res: { data: unknown } = await api(`/sessions/${resolveId()}/newsletter/info`, { method: 'POST', body: { newsletterJid } })
    return res.data
  }

  async function getInviteLink(newsletterJid: string) {
    const res: { data: unknown } = await api(`/sessions/${resolveId()}/newsletter/invite`, { method: 'POST', body: { newsletterJid } })
    return res.data
  }

  async function getMessages(newsletterJid: string, count?: number) {
    const res: { data: unknown } = await api(`/sessions/${resolveId()}/newsletter/messages`, { method: 'POST', body: { newsletterJid, count: count || 50 } })
    return res.data || []
  }

  async function subscribe(newsletterJid: string) {
    await api(`/sessions/${resolveId()}/newsletter/subscribe`, { method: 'POST', body: { newsletterJid } })
  }

  async function unsubscribe(newsletterJid: string) {
    await api(`/sessions/${resolveId()}/newsletter/unsubscribe`, { method: 'POST', body: { newsletterJid } })
  }

  async function muteNewsletter(newsletterJid: string, mute: boolean) {
    await api(`/sessions/${resolveId()}/newsletter/mute`, { method: 'POST', body: { newsletterJid, mute } })
  }

  async function react(newsletterJid: string, messageServerID: string, reaction: string) {
    await api(`/sessions/${resolveId()}/newsletter/react`, { method: 'POST', body: { newsletterJid, messageServerID, reaction } })
  }

  async function markViewed(newsletterJid: string, messageServerIDs: string[]) {
    await api(`/sessions/${resolveId()}/newsletter/viewed`, { method: 'POST', body: { newsletterJid, messageServerIDs } })
  }

  return {
    listNewsletters,
    createNewsletter,
    getNewsletterInfo,
    getInviteLink,
    getMessages,
    subscribe,
    unsubscribe,
    muteNewsletter,
    react,
    markViewed
  }
}
