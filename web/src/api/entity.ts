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
  url?: string
  last_seen_at?: string

  // Location
  street_address?: string
  city?: string
  postal?: string

  // Pricing
  asking_price?: number
  debt_free_price?: number
  debt_share_amount?: number
  price_per_m2?: number

  // Property
  area_m2?: number
  room_layout?: string
  rooms_count?: number
  floor_level?: number
  total_floors?: number
  build_year?: number
  condition?: string
  energy_class?: string
  plot_type?: string
  elevator?: boolean
  sauna?: boolean

  // Monthly charges
  maintenance_charge_monthly?: number
  total_charge_monthly?: number
  water_charge?: number

  // Text
  description_text?: string
  availability_text?: string
  renovations_done_text?: string
  renovations_planned_text?: string
  additional_info_text?: string
  charges_text?: string

  // Extras
  canonical_extra?: DetailField[]
  source_specific?: DetailField[]
  related?: DetailField[]
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
