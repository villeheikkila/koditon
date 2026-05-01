import { useMemo, useState } from 'react'
import { Link } from 'react-router-dom'
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

export default function MatchesPage() {
  const [selectedPostal, setSelectedPostal] = useState('')
  const [status, setStatus] = useState('')
  const postalsQuery = useSaleListingsTransactionMatchPostals({ limit: 250 }, { query: { staleTime: 30_000 } })
  const postals = useMemo(
    () => postalsQuery.data?.status === 200 ? (postalsQuery.data.data.postals ?? []) : [],
    [postalsQuery.data],
  )
  const activePostal = selectedPostal || postals[0]?.postal || ''
  const candidatesQuery = useSaleListingsTransactionMatchCandidates({
    postal: activePostal || undefined,
    status: status || undefined,
    limit: LIMIT,
  }, {
    query: { enabled: !!activePostal, staleTime: 15_000 },
  })
  const candidates = useMemo(
    () => candidatesQuery.data?.status === 200 ? (candidatesQuery.data.data.candidates ?? []) : [],
    [candidatesQuery.data],
  )
  const activeSummary = postals.find(p => p.postal === activePostal)
  const highCount = candidates.filter(c => c.confidence === 'high').length
  const ambiguousCount = candidates.filter(c => c.status === 'ambiguous').length
  return (
    <div className="matches-layout">
      <Nav actions={<span className="search-total">{postals.length.toLocaleString('fi-FI')} postal codes</span>} />
      <div className="matches-body">
        <aside className="matches-sidebar">
          <div className="matches-sidebar-head">
            <div className="sidebar-label">Potential Matches</div>
            <select className="filter-select" value={status} onChange={e => setStatus(e.target.value)}>
              <option value="">All unresolved</option>
              <option value="ambiguous">Ambiguous</option>
              <option value="candidate">Candidate</option>
            </select>
          </div>
          <div className="matches-postal-list">
            {postalsQuery.isPending ? (
              <div className="loading-state"><div className="spinner" /></div>
            ) : postals.length === 0 ? (
              <div className="empty-state compact">No potential matches</div>
            ) : (
              postals.map(postal => (
                <PostalButton
                  key={postal.postal}
                  postal={postal}
                  active={postal.postal === activePostal}
                  onClick={() => setSelectedPostal(postal.postal)}
                />
              ))
            )}
          </div>
        </aside>
        <main className="matches-main">
          <div className="matches-header">
            <div>
              <div className="matches-kicker">Review Queue</div>
              <h1 className="matches-title">{activePostal || 'No postal code selected'}</h1>
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
            {candidates.length > 0 && <span className="section-count">{candidates.length} rows</span>}
          </div>
          <div className="matches-table-shell">
            {!activePostal ? (
              <div className="empty-state">No postal code has potential matches</div>
            ) : candidatesQuery.isPending ? (
              <div className="loading-state"><div className="spinner" /></div>
            ) : candidatesQuery.isError ? (
              <div className="error-state">Failed to load potential matches</div>
            ) : candidates.length === 0 ? (
              <div className="empty-state">No candidates for this filter</div>
            ) : (
              <CandidateTable candidates={candidates} />
            )}
          </div>
        </main>
      </div>
    </div>
  )
}

function PostalButton({ postal, active, onClick }: { postal: TransactionMatchPostalSummary; active: boolean; onClick: () => void }) {
  return (
    <button className={`matches-postal-btn${active ? ' matches-postal-btn--active' : ''}`} onClick={onClick}>
      <span className="matches-postal-code">{postal.postal}</span>
      <span className="matches-postal-count">{fmt(postal.candidate_count)}</span>
      <span className="matches-postal-sub">{fmt(postal.listing_count)} listings · {fmt(postal.ambiguous_count)} ambiguous</span>
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

function CandidateTable({ candidates }: { candidates: TransactionMatchCandidate[] }) {
  return (
    <div className="table-wrap matches-table-wrap">
      <table>
        <thead>
          <tr>
            <th>Score</th>
            <th>Listing</th>
            <th>Transaction</th>
            <th className="right">Ask</th>
            <th className="right">Sold</th>
            <th className="right">Delta</th>
            <th>Facts</th>
          </tr>
        </thead>
        <tbody>
          {candidates.map(candidate => (
            <CandidateRow key={candidate.id} candidate={candidate} />
          ))}
        </tbody>
      </table>
    </div>
  )
}

function CandidateRow({ candidate }: { candidate: TransactionMatchCandidate }) {
  const listing = candidate.listing
  const transaction = candidate.transaction
  const facts = [
    listing.room_layout || transaction.description,
    fmtArea(listing.area_m2),
    listing.build_year || transaction.build_year,
    listing.floor_level && listing.total_floors ? `${listing.floor_level}/${listing.total_floors}` : transaction.floor,
    transaction.energy_class,
  ].filter(Boolean).join(' · ')
  return (
    <tr>
      <td>
        <div className="match-score">
          <span>{candidate.score}</span>
          <small>{candidate.confidence}</small>
        </div>
      </td>
      <td className="matches-entity-cell">
        <Link className="matches-listing-link" to={`/listing/${encodeURIComponent(listing.id)}`}>
          {listing.street_address || listing.headline || listing.id}
        </Link>
        <div className="matches-entity-sub">{listing.source_provider} · {listing.postal} · last seen {listing.last_seen_at || '—'}</div>
      </td>
      <td className="matches-entity-cell">
        <div className="matches-transaction-main">{transaction.description || transaction.id}</div>
        <div className="matches-entity-sub">{transaction.type} · {transaction.period_identifier || transaction.created_at}</div>
      </td>
      <td className="right mono">{fmtPrice(listing.asking_price)}</td>
      <td className="right mono">{fmtPrice(transaction.price)}</td>
      <td className="right mono">{fmtDelta(candidate.price_delta_percent)}</td>
      <td className="muted matches-facts">{facts || '—'}</td>
    </tr>
  )
}
