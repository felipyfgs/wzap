export type MessageType
  = | 'text'
    | 'image'
    | 'video'
    | 'document'
    | 'audio'
    | 'contact'
    | 'location'
    | 'poll'
    | 'sticker'
    | 'link'
    | 'button'
    | 'list'

export const MESSAGE_TYPE_OPTIONS: { label: string, value: MessageType }[] = [
  { label: 'Text', value: 'text' },
  { label: 'Image', value: 'image' },
  { label: 'Video', value: 'video' },
  { label: 'Document', value: 'document' },
  { label: 'Audio', value: 'audio' },
  { label: 'Contact', value: 'contact' },
  { label: 'Location', value: 'location' },
  { label: 'Poll', value: 'poll' },
  { label: 'Sticker', value: 'sticker' },
  { label: 'Link', value: 'link' },
  { label: 'Button', value: 'button' },
  { label: 'List', value: 'list' }
]

const TYPE_TO_ENDPOINT: Record<MessageType, string> = {
  text: 'text',
  image: 'image',
  video: 'video',
  document: 'document',
  audio: 'audio',
  contact: 'contact',
  location: 'location',
  poll: 'poll',
  sticker: 'sticker',
  link: 'link',
  button: 'button',
  list: 'list'
}

export interface SendTextPayload {
  phone: string
  body: string
}

export interface SendMediaPayload {
  phone: string
  mimeType: string
  base64?: string
  url?: string
  caption?: string
  fileName?: string
}

export interface SendContactPayload {
  phone: string
  name: string
  vcard: string
}

export interface SendLocationPayload {
  phone: string
  latitude: number
  longitude: number
  name?: string
  address?: string
}

export interface SendPollPayload {
  phone: string
  name: string
  options: string[]
  selectableCount?: number
}

export interface SendStickerPayload {
  phone: string
  mimeType: string
  base64: string
}

export interface SendLinkPayload {
  phone: string
  url: string
  title?: string
  description?: string
}

export interface ButtonItem {
  id: string
  text: string
}

export interface SendButtonPayload {
  phone: string
  body: string
  footer?: string
  buttons: ButtonItem[]
}

export interface ListRow {
  id: string
  title: string
  description?: string
}

export interface ListSection {
  title: string
  rows: ListRow[]
}

export interface SendListPayload {
  phone: string
  title: string
  body: string
  footer?: string
  buttonText: string
  sections: ListSection[]
}

export type SendMessagePayload
  = | SendTextPayload
    | SendMediaPayload
    | SendContactPayload
    | SendLocationPayload
    | SendPollPayload
    | SendStickerPayload
    | SendLinkPayload
    | SendButtonPayload
    | SendListPayload

export function useMessageSender() {
  const { api } = useWzap()

  async function sendMessage(sessionId: string, type: MessageType, payload: SendMessagePayload) {
    const endpoint = TYPE_TO_ENDPOINT[type]
    return await api(`/sessions/${sessionId}/messages/${endpoint}`, {
      method: 'POST',
      body: payload
    })
  }

  return { sendMessage }
}
