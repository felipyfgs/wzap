import { defineEventHandler, getQuery, setResponseHeaders, createError } from 'h3'

function getAllowedHosts(): Set<string> {
  const config = useRuntimeConfig()
  const hosts = new Set<string>()

  const apiBase = new URL(config.apiUrl as string)
  hosts.add(apiBase.hostname)
  if (apiBase.port) hosts.add(apiBase.host)

  try {
    const minioBase = new URL(config.minioEndpoint as string)
    hosts.add(minioBase.hostname)
    if (minioBase.port) hosts.add(minioBase.host)
  } catch {
    // minioEndpoint inválida — ignora
  }

  return hosts
}

export default defineEventHandler(async (event) => {
  const { url } = getQuery(event)

  if (!url || typeof url !== 'string') {
    throw createError({ statusCode: 400, statusMessage: 'Missing url parameter' })
  }

  // SSRF protection: only allow known storage/API hosts
  let parsedUrl: URL
  try {
    parsedUrl = new URL(url)
  } catch {
    throw createError({ statusCode: 400, statusMessage: 'Invalid url parameter' })
  }

  if (parsedUrl.protocol !== 'http:' && parsedUrl.protocol !== 'https:') {
    throw createError({ statusCode: 400, statusMessage: 'Unsupported protocol' })
  }

  const allowedHosts = getAllowedHosts()
  const hostOk = allowedHosts.has(parsedUrl.hostname) || allowedHosts.has(parsedUrl.host)
  if (!hostOk) {
    throw createError({ statusCode: 403, statusMessage: 'Host not allowed' })
  }

  const response = await fetch(url)

  if (!response.ok) {
    throw createError({ statusCode: response.status, statusMessage: 'Failed to fetch media' })
  }

  const contentType = response.headers.get('content-type') || 'application/octet-stream'
  const contentLength = response.headers.get('content-length')

  setResponseHeaders(event, {
    'Content-Type': contentType,
    ...(contentLength ? { 'Content-Length': contentLength } : {})
  })

  return response.body
})
