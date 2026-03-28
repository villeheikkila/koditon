export const customInstance = async <T>(
  url: string,
  options?: RequestInit,
): Promise<T> => {
  const token = localStorage.getItem('koditon_access_token')
  const response = await fetch(url, {
    ...options,
    headers: {
      'Content-Type': 'application/json',
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
      ...options?.headers,
    },
  })

  let data: unknown
  const contentType = response.headers.get('content-type') ?? ''
  if (contentType.includes('application/json') || contentType.includes('application/problem+json')) {
    data = await response.json()
  } else {
    data = await response.text()
  }

  return { data, status: response.status, headers: response.headers } as T
}

export type ErrorType<E> = E
export type BodyType<B> = B
