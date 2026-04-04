export interface SessionProfile {
  pushName?: string
  businessName?: string
  platform?: string
  pictureUrl?: string
  status?: string
}

export interface SessionProxy {
  host?: string
  port?: number
  protocol?: string
  username?: string
  password?: string
}

export interface SessionSettings {
  alwaysOnline: boolean
  rejectCall: boolean
  msgRejectCall?: string
  readMessages: boolean
  ignoreGroups: boolean
  ignoreStatus: boolean
}

export interface Session {
  id: string
  name: string
  apiKey?: string
  jid?: string
  qrCode?: string
  connected: number
  status: string
  engine?: string
  phoneNumberId?: string
  businessAccountId?: string
  pushName?: string
  businessName?: string
  platform?: string
  chatwootEnabled?: boolean
  proxy?: SessionProxy
  settings?: SessionSettings
  createdAt?: string
  updatedAt?: string
}

export interface SessionCreated extends Omit<Session, 'pushName' | 'businessName' | 'platform' | 'chatwootEnabled'> {
  token: string
  webhook?: Webhook
}

export interface Webhook {
  id: string
  sessionId: string
  url: string
  secret?: string
  events: string[]
  enabled: boolean
  natsEnabled: boolean
  createdAt?: string
}
