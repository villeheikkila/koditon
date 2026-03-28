import { useQuery, keepPreviousData } from '@tanstack/react-query'
import { getToken } from '../lib/auth'

export interface SearchRow {
  canonical_id: string
  source: string
  kind: string
  headline?: string
  address?: string
  city?: string
  postal?: string
  price?: number
  area?: number
  room_layout?: string
  url?: string
  last_seen_at?: string
}

export interface SearchResult {
  rows: SearchRow[]
  total: number
  page: number
  page_size: number
}

export interface SearchParams {
  q?: string
  source?: string
  kind?: string
  city?: string
  postal?: string
  min_price?: number
  max_price?: number
  min_area?: number
  max_area?: number
  sort?: string
  page?: number
  page_size?: number
}

export async function fetchSearch(params: SearchParams): Promise<SearchResult> {
  const qs = new URLSearchParams()
  if (params.q)         qs.set('q', params.q)
  if (params.source)    qs.set('source', params.source)
  if (params.kind)      qs.set('kind', params.kind)
  if (params.city)      qs.set('city', params.city)
  if (params.postal)    qs.set('postal', params.postal)
  if (params.min_price != null) qs.set('min_price', String(params.min_price))
  if (params.max_price != null) qs.set('max_price', String(params.max_price))
  if (params.min_area != null)  qs.set('min_area', String(params.min_area))
  if (params.max_area != null)  qs.set('max_area', String(params.max_area))
  if (params.sort)      qs.set('sort', params.sort)
  if (params.page != null)      qs.set('page', String(params.page))
  if (params.page_size != null) qs.set('page_size', String(params.page_size))

  const token = getToken()
  const headers: Record<string, string> = {}
  if (token) headers['Authorization'] = `Bearer ${token}`

  const res = await fetch(`/api/v1/search?${qs}`, { headers })
  if (!res.ok) throw new Error(`Search failed: ${res.status}`)
  return res.json()
}

export function useSearch(params: SearchParams, enabled = true) {
  return useQuery({
    queryKey: ['search', params],
    queryFn: () => fetchSearch(params),
    enabled,
    placeholderData: keepPreviousData,
    staleTime: 30_000,
  })
}
