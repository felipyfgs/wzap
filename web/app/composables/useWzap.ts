const TOKEN_KEY = 'wzap_token'

export function useWzap() {
  const token = useState<string>(TOKEN_KEY, () => {
    if (import.meta.client) {
      return localStorage.getItem(TOKEN_KEY) || ''
    }
    return ''
  })

  const setToken = (t: string) => {
    token.value = t
    if (import.meta.client) {
      localStorage.setItem(TOKEN_KEY, t)
    }
  }

  const api = async <T = unknown>(path: string, options: Omit<RequestInit, 'body'> & { body?: unknown } = {}): Promise<T> => {
    const headers: Record<string, string> = {
      Authorization: token.value
    }
    if (options.body !== undefined && !(options.body instanceof FormData)) {
      headers['Content-Type'] = 'application/json'
    }
    return await $fetch<T>(`/api${path}`, {
      ...options,
      headers: {
        ...headers,
        ...(options.headers as Record<string, string>)
      }
    })
  }

  const isAuthenticated = computed(() => !!token.value)

  const logout = () => {
    token.value = ''
    if (import.meta.client) {
      localStorage.removeItem(TOKEN_KEY)
    }
  }

  return { api, token, setToken, isAuthenticated, logout }
}
