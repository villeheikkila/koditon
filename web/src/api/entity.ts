import { useQuery } from '@tanstack/react-query'
import { getToken } from '../lib/auth'

export interface DetailField {
  label: string
  value: string
}

export interface EntityDetail {
  canonical_id: string
  source: string
  kind: string
  native_id: string
  headline: string
  address?: string
  city?: string
  postal?: string
  price?: number
  area?: number
  room_layout?: string
  url?: string
  last_seen_at?: string
  canonical_extra?: DetailField[]
  source_specific?: DetailField[]
  related?: DetailField[]
  raw_json?: string
}

async function fetchEntityDetail(id: string): Promise<EntityDetail> {
  const token = getToken()
  const res = await fetch('/api/v1/entity?' + new URLSearchParams({ id }), {
    headers: token ? { Authorization: `Bearer ${token}` } : {},
  })
  if (!res.ok) {
    const body = await res.json().catch(() => ({}))
    throw new Error(body?.detail ?? `Entity not found (${res.status})`)
  }
  return res.json()
}

export function useEntityDetail(id: string) {
  return useQuery({
    queryKey: ['entity', id],
    queryFn: () => fetchEntityDetail(id),
    enabled: !!id,
    retry: false,
  })
}
