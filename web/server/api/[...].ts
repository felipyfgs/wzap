import { defineEventHandler, getRequestHeader, setResponseHeaders, proxyRequest } from 'h3'

export default defineEventHandler(async (event) => {
  const config = useRuntimeConfig()
  const target = config.apiUrl

  const path = event.path.replace(/^\/api/, '')

  setResponseHeaders(event, {
    'Access-Control-Allow-Origin': '*',
    'Access-Control-Allow-Methods': 'GET, POST, PUT, DELETE, OPTIONS',
    'Access-Control-Allow-Headers': 'Content-Type, Authorization'
  })

  if (event.method === 'OPTIONS') {
    return ''
  }

  return await proxyRequest(event, target + path, {
    headers: {
      Authorization: getRequestHeader(event, 'authorization') || ''
    }
  })
})
