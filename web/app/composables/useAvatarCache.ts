const _cache = new Map<string, string | null>()
const _pending = new Map<string, Promise<string | null>>()

export function useAvatarCache() {
  const { api } = useWzap()

  async function fetchAvatar(sessionId: string, jid: string): Promise<string | null> {
    const key = `${sessionId}::${jid}`

    if (_cache.has(key)) return _cache.get(key)!

    if (_pending.has(key)) return _pending.get(key)!

    const promise = api(`/sessions/${sessionId}/contacts/avatar`, {
      method: 'POST',
      body: { phone: jid }
    })
      .then((res: { data: { url: string } }) => {
        const url = res?.data?.url || null
        _cache.set(key, url)
        return url
      })
      .catch(() => {
        _cache.set(key, null)
        return null
      })
      .finally(() => {
        _pending.delete(key)
      })

    _pending.set(key, promise)
    return promise
  }

  function getCached(sessionId: string, jid: string): string | null | undefined {
    return _cache.get(`${sessionId}::${jid}`)
  }

  return { fetchAvatar, getCached }
}
