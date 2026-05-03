import { type ReactNode, useMemo, useState } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import Nav from '../components/Nav'
import {
  useSaleListingsTransactionMatchCandidates,
  useSaleListingsTransactionMatchPostals,
  type TransactionMatchCandidate,
  type TransactionMatchPostalSummary,
} from '../api/koditon'

const LIMIT = 200

function fmt(n: number) {
  return new Intl.NumberFormat('fi-FI').format(n)
}

function fmtPrice(n?: number) {
  return n == null ? '—' : `${fmt(n)} €`
}

function fmtArea(n?: number) {
  return n == null ? '—' : `${n.toFixed(1)} m²`
}

function fmtDelta(n?: number) {
  if (n == null) return '—'
  return `${(n * 100).toFixed(1)}%`
}

function fmtBool(value?: boolean) {
  if (value == null) return undefined
  return value ? 'yes' : 'no'
}

function fmtPlotOwned(value?: boolean, raw?: string) {
  if (value != null) return value ? 'owned' : 'rented'
  return raw
}

function fmtNormalized(raw?: string, code?: string) {
  if (!code) return raw
  return <span className="match-text-hit">{code}</span>
}

function commonPrefixLength(a?: string, b?: string) {
  if (!a || !b) return 0
  const max = Math.min(a.length, b.length)
  let index = 0
  while (index < max && a[index]?.toLocaleLowerCase() === b[index]?.toLocaleLowerCase()) index += 1
  return index
}

function highlightedPrefix(value?: string, matchWith?: string) {
  if (!value) return undefined
  const length = commonPrefixLength(value, matchWith)
  if (length < 2) return value
  return (
    <>
      <span className="match-text-hit">{value.slice(0, length)}</span>{value.slice(length)}
    </>
  )
}

export default function MatchesPage() {
  const [searchParams, setSearchParams] = useSearchParams()
  const [selectedPostal, setSelectedPostal] = useState(() => searchParams.get('postal') ?? '')
  const [status, setStatus] = useState(() => searchParams.get('status') ?? '')
  const selectedTransaction = searchParams.get('transaction') ?? ''
  const [postalSearch, setPostalSearch] = useState('')
  const postalsQuery = useSaleListingsTransactionMatchPostals({ limit: 250 }, { query: { staleTime: 30_000 } })
  const postals = useMemo(
    () => postalsQuery.data?.status === 200 ? (postalsQuery.data.data.postals ?? []) : [],
    [postalsQuery.data],
  )
  const filteredPostals = useMemo(() => {
    const query = postalSearch.trim().toLowerCase()
    if (!query) return postals
    return postals.filter(postal => [
      postal.postal,
      postal.name_fi,
      postal.municipality_name,
    ].some(value => value?.toLowerCase().includes(query)))
  }, [postalSearch, postals])
  const activePostal = selectedPostal || postals[0]?.postal || ''
  const candidatesQuery = useSaleListingsTransactionMatchCandidates({
    postal: activePostal || undefined,
    status: status || undefined,
    transaction: selectedTransaction || undefined,
    limit: LIMIT,
  }, {
    query: { enabled: !!activePostal, staleTime: 15_000 },
  })
  const candidates = useMemo(
    () => candidatesQuery.data?.status === 200 ? (candidatesQuery.data.data.candidates ?? []) : [],
    [candidatesQuery.data],
  )
  const visibleCandidates = candidates
  const activeSummary = postals.find(p => p.postal === activePostal)
  const highCount = visibleCandidates.filter(c => c.confidence === 'high').length
  const ambiguousCount = visibleCandidates.filter(c => c.status === 'ambiguous').length
  const activeTitle = [activePostal, activeSummary?.name_fi || activeSummary?.municipality_name].filter(Boolean).join(' ')
  function selectPostal(postal: string) {
    setSelectedPostal(postal)
    const next = new URLSearchParams(searchParams)
    if (postal) next.set('postal', postal)
    else next.delete('postal')
    next.delete('transaction')
    setSearchParams(next, { replace: true })
  }
  function selectStatus(nextStatus: string) {
    setStatus(nextStatus)
    const next = new URLSearchParams(searchParams)
    if (nextStatus) next.set('status', nextStatus)
    else next.delete('status')
    setSearchParams(next, { replace: true })
  }
  function clearTransactionFocus() {
    const next = new URLSearchParams(searchParams)
    next.delete('transaction')
    setSearchParams(next, { replace: true })
  }
  return (
    <div className="matches-layout">
      <Nav actions={<span className="search-total">{postals.length.toLocaleString('fi-FI')} postal codes</span>} />
      <div className="matches-body">
        <aside className="matches-sidebar">
          <div className="matches-sidebar-head">
            <div className="sidebar-label">Potential Matches</div>
            <select className="filter-select" value={status} onChange={e => selectStatus(e.target.value)}>
              <option value="">All unresolved</option>
              <option value="ambiguous">Ambiguous</option>
              <option value="candidate">Candidate</option>
            </select>
            <input
              className="filter-input matches-postal-search"
              placeholder="Search postal or area"
              value={postalSearch}
              onChange={e => setPostalSearch(e.target.value)}
            />
          </div>
          <div className="matches-postal-list">
            {postalsQuery.isPending ? (
              <div className="loading-state"><div className="spinner" /></div>
            ) : postals.length === 0 ? (
              <div className="empty-state compact">No potential matches</div>
            ) : filteredPostals.length === 0 ? (
              <div className="empty-state compact">No postal codes match the search</div>
            ) : (
              filteredPostals.map(postal => (
                <PostalButton
                  key={postal.postal}
                  postal={postal}
                  active={postal.postal === activePostal}
                  onClick={() => selectPostal(postal.postal)}
                />
              ))
            )}
          </div>
        </aside>
        <main className="matches-main">
          <div className="matches-header">
            <div>
              <div className="matches-kicker">Review Queue</div>
              <h1 className="matches-title">{activeTitle || 'No postal code selected'}</h1>
            </div>
            {activeSummary && (
              <div className="matches-header-meta">
                <span>{fmt(activeSummary.candidate_count)} candidates</span>
                <span>{fmt(activeSummary.listing_count)} listings</span>
                <span>{fmt(activeSummary.transaction_count)} transactions</span>
              </div>
            )}
          </div>
          <div className="stats-grid matches-stats">
            <Stat label="Rows" value={fmt(candidates.length)} sub={`showing up to ${LIMIT}`} />
            <Stat label="High confidence" value={fmt(highCount)} sub="current filter" />
            <Stat label="Ambiguous" value={fmt(ambiguousCount)} sub="needs decision" />
            <Stat label="Postal queue" value={activeSummary ? fmt(activeSummary.candidate_count) : '—'} sub="all unresolved" />
          </div>
          <div className="section-header">
            <span className="section-title">Candidates</span>
            {visibleCandidates.length > 0 && <span className="section-count">{visibleCandidates.length} rows</span>}
          </div>
          {selectedTransaction && (
            <div className="match-focus-banner">
              <span>Showing likely matches for transaction {selectedTransaction}</span>
              <button type="button" onClick={clearTransactionFocus}>Show all in postal code</button>
            </div>
          )}
          <div className="matches-table-shell">
            {!activePostal ? (
              <div className="empty-state">No postal code has potential matches</div>
            ) : candidatesQuery.isPending ? (
              <div className="loading-state"><div className="spinner" /></div>
            ) : candidatesQuery.isError ? (
              <div className="error-state">Failed to load potential matches</div>
            ) : visibleCandidates.length === 0 ? (
              <div className="empty-state">No candidates for this filter</div>
            ) : (
              <CandidateCards candidates={visibleCandidates} />
            )}
          </div>
        </main>
      </div>
    </div>
  )
}

function PostalButton({ postal, active, onClick }: { postal: TransactionMatchPostalSummary; active: boolean; onClick: () => void }) {
  const area = [postal.name_fi, postal.municipality_name].filter(Boolean).join(', ')
  return (
    <button className={`matches-postal-btn${active ? ' matches-postal-btn--active' : ''}`} onClick={onClick}>
      <span className="matches-postal-code">{postal.postal}</span>
      <span className="matches-postal-count">{fmt(postal.candidate_count)}</span>
      {area && <span className="matches-postal-area">{area}</span>}
      <span className="matches-postal-sub">{fmt(postal.listing_count)} listings · {fmt(postal.transaction_count)} transactions · {fmt(postal.ambiguous_count)} ambiguous</span>
    </button>
  )
}

function Stat({ label, value, sub }: { label: string; value: string; sub: string }) {
  return (
    <div className="stat-card">
      <div className="stat-label">{label}</div>
      <div className="stat-value accent">{value}</div>
      <div className="stat-sub">{sub}</div>
    </div>
  )
}

function CandidateCards({ candidates }: { candidates: TransactionMatchCandidate[] }) {
  return (
    <div className="matches-card-list">
      {candidates.map(candidate => (
        <CandidateCard key={candidate.id} candidate={candidate} />
      ))}
    </div>
  )
}

function CandidateCard({ candidate }: { candidate: TransactionMatchCandidate }) {
  const listing = candidate.listing
  const transaction = candidate.transaction
  const listingTitle = listing.street_address || listing.headline || listing.id
  const transactionTitle = transaction.description || transaction.id
  const transactionLayout = transaction.description
  const rows: [string, ReactNode | undefined, ReactNode | undefined][] = [
    ['Layout', highlightedPrefix(listing.room_layout, transactionLayout), highlightedPrefix(transactionLayout, listing.room_layout)],
    ['Condition', fmtNormalized(listing.condition, listing.condition_match_code), fmtNormalized(transaction.condition, transaction.condition_match_code)],
    ['Area', fmtArea(listing.area_m2), fmtArea(transaction.area_m2)],
    ['Build', listing.build_year, transaction.build_year || undefined],
    ['Floor', listing.floor_level && listing.total_floors ? `${listing.floor_level}/${listing.total_floors}` : listing.floor_level, transaction.floor],
    ['€/m²', listing.price_per_m2 == null ? undefined : fmt(Math.round(listing.price_per_m2)), fmt(transaction.price_per_m2)],
    ['Elevator', fmtBool(listing.elevator), fmtBool(transaction.elevator)],
    ['Energy', listing.energy_match_code || listing.energy_label, transaction.energy_match_code || transaction.energy_class],
    ['Plot', fmtPlotOwned(listing.plot_owned, listing.plot_ownership_raw), fmtPlotOwned(transaction.plot_owned, transaction.plot)],
    ['Appeared', listing.first_seen_at, transaction.created_at],
    ['Unlisted', listing.last_seen_at, undefined],
  ]
  return (
    <article className="match-card">
      <div className="match-card-head">
        <div className="match-card-score">
          <span>{candidate.score}</span>
          <small>{candidate.confidence} · {candidate.status}</small>
        </div>
        <div className="match-card-prices">
          <span>Ask {fmtPrice(listing.asking_price)}</span>
          <span>Sold {fmtPrice(transaction.price)}</span>
          <strong>{fmtDelta(candidate.price_delta_percent)}</strong>
        </div>
      </div>
      <div className="match-card-grid">
        <section className="match-card-side">
          <div className="match-card-label">Listing</div>
          <Link className="matches-listing-link match-card-title" to={`/listing/${encodeURIComponent(listing.id)}`}>{listingTitle}</Link>
          <div className="matches-entity-sub">{listing.source_provider} · {listing.postal || '—'} · {listing.city || '—'}</div>
          <FactGrid items={rows.map(([label, value]) => [label, value])} />
        </section>
        <section className="match-card-side">
          <div className="match-card-label">Transaction</div>
          <div className="matches-transaction-main match-card-title">{transactionTitle}</div>
          <div className="matches-entity-sub">{transaction.type || '—'} · {transaction.category || '—'} · {transaction.period_identifier || transaction.created_at || '—'}</div>
          <FactGrid items={rows.map(([label, , value]) => [label, value])} />
        </section>
      </div>
    </article>
  )
}

function FactGrid({ items }: { items: [string, ReactNode | undefined][] }) {
  return (
    <dl className="match-fact-grid">
      {items.map(([label, value]) => (
        <div className="match-fact" key={label}>
          <dt>{label}</dt>
          <dd>{value == null || value === '' ? '—' : value}</dd>
        </div>
      ))}
    </dl>
  )
}
