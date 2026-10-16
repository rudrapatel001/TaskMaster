// The one place that talks to the backend. It keeps the access token in memory
// (never in localStorage), sends the refresh cookie, and quietly refreshes an
// expired session.

const API_URL: string = import.meta.env.VITE_API_URL ?? 'http://localhost:8080'

export interface User {
  id: string
  email: string
  name: string
  created_at: string
  updated_at: string
}

export interface Session {
  user: User
  access_token: string
  expires_at: string
}

/** A failed API call, with the details the backend sent. */
export class ApiError extends Error {
  status: number
  code: string
  fields: Record<string, string>
  requestId?: string

  constructor(
    status: number,
    code: string,
    message: string,
    fields: Record<string, string> = {},
    requestId?: string,
  ) {
    super(message)
    this.name = 'ApiError'
    this.status = status
    this.code = code
    this.fields = fields
    this.requestId = requestId
  }
}

// The access token lives only in this variable. A page reload clears it, and
// the refresh cookie then gets a new one (see refreshSession).
let accessToken: string | null = null

export function setAccessToken(token: string | null): void {
  accessToken = token
}

// Called when a logged-in request still fails with 401 after a refresh
// attempt, which means the session is really over. The auth provider uses it
// to send the user back to the login page.
let onSessionExpired: (() => void) | null = null

export function setSessionExpiredHandler(handler: (() => void) | null): void {
  onSessionExpired = handler
}

interface RequestOptions {
  method?: 'GET' | 'POST' | 'PATCH' | 'PUT' | 'DELETE'
  body?: unknown
  /** Set to false for endpoints that need no login (login, register, logout). */
  auth?: boolean
}

function send(path: string, { method = 'GET', body, auth = true }: RequestOptions): Promise<Response> {
  const headers: Record<string, string> = {}
  if (body !== undefined) headers['Content-Type'] = 'application/json'
  if (auth && accessToken) headers['Authorization'] = `Bearer ${accessToken}`

  return fetch(`${API_URL}${path}`, {
    method,
    headers,
    body: body === undefined ? undefined : JSON.stringify(body),
    credentials: 'include', // lets the browser send and store the refresh cookie
  })
}

// The two shapes the backend answers with: {"data": ...} or {"error": {...}}.
interface ResponseBody {
  data?: unknown
  error?: {
    code?: string
    message?: string
    fields?: Record<string, string>
    request_id?: string
  }
}

async function parse<T>(res: Response): Promise<T> {
  if (res.status === 204) return undefined as T

  let payload: ResponseBody | null = null
  try {
    payload = (await res.json()) as ResponseBody
  } catch {
    // not JSON (for example a proxy error page); handled below
  }

  if (!res.ok) {
    const e = payload?.error
    throw new ApiError(
      res.status,
      e?.code ?? 'unknown_error',
      e?.message ?? 'Something went wrong',
      e?.fields ?? {},
      e?.request_id,
    )
  }
  return payload?.data as T
}

/**
 * Calls the API and returns the "data" part of the response.
 * If the access token has expired (401), it refreshes the session once and
 * retries the request, so callers never notice.
 */
export async function api<T>(path: string, options: RequestOptions = {}): Promise<T> {
  const tokenUsed = accessToken // remember which token this request was sent with
  let res = await send(path, options)

  if (res.status === 401 && options.auth !== false) {
    if (accessToken !== null && accessToken !== tokenUsed) {
      // While this request was in flight, another request already refreshed
      // the session. No need to refresh again: just retry with the new token.
      res = await send(path, options)
    } else {
      const session = await refreshSession()
      if (session) res = await send(path, options)
    }

    if (res.status === 401) onSessionExpired?.()
  }

  return parse<T>(res)
}

// If several requests discover an expired token at the same moment, they all
// wait for ONE refresh instead of each starting their own. (Each refresh
// replaces the cookie, so parallel refreshes would fight each other.)
let refreshInFlight: Promise<Session | null> | null = null

/** Trades the refresh cookie for a new access token. Null means "not logged in". */
export function refreshSession(): Promise<Session | null> {
  refreshInFlight ??= doRefresh().finally(() => {
    refreshInFlight = null
  })
  return refreshInFlight
}

async function doRefresh(): Promise<Session | null> {
  for (let attempt = 0; attempt < 3; attempt++) {
    try {
      const res = await fetch(`${API_URL}/auth/refresh`, {
        method: 'POST',
        credentials: 'include',
      })

      if (res.ok) {
        const { data } = (await res.json()) as { data: Session }
        accessToken = data.access_token
        return data
      }

      // 409 means another tab is refreshing at this very moment. Wait briefly,
      // then try again: the cookie will have been replaced by then.
      if (res.status === 409) {
        await new Promise((resolve) => setTimeout(resolve, 300))
        continue
      }
    } catch {
      // network error: treat as "not logged in" for now
    }
    break
  }

  accessToken = null
  return null
}