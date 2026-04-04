export function useChatOperations(sessionId: Ref<string> | string) {
  const { api } = useWzap()
  const resolveId = () => typeof sessionId === 'string' ? sessionId : sessionId.value

  async function archiveChat(jid: string) {
    await api(`/sessions/${resolveId()}/chat/archive`, { method: 'POST', body: { jid } })
  }

  async function unarchiveChat(jid: string) {
    await api(`/sessions/${resolveId()}/chat/unarchive`, { method: 'POST', body: { jid } })
  }

  async function muteChat(jid: string) {
    await api(`/sessions/${resolveId()}/chat/mute`, { method: 'POST', body: { jid } })
  }

  async function unmuteChat(jid: string) {
    await api(`/sessions/${resolveId()}/chat/unmute`, { method: 'POST', body: { jid } })
  }

  async function pinChat(jid: string) {
    await api(`/sessions/${resolveId()}/chat/pin`, { method: 'POST', body: { jid } })
  }

  async function unpinChat(jid: string) {
    await api(`/sessions/${resolveId()}/chat/unpin`, { method: 'POST', body: { jid } })
  }

  async function deleteChat(jid: string) {
    await api(`/sessions/${resolveId()}/chat/delete`, { method: 'POST', body: { jid } })
  }

  async function markRead(jid: string, messageIds: string[]) {
    await api(`/sessions/${resolveId()}/chat/read`, { method: 'POST', body: { jid, messageIds } })
  }

  async function markUnread(jid: string) {
    await api(`/sessions/${resolveId()}/chat/unread`, { method: 'POST', body: { jid } })
  }

  return {
    archiveChat,
    unarchiveChat,
    muteChat,
    unmuteChat,
    pinChat,
    unpinChat,
    deleteChat,
    markRead,
    markUnread
  }
}
