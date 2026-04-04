export function useLabels(sessionId: Ref<string> | string) {
  const { api } = useWzap()
  const resolveId = () => typeof sessionId === 'string' ? sessionId : sessionId.value

  async function addLabelToChat(jid: string, labelId: string) {
    await api(`/sessions/${resolveId()}/label/chat`, { method: 'POST', body: { jid, labelId } })
  }

  async function removeLabelFromChat(jid: string, labelId: string) {
    await api(`/sessions/${resolveId()}/unlabel/chat`, { method: 'POST', body: { jid, labelId } })
  }

  async function addLabelToMessage(jid: string, labelId: string, messageId: string) {
    await api(`/sessions/${resolveId()}/label/message`, { method: 'POST', body: { jid, labelId, messageId } })
  }

  async function removeLabelFromMessage(jid: string, labelId: string, messageId: string) {
    await api(`/sessions/${resolveId()}/unlabel/message`, { method: 'POST', body: { jid, labelId, messageId } })
  }

  async function editLabel(labelId: string, name: string, color: number, deleted = false) {
    await api(`/sessions/${resolveId()}/label/edit`, { method: 'POST', body: { labelId, name, color, deleted } })
  }

  return {
    addLabelToChat,
    removeLabelFromChat,
    addLabelToMessage,
    removeLabelFromMessage,
    editLabel
  }
}
