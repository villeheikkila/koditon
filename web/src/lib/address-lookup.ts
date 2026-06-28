import type { TargetOverviewField } from '../api/koditon'

export type AddressLookupInput = {
  address?: string | null
  city?: string | null
  postal?: string | null
  source?: string | null
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
  const source = input.source?.trim()
  if (source && source !== 'all') params.set('source', source)
  return `/address?${params.toString()}`
}

export function addressLookupInputFromParams(params: URLSearchParams): AddressLookupInput {
  return {
    address: params.get('lookup_address'),
    city: params.get('lookup_city'),
    postal: params.get('lookup_postal'),
    source: params.get('lookup_source'),
  }
}

export function appendAddressLookupParams(params: URLSearchParams, lookup?: AddressLookupInput) {
  if (lookup?.address?.trim()) params.set('lookup_address', lookup.address.trim())
  if (lookup?.city?.trim()) params.set('lookup_city', lookup.city.trim())
  if (lookup?.postal?.trim()) params.set('lookup_postal', lookup.postal.trim())
  if (lookup?.source?.trim() && lookup.source !== 'all') params.set('lookup_source', lookup.source.trim())
}

export function withAddressLookupContext(path: string, lookup?: AddressLookupInput) {
  if (!path) return ''
  const [base, query = ''] = path.split('?', 2)
  const params = new URLSearchParams(query)
  appendAddressLookupParams(params, lookup)
  const queryString = params.toString()
  return queryString ? `${base}?${queryString}` : base
}

export function sourceEntityPath(input: SourceEntityInput) {
  const canonicalId = input.canonicalId?.trim()
  if (!canonicalId) return ''
  const kind = input.kind?.trim().toLowerCase() || canonicalKind(canonicalId)
  const route = sourceEntityRoute(kind)
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

function sourceEntityRoute(kind: string) {
  if (kind === 'announcement' || kind === 'building') return 'housing-company'
  if (kind === 'rental') return 'rental'
  return 'listing'
}
