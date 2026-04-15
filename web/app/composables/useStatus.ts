export interface Status {
  id: string
  sessionId: string
  senderJid: string
  senderName?: string
  fromMe: boolean
  statusType: string
  body: string
  mediaType?: string
  mediaUrl?: string
  timestamp: string
  expiresAt: string
  createdAt: string
}

export function useStatus() {
  const { api } = useWzap()

  async function fetchStatuses(sessionId: string, limit = 50, offset = 0): Promise<Status[]> {
    const res: { data: Status[] } = await api(`/sessions/${sessionId}/stories?limit=${limit}&offset=${offset}`)
    return res.data || []
  }

  async function fetchContactStatuses(sessionId: string, senderJid: string): Promise<Status[]> {
    const res: { data: Status[] } = await api(`/sessions/${sessionId}/stories/${encodeURIComponent(senderJid)}`)
    return res.data || []
  }

  async function sendStatusText(sessionId: string, text: string, backgroundColor?: string, font?: number): Promise<string> {
    const res: { data: { mid: string } } = await api(`/sessions/${sessionId}/stories/text`, {
      method: 'POST',
      body: { text, backgroundColor, font }
    })
    return res.data?.mid || ''
  }

  async function sendStatusImage(sessionId: string, payload: { mimeType: string; base64?: string; url?: string; caption?: string }): Promise<string> {
    const res: { data: { mid: string } } = await api(`/sessions/${sessionId}/stories/image`, {
      method: 'POST',
      body: payload
    })
    return res.data?.mid || ''
  }

  async function sendStatusVideo(sessionId: string, payload: { mimeType: string; base64?: string; url?: string; caption?: string }): Promise<string> {
    const res: { data: { mid: string } } = await api(`/sessions/${sessionId}/stories/video`, {
      method: 'POST',
      body: payload
    })
    return res.data?.mid || ''
  }

  return { fetchStatuses, fetchContactStatuses, sendStatusText, sendStatusImage, sendStatusVideo }
}