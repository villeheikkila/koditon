const LEGACY_TOKEN_KEY = 'koditon_access_token'
const RETURN_TO_KEY = 'koditon_return_to'
const REFRESH_SKEW_SECONDS = 60

export type WebSession = {
  access_token: string
  access_token_expires_at: number
  user_id: string
}

let accessToken: string | null = null
let accessTokenExpiresAt = 0
let userId: string | null = null
let inFlightRefresh: Promise<string> | null = null

export function getToken(): string | null {
  return accessToken
}

export function hasAccessToken(): boolean {
  return !!accessToken
}

export function setSession(session: WebSession): void {
  accessToken = session.access_token
  accessTokenExpiresAt = session.access_token_expires_at
  userId = session.user_id
  localStorage.removeItem(LEGACY_TOKEN_KEY)
}

export function clearToken(): void {
  accessToken = null
  accessTokenExpiresAt = 0
  userId = null
  localStorage.removeItem(LEGACY_TOKEN_KEY)
}

export function currentUserId(): string | null {
  return userId
}

export function getFreshTokenOrNull(): string | null {
  if (!accessToken) return null
  if (accessTokenExpiresAt <= nowSeconds() + REFRESH_SKEW_SECONDS) return null
  return accessToken
}

export async function getValidAccessToken(): Promise<string | null> {
  const fresh = getFreshTokenOrNull()
  if (fresh) return fresh
  return refreshSession()
}

export async function refreshSession(): Promise<string> {
  if (inFlightRefresh) return inFlightRefresh
  inFlightRefresh = refreshSessionRequest().finally(() => {
    inFlightRefresh = null
  })
  return inFlightRefresh
}

export async function restoreSession(): Promise<boolean> {
  try {
    await refreshSession()
    return true
  } catch {
    clearToken()
    return false
  }
}

export async function signOutSession(): Promise<void> {
  const token = accessToken
  try {
    await fetch('/auth/session/sign-out', {
      method: 'POST',
      credentials: 'include',
      headers: {
        ...(token ? { Authorization: `Bearer ${token}` } : {}),
      },
    })
  } finally {
    clearToken()
  }
}

export function rememberReturnPath(): void {
  const path = `${window.location.pathname}${window.location.search}${window.location.hash}`
  if (path !== '/' && !path.startsWith('/email/confirm/')) {
    sessionStorage.setItem(RETURN_TO_KEY, path)
  }
}

export function consumeReturnPath(): string | null {
  const path = sessionStorage.getItem(RETURN_TO_KEY)
  sessionStorage.removeItem(RETURN_TO_KEY)
  return path
}

export function clearAndRedirectToSignIn(): void {
  clearToken()
  rememberReturnPath()
  if (window.location.pathname !== '/') {
    window.location.assign('/')
  }
}

async function refreshSessionRequest(): Promise<string> {
  const response = await fetch('/auth/session/refresh', {
    method: 'POST',
    credentials: 'include',
  })
  if (!response.ok) {
    clearToken()
    throw new Error('Session refresh failed')
  }
  const data = await response.json() as WebSession
  setSession(data)
  return data.access_token
}

function nowSeconds(): number {
  return Math.floor(Date.now() / 1000)
}
