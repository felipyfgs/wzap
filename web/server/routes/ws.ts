export default defineWebSocketHandler({
  open(peer) {
    const config = useRuntimeConfig()
    const target = config.apiUrl.replace(/^http/, 'ws') + '/ws'

    const reqUrl = peer.request?.url ?? ''
    const qs = reqUrl.includes('?') ? reqUrl.slice(reqUrl.indexOf('?')) : ''

    const upstream = new WebSocket(target + qs)
    ;(peer as any)._upstream = upstream

    upstream.addEventListener('message', e => peer.send(e.data as string))
    upstream.addEventListener('close', e => peer.close(e.code))
    upstream.addEventListener('error', () => peer.close(1011))
  },

  message(peer, msg) {
    const upstream: WebSocket | undefined = (peer as any)._upstream
    if (upstream?.readyState === WebSocket.OPEN) {
      upstream.send(msg.text())
    }
  },

  close(peer) {
    const upstream: WebSocket | undefined = (peer as any)._upstream
    upstream?.close()
  }
})
