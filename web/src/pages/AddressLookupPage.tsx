import { useMemo, useState, type FormEvent } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import LiveSourceLink from '../components/LiveSourceLink'
import Nav from '../components/Nav'
import { useAddressLookup, type AddressInsight, type AddressListing, type AddressOffering, type AddressRawTransaction, type AddressSourceCandidate, type AddressSourceRecord, type AddressTransactionLink } from '../api/koditon'
import { appendAddressLookupParams, buildAddressLookupPath, sourceEntityPath, withAddressLookupContext, type AddressLookupInput } from '../lib/address-lookup'

const sources = [
  ['all', 'All sources'],
  ['frontdoor', 'Frontdoor'],
  ['shortcut', 'Shortcut'],
] as const

export default function AddressLookupPage() {
  const [urlParams, setUrlParams] = useSearchParams()
  const lookupParams = useMemo(() => {
    const address = urlParams.get('address')?.trim() ?? ''
    return address ? { address, city: valueOrUndefined(urlParams.get('city')), postal: valueOrUndefined(urlParams.get('postal')), source: urlParams.get('source') || 'all', page_size: lookupPageSize(urlParams) } : undefined
  }, [urlParams])
  const lookup = useAddressLookup(lookupParams, { query: { enabled: Boolean(lookupParams?.address), placeholderData: previous => previous } })
  const body = lookup.data?.status === 200 ? lookup.data.data : undefined
  const listings = body?.listings ?? []
  const offerings = body?.offerings ?? []
  const rawTransactions = body?.raw_transactions ?? []
  const [viewMode, setViewMode] = useState<'grouped' | 'raw'>('grouped')
  const linkedCount = listings.filter(listing => (listing.transactions ?? []).some(isLinkedTransaction)).length
  const candidateCount = listings.reduce((count, listing) => count + (listing.transactions ?? []).filter(transaction => !isLinkedTransaction(transaction)).length, 0)
  const unaggregatedCount = listings.filter(listing => !listing.offering_id).length
  const sourceRecordCount = listings.reduce((count, listing) => count + (listing.source_records ?? []).filter(record => record.listing_id !== listing.listing_id).length, 0)
  const sourceCandidateCount = listings.reduce((count, listing) => count + (listing.source_candidates?.length ?? 0), 0)
  const offeringCount = offerings.length || new Set(listings.map(listing => listing.offering_id).filter(Boolean)).size
  const rawListings = viewMode === 'grouped' ? listings.filter(listing => !listing.offering_id) : listings
  const reviewLookup = body ? { address: body.address, city: body.city, postal: body.postal, source: body.source, pageSize: lookupParams?.page_size } : undefined
  function showMoreListings() {
    const next = new URLSearchParams(urlParams)
    next.set('page_size', '100')
    setUrlParams(next)
  }
  return (
    <main className="address-lookup-page">
      <Nav />
      <div className="address-lookup-body">
        <aside className="address-lookup-sidebar">
          <AddressLookupForm key={urlParams.toString()} initialParams={urlParams} isFetching={lookup.isFetching} onChange={setUrlParams} />
        </aside>
        <section className="address-lookup-results">
          <header className="address-lookup-header">
            <div>
              <h1>Address lookup</h1>
              <p>{body ? [body.address, body.city, body.postal].filter(Boolean).join(' / ') : 'Search source listings and connected prices transactions.'}</p>
            </div>
            {body && (
              <div className="address-lookup-stats">
                <Metric label={body.has_more_listings ? 'Source listings shown' : 'Source listings'} value={body.has_more_listings ? `${listings.length} / ${body.listing_count}` : listings.length} />
                <Metric label="Unaggregated" value={unaggregatedCount} />
                <Metric label="Offerings" value={offeringCount} />
                <Metric label="Source records" value={sourceRecordCount} />
                <Metric label="Linked" value={linkedCount} />
                <Metric label="Candidates" value={candidateCount} />
                <Metric label="Source candidates" value={sourceCandidateCount} />
                <Metric label="History" value={rawTransactions.length} />
              </div>
            )}
          </header>
          {!lookupParams?.address && <EmptyState title="Enter an address" text="Use the filters to check Shortcut and Frontdoor source listings, then inspect existing prices links." />}
          {lookup.isError && <div className="error-state">Address lookup failed.</div>}
          {lookup.isFetching && !body && <div className="loading-state">Loading address data</div>}
          {body && listings.length === 0 && <EmptyState title="No listings found" text="Try the street address without apartment letters, or add city and postal filters." />}
          {listings.length > 0 && (
            <div className="address-result-controls">
              <div className="address-view-tabs">
                <button className={viewMode === 'grouped' ? 'is-active' : ''} type="button" onClick={() => setViewMode('grouped')}>Grouped offerings</button>
                <button className={viewMode === 'raw' ? 'is-active' : ''} type="button" onClick={() => setViewMode('raw')}>Raw source listings</button>
              </div>
              {body?.has_more_listings && (lookupParams?.page_size ?? 50) < 100 && <button className="search-clear-btn" type="button" onClick={showMoreListings}>Show 100 source listings</button>}
            </div>
          )}
          {viewMode === 'grouped' && offerings.length > 0 && (
            <div className={`address-listing-list${lookup.isFetching ? ' address-listing-list--loading' : ''}`}>
              {offerings.map(offering => <OfferingCard key={offering.offering_id} offering={offering} lookup={reviewLookup} />)}
            </div>
          )}
          {viewMode === 'grouped' && rawListings.length > 0 && (
            <section className="address-ungrouped-section">
              <header>
                <h2>Ungrouped source listings</h2>
                <span>{rawListings.length} not linked to a canonical offering</span>
              </header>
              <div className={`address-listing-list${lookup.isFetching ? ' address-listing-list--loading' : ''}`}>
                {rawListings.map(listing => <ListingCard key={listing.listing_id} listing={listing} lookup={reviewLookup} />)}
              </div>
            </section>
          )}
          {viewMode === 'raw' && listings.length > 0 && (
            <div className={`address-listing-list${lookup.isFetching ? ' address-listing-list--loading' : ''}`}>
              {listings.map(listing => <ListingCard key={listing.listing_id} listing={listing} lookup={reviewLookup} />)}
            </div>
          )}
          {body && <RawTransactionPanel transactions={rawTransactions} listings={listings} lookup={reviewLookup} />}
        </section>
      </div>
    </main>
  )
}

function OfferingCard({ offering, lookup }: { offering: AddressOffering; lookup?: AddressLookupInput }) {
  const records = offering.source_records ?? []
  const transactions = offering.transactions ?? []
  const linkedTransactions = transactions.filter(isLinkedTransaction)
  const candidateTransactions = transactions.filter(transaction => !isLinkedTransaction(transaction))
  const representative = offering.representative
  const sourcePath = sourceEntityPath({ canonicalId: representative.canonical_id, kind: representative.kind })
  const recordLookup = new Map(records.map(record => [record.listing_id, record]))
  return (
    <article className="address-listing-card address-offering-card">
      <header className="address-listing-card-head">
        <div>
          <div className="search-card-badges">
            <span className="search-badge search-badge--kind">Grouped offering</span>
            {offering.sources?.map(source => <span className={`search-badge search-badge--${source === 'shortcut' ? 'shortcut' : 'frontdoor'}`} key={source}>{sourceLabel(source)}</span>)}
            {offering.housing_company_id && <span className="search-badge search-badge--linked">Company</span>}
            {linkedTransactions.length > 0 && <span className="address-link-badge">Prices linked</span>}
            {candidateTransactions.length > 0 && <span className="address-candidate-badge">Candidates</span>}
            {Boolean(offering.insights?.length) && <span className="search-badge search-badge--insight">{offering.insights?.length} insights</span>}
            {offering.source_candidate_count > 0 && <span className="address-candidate-badge">Source candidates</span>}
          </div>
          <h2>{offering.headline || offering.address || offering.offering_id}</h2>
          <p>{[offering.address, offering.city, offering.postal].filter(Boolean).join(' / ')}</p>
        </div>
        <div className="address-listing-price">
          <strong>{formatEUR(offering.asking_price)}</strong>
          <span>{formatArea(offering.area)}</span>
        </div>
      </header>
      <div className="address-listing-meta">
        <span>Offering {shortID(offering.offering_id)}</span>
        {offering.housing_company_name && <span>{offering.housing_company_name}</span>}
        <span>{offering.source_count} source records</span>
        <span>{offering.room_layout || 'No room layout'}</span>
        <span>{formatDate(offering.last_seen_at)}</span>
      </div>
      <div className="address-listing-actions">
        <Link to={withAddressLookupContext(`/target/offering/${offering.offering_id}`, lookup)}>Canonical offering</Link>
        {offering.housing_company_id && <Link to={withAddressLookupContext(`/target/housing_company/${offering.housing_company_id}`, lookup)}>Housing company</Link>}
        {sourcePath && <Link to={withAddressLookupContext(sourcePath, lookup)}>Representative source</Link>}
        <LiveSourceLink available={representative.external_url_available} url={representative.url} />
      </div>
      <InsightList insights={offering.insights} limit={6} />
      {records.length > 0 && <SourceRecordList records={records} lookup={lookup} title="Provider source records" />}
      {linkedTransactions.length > 0 && <TransactionTable title="Connected prices" transactions={linkedTransactions} sourceRecords={recordLookup} lookup={lookup} />}
      {candidateTransactions.length > 0 && <TransactionTable title="Candidate prices matches" transactions={candidateTransactions} sourceRecords={recordLookup} lookup={lookup} variant="candidate" />}
      {transactions.length === 0 && <div className="address-no-transaction">No connected prices transaction or saved match candidate for this grouped offering.</div>}
    </article>
  )
}

function AddressLookupForm({ initialParams, isFetching, onChange }: { initialParams: URLSearchParams; isFetching: boolean; onChange: (params: URLSearchParams) => void }) {
  const [addressInput, setAddressInput] = useState(() => initialParams.get('address') ?? '')
  const [cityInput, setCityInput] = useState(() => initialParams.get('city') ?? '')
  const [postalInput, setPostalInput] = useState(() => initialParams.get('postal') ?? '')
  const [sourceInput, setSourceInput] = useState(() => initialParams.get('source') ?? 'all')
  const [pageSizeInput, setPageSizeInput] = useState(() => String(lookupPageSize(initialParams)))
  function submit(event: FormEvent) {
    event.preventDefault()
    const address = addressInput.trim()
    if (!address) return
    const next = new URLSearchParams()
    next.set('address', address)
    if (cityInput.trim()) next.set('city', cityInput.trim())
    if (postalInput.trim()) next.set('postal', postalInput.trim())
    if (sourceInput !== 'all') next.set('source', sourceInput)
    if (pageSizeInput !== '50') next.set('page_size', pageSizeInput)
    onChange(next)
  }
  function clearFilters() {
    onChange(new URLSearchParams())
  }
  return (
    <form className="address-lookup-form" onSubmit={submit}>
      <label className="search-filter-group">
        <span className="search-filter-label">Address</span>
        <input id="address-lookup-address" name="address" autoComplete="street-address" className="search-input" value={addressInput} onChange={event => setAddressInput(event.target.value)} placeholder="Askvägen 4" type="search" />
      </label>
      <label className="search-filter-group">
        <span className="search-filter-label">City</span>
        <input id="address-lookup-city" name="city" autoComplete="address-level2" className="search-input" value={cityInput} onChange={event => setCityInput(event.target.value)} placeholder="Maarianhamina" type="search" />
      </label>
      <label className="search-filter-group">
        <span className="search-filter-label">Postal</span>
        <input id="address-lookup-postal" name="postal" autoComplete="postal-code" className="search-input" value={postalInput} onChange={event => setPostalInput(event.target.value)} placeholder="22100" inputMode="numeric" />
      </label>
      <label className="search-filter-group">
        <span className="search-filter-label">Source</span>
        <select id="address-lookup-source" name="source" className="search-select" value={sourceInput} onChange={event => setSourceInput(event.target.value)}>
          {sources.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </select>
      </label>
      <label className="search-filter-group">
        <span className="search-filter-label">Source listings</span>
        <select id="address-lookup-page-size" name="page_size" className="search-select" value={pageSizeInput} onChange={event => setPageSizeInput(event.target.value)}>
          <option value="25">25</option>
          <option value="50">50</option>
          <option value="100">100</option>
        </select>
      </label>
      <button className="search-filter-trigger" type="submit" disabled={!addressInput.trim() || isFetching}>{isFetching ? 'Searching' : 'Find listings'}</button>
      <button className="search-clear-btn" type="button" onClick={clearFilters} disabled={!addressInput && !cityInput && !postalInput && sourceInput === 'all' && pageSizeInput === '50'}>Clear</button>
    </form>
  )
}

function RawTransactionPanel({ transactions, listings, lookup }: { transactions: AddressRawTransaction[]; listings: AddressListing[]; lookup?: AddressLookupInput }) {
  const linkedHere = transactions.filter(transaction => transaction.linked_to_lookup).length
  const candidateHere = transactions.filter(transaction => !transaction.linked_to_lookup && transaction.candidate_to_lookup).length
  const matchedElsewhere = transactions.filter(transaction => !transaction.linked_to_lookup && !transaction.candidate_to_lookup && transaction.is_matched).length
  const unlinked = transactions.filter(transaction => !transaction.linked_to_lookup && !transaction.is_matched && !transaction.candidate_to_lookup).length
  return (
    <section className="address-raw-transactions">
      <header>
        <div>
          <h2>Prices history</h2>
          <span>{linkedHere} linked here / {candidateHere} candidates here / {matchedElsewhere} matched elsewhere / {unlinked} postal history</span>
        </div>
      </header>
      {transactions.length === 0 && <div className="address-raw-transaction-empty">No linked, candidate, or postal prices history found for this lookup.</div>}
      <div className="address-raw-transaction-list">
        {transactions.map(transaction => {
          const facts = priceTransactionFacts(transaction)
          const candidateMatches = candidateRawTransactionMatches(transaction, listings)
          return (
            <div className={`address-raw-transaction${transaction.linked_to_lookup ? ' address-raw-transaction--linked' : transaction.candidate_to_lookup ? ' address-raw-transaction--candidate' : ''}`} key={transaction.transaction_id}>
              <div>
                <strong>{transaction.description || transaction.transaction_id}</strong>
                <span>{pricesTransactionIdentity(transaction.transaction_id)}</span>
                <span>{[transaction.category, transaction.type, transaction.period_identifier].filter(Boolean).join(' / ')}</span>
                {facts && <span>{facts}</span>}
              </div>
              <div>
                <strong>{formatEUR(transaction.price)}</strong>
                <span>{[formatArea(transaction.area), formatPricePerM2(transaction.price_per_square_meter)].filter(Boolean).join(' / ')}</span>
              </div>
              <div>
                <strong>{rawTransactionStatus(transaction)}</strong>
                <span>{[transaction.neighborhood, transaction.postal, formatShortDate(transaction.created_at)].filter(Boolean).join(' / ')}</span>
                <RawTransactionMatches transaction={transaction} candidateMatches={candidateMatches} lookup={lookup} />
              </div>
              <Link className="address-transaction-review" to={transactionMatchURL(transaction, lookup)}>Review</Link>
            </div>
          )
        })}
      </div>
    </section>
  )
}

function ListingCard({ listing, lookup }: { listing: AddressListing; lookup?: AddressLookupInput }) {
  const transactions = listing.transactions ?? []
  const linkedTransactions = transactions.filter(isLinkedTransaction)
  const candidateTransactions = transactions.filter(transaction => !isLinkedTransaction(transaction))
  const sourceRecords = (listing.source_records ?? []).filter(record => record.listing_id !== listing.listing_id)
  const sourceCandidates = listing.source_candidates ?? []
  const transactionSources = sourceRecordLookup(listing)
  const coverage = sourceCoverage(listing)
  const sourcePath = sourceEntityPath({ canonicalId: listing.canonical_id, kind: listing.kind })
  const postalReviewPath = postalMatchURL(listing.postal, lookup)
  return (
    <article className="address-listing-card">
      <header className="address-listing-card-head">
        <div>
          <div className="search-card-badges">
            <span className={`search-badge search-badge--${listing.source === 'shortcut' ? 'shortcut' : 'frontdoor'}`}>{sourceLabel(listing.source)}</span>
            <span className="search-badge search-badge--kind">{listing.kind}</span>
            {linkedTransactions.length > 0 && <span className="address-link-badge">Prices linked</span>}
            {candidateTransactions.length > 0 && <span className="address-candidate-badge">Candidates</span>}
            {Boolean(listing.insights?.length) && <span className="search-badge search-badge--insight">{listing.insights?.length} insights</span>}
            {sourceCandidates.length > 0 && <span className="address-candidate-badge">Source candidates</span>}
            {listing.housing_company_id && <span className="search-badge search-badge--linked">Company</span>}
          </div>
          <h2>{listing.headline || listing.address || listing.canonical_id}</h2>
          <p>{[listing.address, listing.city, listing.postal].filter(Boolean).join(' / ')}</p>
        </div>
        <div className="address-listing-price">
          <strong>{formatEUR(listing.asking_price)}</strong>
          <span>{formatArea(listing.area)}</span>
        </div>
      </header>
      <div className="address-listing-meta">
        <span>{sourceLabel(listing.source)} {listing.native_id}</span>
        <span>{listing.offering_id ? `Offering ${shortID(listing.offering_id)}` : 'Unaggregated source listing'}</span>
        {listing.housing_company_name && <span>{listing.housing_company_name}</span>}
        <span>{listing.room_layout || 'No room layout'}</span>
        <span>{formatDate(listing.last_seen_at)}</span>
        {listing.price_match_status && <span>{listing.price_match_status}</span>}
        {listing.source_match_status && <span>{listing.source_match_status}</span>}
      </div>
      <ListingTimeline listing={listing} />
      {coverage.length > 0 && <div className="address-source-coverage">{coverage.map(item => <span key={item}>{item}</span>)}</div>}
      <InsightList insights={listing.insights} limit={4} />
      <SourceTexts texts={listing.texts} />
      {sourceRecords.length > 0 && <SourceRecordList records={sourceRecords} lookup={lookup} />}
      {sourceCandidates.length > 0 && <SourceCandidateList candidates={sourceCandidates} lookup={lookup} />}
      <div className="address-listing-actions">
        {listing.offering_id && <Link to={withAddressLookupContext(`/target/offering/${listing.offering_id}`, lookup)}>Canonical offering</Link>}
        {listing.housing_company_id && <Link to={withAddressLookupContext(`/target/housing_company/${listing.housing_company_id}`, lookup)}>Housing company</Link>}
        {sourcePath && <Link to={withAddressLookupContext(sourcePath, lookup)}>Source detail</Link>}
        {postalReviewPath && <Link to={postalReviewPath}>Review postal candidates</Link>}
        <LiveSourceLink available={listing.external_url_available} url={listing.url} />
      </div>
      {linkedTransactions.length > 0 && <TransactionTable title="Connected prices" transactions={linkedTransactions} sourceRecords={transactionSources} lookup={lookup} />}
      {candidateTransactions.length > 0 && <TransactionTable title="Candidate prices matches" transactions={candidateTransactions} sourceRecords={transactionSources} lookup={lookup} variant="candidate" />}
      {transactions.length === 0 && <div className="address-no-transaction">No connected prices transaction or saved match candidate for this listing.</div>}
    </article>
  )
}

function ListingTimeline({ listing }: { listing: AddressListing }) {
  const priceHistory = [
    listing.previous_asking_price != null && listing.previous_asking_price !== listing.asking_price ? `Ask was ${formatEUR(listing.previous_asking_price)}` : '',
    listing.previous_debt_free_price != null && listing.previous_debt_free_price !== listing.debt_free_price ? `Debt-free was ${formatEUR(listing.previous_debt_free_price)}` : '',
  ].filter(Boolean)
  return (
    <div className="address-listing-timeline">
      <span>First seen {formatShortDate(listing.first_seen_at)}</span>
      <span>Last seen {formatShortDate(listing.last_seen_at)}</span>
      <span>Updated {formatShortDate(listing.updated_at)}</span>
      {priceHistory.map(item => <strong key={item}>{item}</strong>)}
    </div>
  )
}

function SourceCandidateList({ candidates, lookup }: { candidates: AddressSourceCandidate[]; lookup?: AddressLookupInput }) {
  return (
    <div className="address-source-records address-source-candidates">
      <span className="address-source-records-title">Candidate source matches</span>
      {candidates.map(candidate => {
        const sourcePath = sourceEntityPath({ canonicalId: candidate.canonical_id, kind: candidate.kind })
        const lookupPath = buildAddressLookupPath({ address: candidate.address, city: candidate.city, postal: candidate.postal, source: candidate.source, pageSize: lookup?.pageSize })
        return (
          <div className="address-source-record" key={`${candidate.listing_id}:${candidate.candidate_offering_id}`}>
            <div>
              <strong>{candidate.headline || candidate.address || candidate.native_id}</strong>
              <span>{[candidate.address, candidate.city, candidate.postal].filter(Boolean).join(' / ')}</span>
            </div>
            <div>
              <strong>{formatEUR(candidate.asking_price)}</strong>
              <span>{[formatArea(candidate.area), candidate.room_layout].filter(Boolean).join(' / ')}</span>
            </div>
            <div>
              <strong>{sourceLabel(candidate.source)} {candidate.native_id}</strong>
              <span>{[candidate.status, candidate.confidence, formatScore(candidate.score), formatPercent(candidate.price_delta_percent)].filter(Boolean).join(' / ')}</span>
              <span>{candidate.direction}</span>
            </div>
            {candidate.reasons_summary && candidate.reasons_summary.length > 0 && (
              <div className="address-transaction-evidence">
                {candidate.reasons_summary.map(item => <span key={item}>{item}</span>)}
              </div>
            )}
            <div className="address-source-record-actions">
              {lookupPath && <Link to={lookupPath}>Address lookup</Link>}
              {candidate.candidate_offering_id && <Link to={withAddressLookupContext(`/target/offering/${candidate.candidate_offering_id}`, lookup)}>Candidate offering</Link>}
              {sourcePath && <Link to={withAddressLookupContext(sourcePath, lookup)}>Source detail</Link>}
              <LiveSourceLink available={candidate.external_url_available} url={candidate.url} />
            </div>
          </div>
        )
      })}
    </div>
  )
}

function SourceRecordList({ records, lookup, title = 'Same offering source records' }: { records: AddressSourceRecord[]; lookup?: AddressLookupInput; title?: string }) {
  return (
    <div className="address-source-records">
      <span className="address-source-records-title">{title}</span>
      {records.map(record => {
        const lookupPath = buildAddressLookupPath({ address: record.address, city: record.city, postal: record.postal, source: record.source, pageSize: lookup?.pageSize })
        const sourcePath = sourceEntityPath({ canonicalId: record.canonical_id, kind: record.kind })
        return (
          <div className="address-source-record" key={record.listing_id}>
            <div>
              <strong>{record.headline || record.address || record.native_id}</strong>
              <span>{[record.address, record.city, record.postal].filter(Boolean).join(' / ')}</span>
            </div>
            <div>
              <strong>{formatEUR(record.asking_price)}</strong>
              <span>{[formatArea(record.area), record.room_layout].filter(Boolean).join(' / ')}</span>
            </div>
            <div>
              <strong>{sourceLabel(record.source)} {record.native_id}</strong>
              <span>{sourceRecordTimeline(record)}</span>
              {sourceRecordLink(record) && <span>{sourceRecordLink(record)}</span>}
            </div>
            <InsightList insights={record.insights} limit={3} compact />
            <div className="address-source-record-actions">
              {record.previous_asking_price != null && record.previous_asking_price !== record.asking_price && <span>Was {formatEUR(record.previous_asking_price)}</span>}
              {lookupPath && <Link to={lookupPath}>Address lookup</Link>}
              {sourcePath && <Link to={withAddressLookupContext(sourcePath, lookup)}>Source detail</Link>}
              <LiveSourceLink available={record.external_url_available} url={record.url} />
            </div>
            <SourceTexts texts={record.texts} compact />
          </div>
        )
      })}
    </div>
  )
}

function SourceTexts({ texts, compact = false }: { texts?: AddressListing['texts']; compact?: boolean }) {
  const items = sourceTextItems(texts)
  if (items.length === 0) return null
  return (
    <div className={compact ? 'address-source-record-texts' : 'address-listing-texts'}>
      {items.map(item => (
        <div key={item.label}>
          <strong>{item.label}</strong>
          <span>{item.value}</span>
        </div>
      ))}
    </div>
  )
}

function InsightList({ insights, limit, compact = false }: { insights?: AddressInsight[] | null; limit: number; compact?: boolean }) {
  const visible = insights?.slice(0, limit) ?? []
  if (visible.length === 0) return null
  return (
    <div className={compact ? 'address-insights address-insights--compact' : 'address-insights'}>
      {visible.map(insight => (
        <span key={`${insight.source_field}:${insight.key}:${insight.value}`}>
          <strong>{insightLabel(insight)}</strong>
          {insight.value}
        </span>
      ))}
      {insights && insights.length > visible.length && <span>{insights.length - visible.length} more</span>}
    </div>
  )
}

function TransactionTable({ title, transactions, sourceRecords, lookup, variant = 'linked' }: { title: string; transactions: AddressTransactionLink[]; sourceRecords: Map<string, AddressSourceRecord>; lookup?: AddressLookupInput; variant?: 'linked' | 'candidate' }) {
  return (
    <div className={`address-transaction-table address-transaction-table--${variant}`}>
      <div className="address-transaction-title">{title}</div>
      {transactions.map(transaction => {
        const evidence = transaction.reasons_summary?.length ? transaction.reasons_summary : transactionEvidence(transaction, sourceRecords)
        const facts = priceTransactionFacts(transaction)
        return (
          <div className="address-transaction-row" key={`${transaction.transaction_id}:${transaction.link_type}`}>
            <div>
              <strong>{transaction.description || transaction.transaction_id}</strong>
              <span>{pricesTransactionIdentity(transaction.transaction_id)}</span>
              <span>{[transaction.category, transaction.type, transaction.period_identifier].filter(Boolean).join(' / ')}</span>
              {facts && <span>{facts}</span>}
            </div>
            <div>
              <strong>{formatEUR(transaction.price)}</strong>
              <span>{[formatArea(transaction.area), formatPricePerM2(transaction.price_per_square_meter)].filter(Boolean).join(' / ')}</span>
            </div>
            <div>
              <strong>{formatLinkType(transaction.link_type)}</strong>
              <span>{[formatLinkStatus(transaction.link_status), transaction.confidence, formatScore(transaction.score), formatPercent(transaction.price_delta_percent)].filter(Boolean).join(' / ')}</span>
              <Link className="address-transaction-review" to={transactionMatchURL(transaction, lookup)}>Review match</Link>
            </div>
            {evidence.length > 0 && (
              <div className="address-transaction-evidence">
                {evidence.map(item => <span key={item}>{item}</span>)}
              </div>
            )}
          </div>
        )
      })}
    </div>
  )
}

function Metric({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="model-metric">
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function EmptyState({ title, text }: { title: string; text: string }) {
  return (
    <div className="search-empty-hint">
      <strong>{title}</strong>
      <span className="search-empty-sub">{text}</span>
    </div>
  )
}

function valueOrUndefined(value: string | null) {
  const trimmed = value?.trim()
  return trimmed || undefined
}

function lookupPageSize(params: URLSearchParams) {
  const value = Number(params.get('page_size'))
  if (value === 25 || value === 50 || value === 100) return value
  return 50
}

function sourceLabel(source?: string) {
  if (source === 'shortcut') return 'Shortcut'
  if (source === 'frontdoor') return 'Frontdoor'
  return source || 'Source'
}

function isLinkedTransaction(transaction: AddressTransactionLink) {
  const type = transaction.link_type?.toLowerCase()
  const status = transaction.link_status?.toLowerCase()
  return type === 'direct' || type === 'offering' || type === 'source_record' || status === 'linked' || status === 'auto_linked'
}

function formatEUR(value?: number | null) {
  if (value == null) return 'n/a'
  return `${new Intl.NumberFormat('fi-FI').format(value)} EUR`
}

function formatArea(value?: number | null) {
  if (value == null) return ''
  return `${new Intl.NumberFormat('fi-FI', { maximumFractionDigits: 1 }).format(value)} m2`
}

function formatPricePerM2(value?: number | null) {
  if (value == null) return ''
  return `${new Intl.NumberFormat('fi-FI').format(value)} EUR/m2`
}

function formatDate(value?: string | null) {
  if (!value) return 'No last seen date'
  return new Intl.DateTimeFormat('fi-FI', { year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(value))
}

function formatShortDate(value?: string | null) {
  if (!value) return 'n/a'
  return new Intl.DateTimeFormat('fi-FI', { year: '2-digit', month: '2-digit', day: '2-digit' }).format(new Date(value))
}

function formatScore(value?: number | null) {
  if (value == null) return ''
  return `score ${value}`
}

function shortID(value: string) {
  return value.slice(0, 8)
}

function pricesTransactionIdentity(id: string) {
  return `Prices ${shortID(id)}`
}

function formatBool(value?: boolean | null) {
  if (value == null) return ''
  return value ? 'Yes' : 'No'
}

function priceTransactionFacts(transaction: AddressRawTransaction | AddressTransactionLink) {
  return [
    transaction.build_year ? `Built ${transaction.build_year}` : '',
    transaction.floor ? `Floor ${transaction.floor}` : '',
    formatBool(transaction.elevator) ? `Elevator ${formatBool(transaction.elevator)}` : '',
    transaction.condition,
    transaction.plot,
    transaction.energy_class ? `Energy ${transaction.energy_class}` : '',
  ].filter(Boolean).join(' / ')
}

function sourceRecordTimeline(record: AddressSourceRecord) {
  return `Seen ${formatShortDate(record.first_seen_at)} - ${formatShortDate(record.last_seen_at)}`
}

function sourceRecordLink(record: AddressSourceRecord) {
  return [record.link_status, record.link_method, formatScore(record.link_score)].filter(Boolean).join(' / ')
}

function sourceCoverage(listing: AddressListing) {
  const records = listing.source_records?.length ? listing.source_records : [{
    listing_id: listing.listing_id,
    source: listing.source,
  }]
  const sources = new Set(records.map(record => record.source).filter(Boolean))
  const labels = Array.from(sources).map(source => sourceLabel(source)).sort()
  return [
    `${records.length} ${records.length === 1 ? 'source record' : 'source records'}`,
    labels.length > 0 ? labels.join(' + ') : '',
  ].filter(Boolean)
}

function sourceRecordLookup(listing: AddressListing) {
  const records = listing.source_records?.length ? listing.source_records : [{
    listing_id: listing.listing_id,
    source: listing.source,
    native_id: listing.native_id,
  } as AddressSourceRecord]
  return new Map(records.map(record => [record.listing_id, record]))
}

function sourceTextItems(texts?: AddressListing['texts']) {
  if (!texts) return []
  return [
    ['Availability', texts.availability],
    ['Renovations done', texts.renovations_done],
    ['Renovations planned', texts.renovations_planned],
    ['Additional info', texts.additional_info],
    ['Charges', texts.charges],
  ].flatMap(([label, value]) => {
    const text = typeof value === 'string' ? value.trim() : ''
    return text ? [{ label, value: text }] : []
  })
}

function insightLabel(insight: AddressInsight) {
  return [insight.severity, insight.key].filter(Boolean).join(' / ') || 'Insight'
}

function formatPercent(value?: number | null) {
  if (value == null) return ''
  const percent = Math.abs(value) <= 1 ? value * 100 : value
  return `${new Intl.NumberFormat('fi-FI', { maximumFractionDigits: 1 }).format(percent)}% delta`
}

function transactionMatchURL(transaction: { transaction_id: string; postal?: string | null }, lookup?: AddressLookupInput) {
  const params = new URLSearchParams()
  if (transaction.postal) params.set('postal', transaction.postal)
  params.set('transaction', transaction.transaction_id)
  appendAddressLookupParams(params, lookup)
  return `/matches?${params.toString()}`
}

function postalMatchURL(postal?: string | null, lookup?: AddressLookupInput) {
  const trimmed = postal?.trim()
  if (!trimmed) return ''
  const params = new URLSearchParams()
  params.set('postal', trimmed)
  appendAddressLookupParams(params, lookup)
  return `/matches?${params.toString()}`
}

function rawTransactionStatus(transaction: AddressRawTransaction) {
  if (transaction.scope === 'linked_here') return 'Linked here'
  if (transaction.scope === 'candidate_here') return 'Candidate here'
  if (transaction.scope === 'matched_elsewhere') return [
    countLabel(transaction.matched_listing_count, 'source link'),
    countLabel(transaction.matched_offering_count, 'offering link'),
  ].filter(Boolean).join(' / ')
  return 'Postal history'
}

function countLabel(count: number, label: string) {
  if (count <= 0) return ''
  return `${count} ${label}${count === 1 ? '' : 's'}`
}

type CandidateRawTransactionMatch = {
  id: string
  label: string
  path: string
  status: string
}

function RawTransactionMatches({ transaction, candidateMatches, lookup }: { transaction: AddressRawTransaction; candidateMatches: CandidateRawTransactionMatch[]; lookup?: AddressLookupInput }) {
  const matches = transaction.matches ?? []
  if (matches.length === 0 && candidateMatches.length === 0) return null
  return (
    <span className="address-raw-transaction-matches">
      {matches.map(match => {
        const label = rawTransactionMatchLabel(match)
        const status = [match.status, formatLinkMethod(match.method), formatScore(match.score)].filter(Boolean).join(' / ')
        const path = withAddressLookupContext(rawTransactionMatchPath(match), lookup)
        const text = status ? `${label} (${status})` : label
        return (
          <span key={`${match.type}:${match.id}`}>
            {path ? <Link to={path}>{text}</Link> : text}
            {match.offering_id && <Link to={withAddressLookupContext(`/target/offering/${match.offering_id}`, lookup)}>Canonical offering</Link>}
          </span>
        )
      })}
      {candidateMatches.map(match => {
        const text = match.status ? `${match.label} (${match.status})` : match.label
        const path = withAddressLookupContext(match.path, lookup)
        return path ? <Link key={match.id} to={path}>{text}</Link> : <span key={match.id}>{text}</span>
      })}
    </span>
  )
}

function candidateRawTransactionMatches(transaction: AddressRawTransaction, listings: AddressListing[]) {
  return listings.flatMap(listing => {
    const candidate = (listing.transactions ?? []).find(item => !isLinkedTransaction(item) && item.transaction_id === transaction.transaction_id)
    if (!candidate) return []
    const label = listing.headline || listing.address || [sourceLabel(listing.source), listing.native_id].filter(Boolean).join(' ') || listing.listing_id.slice(0, 8)
    return [{
      id: `candidate:${listing.listing_id}:${transaction.transaction_id}`,
      label,
      path: sourceEntityPath({ canonicalId: listing.canonical_id, kind: listing.kind }),
      status: ['candidate', candidate.confidence, formatScore(candidate.score), formatPercent(candidate.price_delta_percent)].filter(Boolean).join(' / '),
    }]
  })
}

function rawTransactionMatchLabel(match: NonNullable<AddressRawTransaction['matches']>[number]) {
  const target = [sourceLabel(match.source), match.native_id].filter(Boolean).join(' ')
  const label = match.headline || match.address || target || match.id.slice(0, 8)
  if (match.type === 'offering_source') return `Offering source ${label}`
  if (match.type === 'listing') return `Direct source ${label}`
  return label
}

function rawTransactionMatchPath(match: NonNullable<AddressRawTransaction['matches']>[number]) {
  if (match.type === 'offering' && match.id) return `/target/offering/${match.id}`
  return sourceEntityPath({ canonicalId: match.canonical_id })
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

function transactionEvidence(transaction: AddressTransactionLink, sourceRecords: Map<string, AddressSourceRecord>) {
  const reasons = isRecord(transaction.reasons) ? transaction.reasons : {}
  const evidence = [
    reasonString(reasons, 'postal') ? `Postal ${reasonString(reasons, 'postal')}` : '',
    reasonBoolean(reasons, 'layout_prefix') ? 'Layout prefix' : '',
    reasonNumber(reasons, 'area') != null ? `Area ${formatArea(reasonNumber(reasons, 'area'))}` : '',
    reasonPair(reasons, 'build_year', 'Build year'),
    reasonPair(reasons, 'floor_level', 'Floor'),
    reasonPair(reasons, 'total_floors', 'Floors'),
    reasonString(reasons, 'property_type') ? `Type ${reasonString(reasons, 'property_type')}` : '',
    reasonSourceListing(reasons, sourceRecords),
  ]
  return evidence.filter(Boolean).slice(0, 8)
}

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value)
}

function reasonString(reasons: Record<string, unknown>, key: string) {
  const value = reasons[key]
  return typeof value === 'string' && value.trim() ? value.trim() : ''
}

function reasonNumber(reasons: Record<string, unknown>, key: string) {
  const value = reasons[key]
  return typeof value === 'number' && Number.isFinite(value) ? value : undefined
}

function reasonBoolean(reasons: Record<string, unknown>, key: string) {
  return reasons[key] === true
}

function reasonPair(reasons: Record<string, unknown>, key: string, label: string) {
  const value = reasons[key]
  if (!isRecord(value)) return ''
  const listing = value.listing
  const transaction = value.transaction
  if ((typeof listing !== 'string' && typeof listing !== 'number') || (typeof transaction !== 'string' && typeof transaction !== 'number')) return ''
  return `${label} ${listing}/${transaction}`
}

function reasonSourceListing(reasons: Record<string, unknown>, sourceRecords: Map<string, AddressSourceRecord>) {
  const id = reasonString(reasons, 'source_listing_id')
  if (!id) return ''
  const record = sourceRecords.get(id)
  if (record) return `Source ${sourceLabel(record.source)} ${record.native_id}`
  const provider = reasonString(reasons, 'source_listing_provider')
  const nativeID = reasonString(reasons, 'source_listing_native_id')
  if (provider || nativeID) return `Source ${[sourceLabel(provider), nativeID].filter(Boolean).join(' ')}`
  return `Source ${id.slice(0, 8)}`
}
