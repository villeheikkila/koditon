import { Link, useSearchParams } from 'react-router-dom'
import LiveSourceLink from '../components/LiveSourceLink'
import Nav from '../components/Nav'
import { useTransactionMatchCandidates, type TransactionMatchCandidate } from '../api/koditon'
import { addressLookupInputFromParams, appendAddressLookupParams, buildAddressLookupPath, sourceEntityPath, withAddressLookupContext, type AddressLookupInput } from '../lib/address-lookup'

export default function MatchesPage() {
  const [params] = useSearchParams()
  const transaction = params.get('transaction')?.trim() ?? ''
  const postal = params.get('postal')?.trim() ?? ''
  const lookup = addressLookupInputFromParams(params)
  const addressBackPath = buildAddressLookupPath(lookup) || '/address'
  const query = useTransactionMatchCandidates(
    { transaction: transaction || undefined, postal: transaction ? undefined : postal || undefined, limit: transaction ? 50 : 100 },
    { query: { enabled: Boolean(transaction || postal), placeholderData: previous => previous } },
  )
  const body = query.data?.status === 200 ? query.data.data : undefined
  const candidates = body?.candidates ?? []
  const linkedCount = candidates.filter(candidate => isLinkedReviewRow(candidate)).length
  const candidateCount = candidates.filter(candidate => !isLinkedReviewRow(candidate)).length
  const highCount = candidates.filter(candidate => candidate.confidence === 'high').length
  const ambiguousCount = candidates.filter(candidate => candidate.status === 'ambiguous').length
  return (
    <main className="matches-page">
      <Nav />
      <section className="matches-shell">
        <header className="matches-header">
          <div>
            <Link className="model-back" to={addressBackPath}>Address lookup</Link>
            <h1>Prices match review</h1>
            <p>{transaction ? `Transaction ${transaction}` : postal ? `Postal ${postal}` : 'Open a prices review link from address lookup.'}</p>
          </div>
          <div className="address-lookup-stats">
            <Metric label={transaction ? 'Matches' : 'Candidates'} value={candidates.length} />
            <Metric label="Linked" value={linkedCount} />
            <Metric label="Open" value={candidateCount} />
            <Metric label="High" value={highCount} />
            <Metric label="Ambiguous" value={ambiguousCount} />
          </div>
        </header>
        {!transaction && !postal && <div className="search-empty-hint"><strong>No review target</strong><span className="search-empty-sub">Open Review from an address lookup prices row.</span></div>}
        {query.isFetching && !body && <div className="loading-state">Loading prices candidates</div>}
        {query.isError && <div className="error-state">Prices candidates could not be loaded.</div>}
        {body && candidates.length === 0 && <div className="search-empty-hint"><strong>{transaction ? 'No matches found' : 'No candidates found'}</strong><span className="search-empty-sub">{transaction ? 'This prices transaction has no saved candidate or linked source rows.' : 'This postal code has no saved candidate rows.'}</span></div>}
        {candidates.length > 0 && (
          <div className={`matches-list${query.isFetching ? ' matches-list--loading' : ''}`}>
            {candidates.map(candidate => <MatchCard key={candidate.id} candidate={candidate} lookup={lookup} currentTransaction={transaction} />)}
          </div>
        )}
      </section>
    </main>
  )
}

function MatchCard({ candidate, lookup, currentTransaction }: { candidate: TransactionMatchCandidate; lookup: AddressLookupInput; currentTransaction: string }) {
  const detailPath = sourceEntityPath({ canonicalId: candidate.listing.canonical_id })
  const lookupPath = buildAddressLookupPath({ address: candidate.listing.street_address, city: candidate.listing.city, postal: candidate.listing.postal, source: candidate.listing.source_provider })
  const transactionPath = currentTransaction === candidate.transaction.id ? '' : transactionReviewPath(candidate, lookup)
  return (
    <article className="matches-card">
      <header className="matches-card-header">
        <div>
          <div className="search-card-badges">
            <span className="address-candidate-badge">{candidate.status}</span>
            <span className="search-badge search-badge--kind">{candidate.confidence}</span>
            <span className={`search-badge search-badge--${sourceBadgeClass(candidate.listing.source_provider)}`}>{sourceLabel(candidate.listing.source_provider)}</span>
            <span className="search-badge search-badge--kind">{candidate.listing.offering_id ? 'Aggregated' : 'Unaggregated'}</span>
          </div>
          <h2>{candidate.listing.headline || candidate.listing.street_address || candidate.listing.canonical_id}</h2>
          <p>{[candidate.listing.street_address, candidate.listing.city, candidate.listing.postal].filter(Boolean).join(' / ')}</p>
        </div>
        <div className="address-listing-price">
          <strong>{formatCurrency(candidate.listing.asking_price)}</strong>
          <span>{formatArea(candidate.listing.area_m2)}</span>
        </div>
      </header>
      <div className="matches-comparison">
        <section>
          <span className="matches-column-title">Source listing</span>
          <DetailRows rows={[
            ['Source', candidate.listing.source_provider],
            ['Rooms', candidate.listing.room_layout],
            ['Area', formatArea(candidate.listing.area_m2)],
            ['Price', formatCurrency(candidate.listing.asking_price)],
            ['Price/m2', formatPricePerM2(candidate.listing.price_per_m2)],
            ['Build year', candidate.listing.build_year],
            ['Floor', formatFloor(candidate.listing.floor_level, candidate.listing.total_floors)],
            ['Elevator', formatBool(candidate.listing.elevator)],
            ['Condition', candidate.listing.condition],
            ['Energy', candidate.listing.energy_label],
            ['Plot', formatPlot(candidate.listing.plot_ownership_raw, candidate.listing.plot_owned)],
            ['Seen', formatDateRange(candidate.listing.first_seen_at, candidate.listing.last_seen_at)],
          ]} />
        </section>
        <section>
          <span className="matches-column-title">Prices transaction</span>
          <DetailRows rows={[
            ['Description', candidate.transaction.description],
            ['Category', [candidate.transaction.category, candidate.transaction.type].filter(Boolean).join(' / ')],
            ['Area', formatArea(candidate.transaction.area_m2)],
            ['Price', formatCurrency(candidate.transaction.price)],
            ['Price/m2', formatPricePerM2(candidate.transaction.price_per_m2)],
            ['Build year', candidate.transaction.build_year],
            ['Floor', candidate.transaction.floor],
            ['Elevator', formatBool(candidate.transaction.elevator)],
            ['Condition', candidate.transaction.condition],
            ['Energy', candidate.transaction.energy_class],
            ['Plot', formatPlot(candidate.transaction.plot, candidate.transaction.plot_owned)],
            ['Period', candidate.transaction.period_identifier],
            ['Stored', formatDate(candidate.transaction.created_at)],
          ]} />
        </section>
        <section>
          <span className="matches-column-title">Match evidence</span>
          <DetailRows rows={[
            ['Score', candidate.score],
            ['Link', [formatLinkType(candidate.link_type), formatLinkMethod(candidate.link_method)].filter(Boolean).join(' / ')],
            ['Status', formatLinkStatus(candidate.status)],
            ['Confidence', candidate.confidence],
            ['Source ID', sourceIdentity(candidate.listing.source_provider, candidate.listing.native_id)],
            ['Delta', formatPercent(candidate.price_delta_percent)],
            ['Created', formatDate(candidate.created_at)],
            ['Reasons', formatReasons(candidate.reasons)],
          ]} />
        </section>
      </div>
      <div className="address-listing-actions">
        {lookupPath && <Link to={lookupPath}>Address lookup</Link>}
        {candidate.listing.offering_id && <Link to={withAddressLookupContext(`/target/offering/${candidate.listing.offering_id}`, lookup)}>Canonical offering</Link>}
        {detailPath && <Link to={withAddressLookupContext(detailPath, lookup)}>Source detail</Link>}
        {transactionPath && <Link to={transactionPath}>Review transaction</Link>}
        <LiveSourceLink available={candidate.listing.external_url_available} url={candidate.listing.url} />
      </div>
    </article>
  )
}

function isLinkedReviewRow(candidate: TransactionMatchCandidate) {
  return candidate.link_type === 'direct' || candidate.link_type === 'offering' || candidate.link_type === 'source_record' || candidate.status === 'linked' || candidate.status === 'auto_linked'
}

function formatLinkMethod(value?: string) {
  return value?.trim().replaceAll('_', ' ') ?? ''
}

function formatLinkType(value?: string) {
  const normalized = value?.trim()
  if (normalized === 'direct') return 'Direct source'
  if (normalized === 'offering') return 'Offering'
  if (normalized === 'source_record') return 'Offering source'
  if (normalized === 'candidate') return 'Candidate'
  return formatLinkMethod(normalized)
}

function formatLinkStatus(value?: string) {
  const normalized = value?.trim()
  if (normalized === 'auto_linked') return 'Auto linked'
  if (normalized === 'linked') return 'Linked'
  if (normalized === 'candidate') return 'Candidate'
  if (normalized === 'ambiguous') return 'Ambiguous'
  return formatLinkMethod(normalized)
}

function transactionReviewPath(candidate: TransactionMatchCandidate, lookup: AddressLookupInput) {
  const params = new URLSearchParams()
  if (candidate.listing.postal) params.set('postal', candidate.listing.postal)
  params.set('transaction', candidate.transaction.id)
  appendAddressLookupParams(params, lookup)
  return `/matches?${params.toString()}`
}

function Metric({ label, value }: { label: string; value: number }) {
  return (
    <div className="model-metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function DetailRows({ rows }: { rows: Array<[string, string | number | undefined]> }) {
  const visible = rows.filter(([, value]) => value != null && value !== '')
  if (visible.length === 0) return <p className="model-empty">No values.</p>
  return (
    <div className="detail-rows">
      {visible.map(([label, value]) => (
        <div className="detail-row" key={label}>
          <span>{label}</span>
          <strong>{value}</strong>
        </div>
      ))}
    </div>
  )
}

function formatCurrency(value?: number) {
  if (value == null) return 'n/a'
  return `${new Intl.NumberFormat('fi-FI').format(value)} EUR`
}

function formatArea(value?: number) {
  if (value == null) return ''
  return `${new Intl.NumberFormat('fi-FI', { maximumFractionDigits: 1 }).format(value)} m2`
}

function formatPricePerM2(value?: number) {
  if (value == null) return ''
  return `${new Intl.NumberFormat('fi-FI').format(value)} EUR/m2`
}

function formatFloor(level?: number, total?: number) {
  if (level == null && total == null) return ''
  return [level, total].filter(value => value != null).join(' / ')
}

function formatBool(value?: boolean) {
  if (value == null) return ''
  return value ? 'Yes' : 'No'
}

function formatPlot(label?: string, owned?: boolean) {
  return [label, formatBool(owned)].filter(Boolean).join(' / ')
}

function formatDateRange(start?: string, end?: string) {
  return [formatDate(start), formatDate(end)].filter(Boolean).join(' - ')
}

function formatPercent(value?: number) {
  if (value == null) return ''
  const percent = Math.abs(value) <= 1 ? value * 100 : value
  return `${new Intl.NumberFormat('fi-FI', { maximumFractionDigits: 1 }).format(percent)}%`
}

function formatDate(value?: string) {
  if (!value) return ''
  return new Intl.DateTimeFormat('fi-FI').format(new Date(value))
}

function formatReasons(value: unknown) {
  if (!isRecord(value)) return ''
  const evidence = [
    reasonString(value, 'matched_by') ? `Matched by ${reasonString(value, 'matched_by')}` : '',
    reasonString(value, 'postal') ? `Postal ${reasonString(value, 'postal')}` : sourceTargetReason(value, 'postal', 'Postal'),
    sourceTargetReason(value, 'address', 'Address'),
    sourceTargetReason(value, 'street_name', 'Street'),
    sourceTargetReason(value, 'unit_match_key', 'Unit key'),
    reasonPair(value, 'area', 'Area'),
    reasonLayout(value),
    reasonPair(value, 'build_year', 'Build year'),
    reasonPair(value, 'floor_level', 'Floor'),
    reasonPair(value, 'total_floors', 'Floors'),
    reasonPair(value, 'condition', 'Condition'),
    reasonPair(value, 'energy', 'Energy'),
    reasonString(value, 'property_type') ? `Type ${reasonString(value, 'property_type')}` : '',
    reasonBoolean(value, 'layout_prefix') ? 'Layout prefix matched' : '',
    reasonScore(value),
  ].filter(Boolean)
  if (evidence.length > 0) return evidence.slice(0, 8).join(' / ')
  return Object.entries(value).slice(0, 5).map(([key, entry]) => `${key}: ${formatReasonValue(entry)}`).join(' / ')
}

function formatReasonValue(value: unknown): string {
  if (value == null) return ''
  if (typeof value === 'string' || typeof value === 'number' || typeof value === 'boolean') return String(value)
  if (Array.isArray(value)) return value.map(formatReasonValue).filter(Boolean).join(', ')
  if (typeof value === 'object') return Object.entries(value).map(([key, entry]) => `${key} ${formatReasonValue(entry)}`).join(', ')
  return ''
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function reasonString(reasons: Record<string, unknown>, key: string) {
  const value = reasons[key]
  return typeof value === 'string' && value.trim() ? value.trim() : ''
}

function reasonBoolean(reasons: Record<string, unknown>, key: string) {
  return reasons[key] === true
}

function reasonPair(reasons: Record<string, unknown>, key: string, label: string) {
  const value = reasons[key]
  if (!isRecord(value)) return typeof value === 'string' || typeof value === 'number' ? `${label} ${value}` : ''
  const listing = reasonScalar(value.listing)
  const transaction = reasonScalar(value.transaction)
  if (!listing && !transaction) return sourceTargetReason(reasons, key, label)
  return `${label} ${listing || 'n/a'} / ${transaction || 'n/a'}`
}

function sourceTargetReason(reasons: Record<string, unknown>, key: string, label: string) {
  const value = reasons[key]
  if (!isRecord(value)) return ''
  const source = reasonScalar(value.source)
  const target = reasonScalar(value.target)
  if (!source && !target) return ''
  return `${label} ${source || 'n/a'} / ${target || 'n/a'}`
}

function reasonLayout(reasons: Record<string, unknown>) {
  const value = reasons.layout
  if (!isRecord(value)) return reasonPair(reasons, 'layout', 'Layout')
  const listing = reasonScalar(value.listing)
  const transaction = reasonScalar(value.transaction)
  const code = reasonScalar(value.code)
  if (listing || transaction || code) return `Layout${code ? ` ${code}` : ''} ${listing || 'n/a'} / ${transaction || 'n/a'}`
  return sourceTargetReason(reasons, 'layout', 'Layout')
}

function reasonScore(reasons: Record<string, unknown>) {
  const value = reasons.score
  if (!isRecord(value)) return ''
  const total = reasonScalar(value.total)
  if (total) return `Score total ${total}`
  const parts = ['address', 'unit', 'building', 'street', 'street_area_layout', 'street_area_floor_price', 'area', 'layout', 'floor', 'build_year', 'elevator', 'plot', 'energy', 'condition', 'price', 'temporal', 'transaction', 'postal', 'city'].flatMap(key => {
    const score = positiveReasonScore(value[key])
    return score ? [`${key.replaceAll('_', ' ')} ${score}`] : []
  })
  return parts.length > 0 ? `Score ${parts.slice(0, 4).join(', ')}` : ''
}

function positiveReasonScore(value: unknown) {
  if (typeof value === 'number' && Number.isFinite(value) && value > 0) return String(value)
  if (typeof value === 'string' && value.trim() && value.trim() !== '0') return value.trim()
  return ''
}

function reasonScalar(value: unknown) {
  if (typeof value === 'string' && value.trim()) return value.trim()
  if (typeof value === 'number' && Number.isFinite(value)) return String(value)
  if (typeof value === 'boolean') return String(value)
  return ''
}

function sourceLabel(source?: string) {
  if (source === 'shortcut') return 'Shortcut'
  if (source === 'frontdoor') return 'Frontdoor'
  return source || 'Source'
}

function sourceIdentity(source?: string, nativeID?: string) {
  return [sourceLabel(source), nativeID].filter(Boolean).join(' ')
}

function sourceBadgeClass(source?: string) {
  if (source === 'shortcut') return 'shortcut'
  if (source === 'frontdoor') return 'frontdoor'
  return 'kind'
}
