const TOKEN_KEY = 'wzap_token'
const API_BASE_KEY = 'wzap_api_base'

export function useWzap() {
  const token = useState<string>(TOKEN_KEY, () => {
    if (import.meta.client) {
      return localStorage.getItem(TOKEN_KEY) || ''
    }
    return ''
  })

  const apiBase = useState<string>(API_BASE_KEY, () => {
    if (import.meta.client) {
      return localStorage.getItem(API_BASE_KEY) || 'http://localhost:8080'
    }
    return 'http://localhost:8080'
  })

  const setToken = (t: string) => {
    token.value = t
    if (import.meta.client) {
      localStorage.setItem(TOKEN_KEY, t)
    }
  }

  const setApiBase = (url: string) => {
    apiBase.value = url
    if (import.meta.client) {
      localStorage.setItem(API_BASE_KEY, url)
    }
  }

  const api = async <T = any>(path: string, options: any = {}): Promise<T> => {
    return await $fetch<T>(`${apiBase.value}${path}`, {
      ...options,
      headers: {
        Authorization: token.value,
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

  return { api, token, apiBase, setToken, setApiBase, isAuthenticated, logout }
}
