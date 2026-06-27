import type { TargetOverviewField } from '../api/koditon'

export type AddressLookupInput = {
  address?: string | null
  city?: string | null
  postal?: string | null
}

export type SourceEntityInput = {
  canonicalId?: string | null
  kind?: string | null
}

export function buildAddressLookupPath(input: AddressLookupInput) {
  const address = input.address?.trim()
  if (!address) return ''
  const params = new URLSearchParams()
  params.set('address', address)
  if (input.city?.trim()) params.set('city', input.city.trim())
  if (input.postal?.trim()) params.set('postal', input.postal.trim())
  return `/address?${params.toString()}`
}

export function sourceEntityPath(input: SourceEntityInput) {
  const canonicalId = input.canonicalId?.trim()
  if (!canonicalId) return ''
  const kind = input.kind?.trim().toLowerCase() || canonicalKind(canonicalId)
  const route = kind === 'announcement' || kind === 'building' ? 'housing-company' : 'listing'
  return `/${route}/${encodeURIComponent(canonicalId)}`
}

export function looksLikeEntityInput(value: string) {
  const trimmed = value.trim()
  return /^https?:\/\//i.test(trimmed) || /^[a-z]+:[a-z_-]+:.+$/i.test(trimmed)
}

export function addressLookupPathFromOverviewFields(fields: TargetOverviewField[]) {
  return buildAddressLookupPath({
    address: overviewFieldValue(fields, 'address', 'street address', 'street_address', 'location.street_address'),
    city: overviewFieldValue(fields, 'city', 'municipality', 'location.city'),
    postal: overviewFieldValue(fields, 'postal', 'postal code', 'postal_code', 'location.postal'),
  })
}

function overviewFieldValue(fields: TargetOverviewField[], ...labels: string[]) {
  const wanted = new Set(labels.map(normalizeFieldLabel))
  const field = fields.find(item => wanted.has(normalizeFieldLabel(item.label)))
  return field?.value.trim() ?? ''
}

function normalizeFieldLabel(value: string) {
  return value.trim().toLowerCase().replaceAll('_', ' ')
}

function canonicalKind(canonicalId: string) {
  const parts = canonicalId.split(':')
  return parts.length >= 3 ? parts[1]?.trim().toLowerCase() : ''
}
