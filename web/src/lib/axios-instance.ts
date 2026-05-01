import { clearAndRedirectToSignIn, getFreshTokenOrNull, refreshSession } from './auth-store'

export const customInstance = async <T>(
  url: string,
  options?: RequestInit,
): Promise<T> => {
  const token = getFreshTokenOrNull()
  const response = await fetch(url, {
    ...options,
    credentials: 'include',
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  })
  if (response.status === 401 && !isSessionRefresh(url)) {
    const refreshedToken = await refreshSession().catch(() => null)
    if (refreshedToken) {
      const retryResponse = await fetch(url, {
        ...options,
        credentials: 'include',
        headers: {
          'Content-Type': 'application/json',
          ...options?.headers,
          Authorization: `Bearer ${refreshedToken}`,
        },
      })
      return parseResponse<T>(retryResponse)
    }
    clearAndRedirectToSignIn()
  }
  return parseResponse<T>(response)
}

async function parseResponse<T>(response: Response): Promise<T> {
  let data: unknown
  const contentType = response.headers.get('content-type') ?? ''
  if (contentType.includes('application/json') || contentType.includes('application/problem+json')) {
    data = await response.json()
  } else {
    data = await response.text()
  }
  if (!response.ok) {
    const detail = errorDetail(data) ?? `Request failed: ${response.status}`
    throw new Error(detail)
  }
  return { data, status: response.status, headers: response.headers } as T
}

function isSessionRefresh(url: string): boolean {
  return url.includes('/auth/session/refresh')
}

function errorDetail(data: unknown): string | null {
  if (typeof data !== 'object' || data == null) return null
  if ('detail' in data && data.detail != null) return String(data.detail)
  if ('error_description' in data && data.error_description != null) return String(data.error_description)
  return null
}

export type ErrorType<E> = E
export type BodyType<B> = B
