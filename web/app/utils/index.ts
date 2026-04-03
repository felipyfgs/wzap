export function randomInt(min: number, max: number): number {
  return Math.floor(Math.random() * (max - min + 1)) + min
}

export function randomFrom<T>(array: T[]): T {
  return array[Math.floor(Math.random() * array.length)]!
}

export function sessionStatusColor(status: string): 'success' | 'warning' | 'error' | 'neutral' | 'info' {
  const map: Record<string, 'success' | 'warning' | 'error' | 'neutral' | 'info'> = {
    connected: 'success',
    connecting: 'warning',
    pairing: 'info',
    disconnected: 'neutral',
    error: 'error'
  }
  return map[status?.toLowerCase()] ?? 'neutral'
}

export function parseJID(jid: string): { phone: string | null; device: number } {
  if (!jid) return { phone: null, device: 0 }
  const withoutServer = jid.replace(/@.*$/, '')
  const colonIdx = withoutServer.indexOf(':')
  if (colonIdx === -1) return { phone: withoutServer, device: 0 }
  return {
    phone: withoutServer.slice(0, colonIdx),
    device: parseInt(withoutServer.slice(colonIdx + 1), 10) || 0
  }
}
