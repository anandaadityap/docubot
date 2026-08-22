const TOKEN_KEY = 'docubot_token'

export type ApiError = {
  code: string
  message: string
}

export class HttpError extends Error {
  status: number
  code: string
  constructor(status: number, code: string, message: string) {
    super(message)
    this.status = status
    this.code = code
  }
}

export function getToken() {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string | null) {
  if (token) localStorage.setItem(TOKEN_KEY, token)
  else localStorage.removeItem(TOKEN_KEY)
}

async function parseError(res: Response): Promise<HttpError> {
  try {
    const body = (await res.json()) as { error?: ApiError }
    if (body.error) {
      return new HttpError(res.status, body.error.code, body.error.message)
    }
  } catch {
    /* ignore */
  }
  return new HttpError(res.status, 'HTTP_ERROR', res.statusText || 'request failed')
}

export async function api<T>(path: string, init: RequestInit = {}): Promise<T> {
  const headers = new Headers(init.headers)
  if (init.body && !(init.body instanceof FormData) && !headers.has('Content-Type')) {
    headers.set('Content-Type', 'application/json')
  }
  const token = getToken()
  if (token && !headers.has('Authorization')) {
    headers.set('Authorization', `Bearer ${token}`)
  }
  const res = await fetch(path, { ...init, headers })
  if (res.status === 204) {
    return undefined as T
  }
  if (!res.ok) {
    if (res.status === 401 && path !== '/api/v1/auth/login') {
      setToken(null)
    }
    throw await parseError(res)
  }
  const json = (await res.json()) as { data: T }
  return json.data
}
