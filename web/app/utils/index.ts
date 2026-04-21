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
