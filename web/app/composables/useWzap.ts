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

  const api = async <T = any>(path: string, options: any = {}): Promise<T> => {
    return await $fetch<T>(`/api${path}`, {
      ...options,
      headers: {
        'Authorization': token.value,
        'Content-Type': 'application/json',
        ...options.headers
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
