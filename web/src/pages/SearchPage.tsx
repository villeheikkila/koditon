import { useMemo, useState, type FormEvent } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
import LiveSourceLink from '../components/LiveSourceLink'
import Nav from '../components/Nav'
import { usePropertyDocumentsManagerCertificatesUpload, useSearch, useSearchGroupedOfferings, type GroupedOfferingSearchRow, type SearchParams, type SearchResultRow } from '../api/koditon'
import { buildAddressLookupPath, sourceEntityPath } from '../lib/address-lookup'

const targetTypes = [
  ['offering', 'Offering'],
  ['unit', 'Unit'],
  ['building', 'Building'],
  ['housing_company', 'Housing company'],
] as const

const sources = [
  ['all', 'All sources'],
  ['frontdoor', 'Frontdoor'],
  ['shortcut', 'Shortcut'],
] as const

const kinds = [
  ['all', 'All kinds'],
  ['ad', 'Sale ads'],
  ['announcement', 'Announcements'],
  ['building', 'Buildings'],
] as const

const groupings = [
  ['all', 'All grouping'],
  ['grouped', 'Grouped'],
  ['ungrouped', 'Ungrouped'],
] as const

const sorts = [
  ['seen_desc', 'Latest seen'],
  ['price_asc', 'Price ascending'],
  ['price_desc', 'Price descending'],
  ['area_asc', 'Area ascending'],
  ['area_desc', 'Area descending'],
] as const

export default function SearchPage() {
  const [urlParams, setUrlParams] = useSearchParams()
  const searchParams = useMemo(() => searchParamsFromURL(urlParams), [urlParams])
  const viewMode = urlParams.get('view') === 'grouped' ? 'grouped' : 'raw'
  const searchQuery = useSearch(searchParams, { query: { enabled: viewMode === 'raw', placeholderData: previous => previous } })
  const groupedQuery = useSearchGroupedOfferings(searchParams, { query: { enabled: viewMode === 'grouped', placeholderData: previous => previous } })
  const rawBody = searchQuery.data?.status === 200 ? searchQuery.data.data : undefined
  const groupedBody = groupedQuery.data?.status === 200 ? groupedQuery.data.data : undefined
  const body = viewMode === 'grouped' ? groupedBody : rawBody
  const rows = rawBody?.rows ?? []
  const groupedRows = groupedBody?.rows ?? []
  const activeQuery = viewMode === 'grouped' ? groupedQuery : searchQuery
  const pageTitle = viewMode === 'grouped' ? 'Grouped offerings' : 'Source listings'
  const pageDescription = viewMode === 'grouped' ? 'Browse canonical grouped offerings with provider coverage, prices matches, insights, and housing-company context.' : 'Browse provider records through the common listing model, then open raw detail or address grouping.'
  return (
    <main className="search-layout">
      <Nav />
      <div className="search-body">
        <aside className="search-sidebar">
          <SearchFilters key={urlParams.toString()} initialParams={urlParams} onChange={setUrlParams} />
          <TargetTools />
        </aside>
        <section className="search-results">
          <header className="search-page-head">
            <div>
              <h1>{pageTitle}</h1>
              <p>{pageDescription}</p>
            </div>
            <span className="search-total">{body ? `${body.total.toLocaleString('fi-FI')} results` : 'Loading'}</span>
          </header>
          <div className="search-filter-toolbar">
            <span className="search-filter-summary">{filterSummary(searchParams)}</span>
            <div className="search-view-switch">
              {viewMode === 'raw' ? <span className="search-view-tab search-view-tab--active">Raw</span> : <Link className="search-view-tab" to={searchViewPath(urlParams, 'raw')}>Raw</Link>}
              {viewMode === 'grouped' ? <span className="search-view-tab search-view-tab--active">Grouped</span> : <Link className="search-view-tab" to={searchViewPath(urlParams, 'grouped')}>Grouped</Link>}
              <Link className="search-view-tab" to={addressPathFromParams(searchParams) || '/address'}>Address</Link>
            </div>
          </div>
          {activeQuery.isLoading && <div className="search-loading">Loading {viewMode === 'grouped' ? 'grouped offerings' : 'source listings'}</div>}
          {activeQuery.isError && <div className="error-state">Source listing search failed.</div>}
          {!activeQuery.isLoading && body && body.rows?.length === 0 && <div className="search-no-results">No {viewMode === 'grouped' ? 'grouped offerings' : 'source listings'} match these filters.</div>}
          {viewMode === 'raw' && rows.length > 0 && (
            <div className={`search-grid${activeQuery.isFetching ? ' search-grid--faded' : ''}`}>
              {rows.map(row => <SearchListingCard row={row} key={row.canonical_id} />)}
            </div>
          )}
          {viewMode === 'grouped' && groupedRows.length > 0 && (
            <div className={`search-grid${activeQuery.isFetching ? ' search-grid--faded' : ''}`}>
              {groupedRows.map(group => <GroupedSearchCard group={group} key={group.offering_id} />)}
            </div>
          )}
          {body && body.total > body.page_size && <Pagination body={body} urlParams={urlParams} onChange={setUrlParams} />}
        </section>
      </div>
    </main>
  )
}

function SearchFilters({ initialParams, onChange }: { initialParams: URLSearchParams; onChange: (params: URLSearchParams) => void }) {
  const view = initialParams.get('view')
  const [query, setQuery] = useState(initialParams.get('q') ?? '')
  const [source, setSource] = useState(initialParams.get('source') ?? 'all')
  const [kind, setKind] = useState(initialParams.get('kind') ?? 'all')
  const [grouping, setGrouping] = useState(view === 'grouped' ? 'grouped' : initialParams.get('grouping') ?? 'all')
  const [city, setCity] = useState(initialParams.get('city') ?? '')
  const [postal, setPostal] = useState(initialParams.get('postal') ?? '')
  const [minPrice, setMinPrice] = useState(initialParams.get('min_price') ?? '')
  const [maxPrice, setMaxPrice] = useState(initialParams.get('max_price') ?? '')
  const [minArea, setMinArea] = useState(initialParams.get('min_area') ?? '')
  const [maxArea, setMaxArea] = useState(initialParams.get('max_area') ?? '')
  const [sort, setSort] = useState(initialParams.get('sort') ?? 'seen_desc')
  function submit(event: FormEvent) {
    event.preventDefault()
    const next = new URLSearchParams()
    if (view === 'grouped') next.set('view', 'grouped')
    setIfPresent(next, 'q', query)
    setIfPresent(next, 'source', source === 'all' ? '' : source)
    setIfPresent(next, 'kind', kind === 'all' ? '' : kind)
    setIfPresent(next, 'grouping', view === 'grouped' ? 'grouped' : grouping === 'all' ? '' : grouping)
    setIfPresent(next, 'city', city)
    setIfPresent(next, 'postal', postal)
    setIfPresent(next, 'min_price', minPrice)
    setIfPresent(next, 'max_price', maxPrice)
    setIfPresent(next, 'min_area', minArea)
    setIfPresent(next, 'max_area', maxArea)
    setIfPresent(next, 'sort', sort === 'seen_desc' ? '' : sort)
    onChange(next)
  }
  function clear() {
    setQuery('')
    setSource('all')
    setKind('all')
    setGrouping('all')
    setCity('')
    setPostal('')
    setMinPrice('')
    setMaxPrice('')
    setMinArea('')
    setMaxArea('')
    setSort('seen_desc')
    const next = new URLSearchParams()
    if (view === 'grouped') {
      next.set('view', 'grouped')
      next.set('grouping', 'grouped')
    }
    onChange(next)
  }
  return (
    <form onSubmit={submit}>
      <div className="search-filter-group">
        <label className="search-filter-label" htmlFor="search-query">Search</label>
        <input className="search-input" id="search-query" value={query} onChange={event => setQuery(event.target.value)} placeholder="Address, headline, canonical id" />
      </div>
      <div className="search-filter-row">
        <label className="search-filter-group">
          <span className="search-filter-label">Source</span>
          <select className="search-select" value={source} onChange={event => setSource(event.target.value)}>
            {sources.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
        <label className="search-filter-group">
          <span className="search-filter-label">Kind</span>
          <select className="search-select" value={kind} onChange={event => setKind(event.target.value)}>
            {kinds.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
      </div>
      <label className="search-filter-group">
        <span className="search-filter-label">Grouping</span>
        <select className="search-select" value={grouping} disabled={view === 'grouped'} onChange={event => setGrouping(event.target.value)}>
          {groupings.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </select>
      </label>
      <div className="search-filter-row">
        <label className="search-filter-group">
          <span className="search-filter-label">City</span>
          <input className="search-input" value={city} onChange={event => setCity(event.target.value)} />
        </label>
        <label className="search-filter-group">
          <span className="search-filter-label">Postal</span>
          <input className="search-input" value={postal} onChange={event => setPostal(event.target.value)} />
        </label>
      </div>
      <div className="search-filter-group">
        <span className="search-filter-label">Price</span>
        <div className="search-range-row">
          <input className="search-input" inputMode="numeric" value={minPrice} onChange={event => setMinPrice(event.target.value)} placeholder="Min" />
          <span className="search-range-sep">to</span>
          <input className="search-input" inputMode="numeric" value={maxPrice} onChange={event => setMaxPrice(event.target.value)} placeholder="Max" />
        </div>
      </div>
      <div className="search-filter-group">
        <span className="search-filter-label">Area</span>
        <div className="search-range-row">
          <input className="search-input" inputMode="decimal" value={minArea} onChange={event => setMinArea(event.target.value)} placeholder="Min" />
          <span className="search-range-sep">to</span>
          <input className="search-input" inputMode="decimal" value={maxArea} onChange={event => setMaxArea(event.target.value)} placeholder="Max" />
        </div>
      </div>
      <label className="search-filter-group">
        <span className="search-filter-label">Sort</span>
        <select className="search-select" value={sort} onChange={event => setSort(event.target.value)}>
          {sorts.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
        </select>
      </label>
      <button className="search-filter-trigger" type="submit">Apply filters</button>
      <button className="search-clear-btn" type="button" onClick={clear}>Clear filters</button>
    </form>
  )
}

function TargetTools() {
  const navigate = useNavigate()
  const uploadMutation = usePropertyDocumentsManagerCertificatesUpload()
  const [targetType, setTargetType] = useState('offering')
  const [targetID, setTargetID] = useState('')
  const [uploadTargetType, setUploadTargetType] = useState('offering')
  const [uploadTargetID, setUploadTargetID] = useState('')
  const [uploadMessage, setUploadMessage] = useState('')
  function openTarget(event: FormEvent) {
    event.preventDefault()
    if (!targetID.trim()) return
    navigate(`/target/${targetType}/${targetID.trim()}`)
  }
  async function uploadCertificate(file: File | undefined) {
    if (!file) return
    setUploadMessage('')
    const params = uploadTargetID.trim() ? { target_type: uploadTargetType, target_id: uploadTargetID.trim() } : undefined
    const response = await uploadMutation.mutateAsync({ data: { file }, params })
    const body = response.data as { document?: { id?: string } }
    setUploadMessage(`Uploaded ${body.document?.id ?? 'document'}`)
    if (params) navigate(`/target/${uploadTargetType}/${uploadTargetID.trim()}`)
  }
  return (
    <section className="search-target-tools">
      <h2>Target tools</h2>
      <form className="model-form" onSubmit={openTarget}>
        <label>
          <span>Target type</span>
          <select value={targetType} onChange={event => setTargetType(event.target.value)}>
            {targetTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
        <label>
          <span>Target id</span>
          <input value={targetID} onChange={event => setTargetID(event.target.value)} placeholder="UUID" />
        </label>
        <button type="submit">Open target</button>
      </form>
      <div className="model-form">
        <label>
          <span>Attach certificate</span>
          <select value={uploadTargetType} onChange={event => setUploadTargetType(event.target.value)}>
            {targetTypes.map(([value, label]) => <option key={value} value={value}>{label}</option>)}
          </select>
        </label>
        <label>
          <span>Target id</span>
          <input value={uploadTargetID} onChange={event => setUploadTargetID(event.target.value)} placeholder="Optional" />
        </label>
        <label className="model-file-button">
          {uploadMutation.isPending ? 'Uploading...' : 'Choose PDF'}
          <input disabled={uploadMutation.isPending} type="file" accept="application/pdf" onChange={event => uploadCertificate(event.target.files?.[0])} />
        </label>
        {uploadMessage && <p className="model-form-note">{uploadMessage}</p>}
        {uploadMutation.isError && <p className="model-form-error">Upload failed.</p>}
      </div>
    </section>
  )
}

function SearchListingCard({ row }: { row: SearchResultRow }) {
  const detailPath = sourceEntityPath({ canonicalId: row.canonical_id, kind: row.kind })
  const addressPath = buildAddressLookupPath({ address: row.address, city: row.city, postal: row.postal, source: row.source })
  return (
    <article className="search-card search-listing-card">
      <div className="search-card-header">
        <div className="search-card-badges">
          <span className={`search-badge search-badge--${row.source === 'shortcut' ? 'shortcut' : 'frontdoor'}`}>{sourceLabel(row.source)}</span>
          <span className="search-badge search-badge--kind">{row.kind}</span>
          <span className={`search-badge ${row.offering_id ? 'search-badge--linked' : 'search-badge--mode-listings'}`}>{row.offering_id ? 'Grouped' : 'Raw only'}</span>
          {row.housing_company_id && <span className="search-badge search-badge--company">Company</span>}
          {row.link_status && <span className="search-badge search-badge--kind">{row.link_status}</span>}
          {row.price_match_status && <span className="search-badge search-badge--price">Price match</span>}
          {Boolean(row.insight_count) && <span className="search-badge search-badge--insight">{row.insight_count} insights</span>}
        </div>
        <h2 className="search-card-title">{row.headline || row.address || row.canonical_id}</h2>
      </div>
      <p className="search-card-location">{[row.address, row.postal, row.city].filter(Boolean).join(' ')}</p>
      <p className="search-card-sub">{[row.housing_company_name, row.canonical_id, row.native_id].filter(Boolean).join(' / ')}</p>
      <div className="search-listing-facts">
        <Fact label="Price" value={formatEUR(row.price)} />
        <Fact label="Area" value={formatArea(row.area)} />
        <Fact label="Layout" value={row.room_layout} />
        <Fact label="Seen" value={formatDate(row.last_seen_at)} />
        <Fact label="Link score" value={formatScore(row.link_score)} />
        <Fact label="Price match" value={formatPriceMatch(row)} />
        <Fact label="Insights" value={formatInsights(row)} />
      </div>
      <div className="search-listing-actions">
        {detailPath && <Link to={detailPath}>Source detail</Link>}
        {row.offering_id && <Link to={`/target/offering/${row.offering_id}`}>Grouped offering</Link>}
        {row.housing_company_id && <Link to={`/target/housing_company/${row.housing_company_id}`}>Housing company</Link>}
        {row.price_match_transaction_id && <Link to={`/matches?transaction=${row.price_match_transaction_id}`}>Price match</Link>}
        {addressPath && <Link to={addressPath}>Grouped by address</Link>}
        <LiveSourceLink available={row.external_url_available} url={row.url} />
      </div>
    </article>
  )
}

function GroupedSearchCard({ group }: { group: GroupedOfferingSearchRow }) {
  const addressPath = buildAddressLookupPath({ address: group.address, city: group.city, postal: group.postal })
  const sources = group.sources ?? []
  return (
    <article className="search-card search-listing-card">
      <div className="search-card-header">
        <div className="search-card-badges">
          <span className="search-badge search-badge--linked">Grouped offering</span>
          {sources.map(source => <span className={`search-badge search-badge--${source === 'shortcut' ? 'shortcut' : 'frontdoor'}`} key={source}>{sourceLabel(source)}</span>)}
          {group.housing_company_id && <span className="search-badge search-badge--company">Company</span>}
          {group.price_match_status && <span className="search-badge search-badge--price">Price match</span>}
          {Boolean(group.insight_count) && <span className="search-badge search-badge--insight">{group.insight_count} insights</span>}
        </div>
        <h2 className="search-card-title">{group.headline || group.address || group.offering_id}</h2>
      </div>
      <p className="search-card-location">{[group.address, group.postal, group.city].filter(Boolean).join(' ')}</p>
      <p className="search-card-sub">{[group.housing_company_name, `Offering ${group.offering_id}`, `${group.source_count} source records`].filter(Boolean).join(' / ')}</p>
      <div className="search-listing-facts">
        <Fact label="Price" value={formatEUR(group.price)} />
        <Fact label="Area" value={formatArea(group.area)} />
        <Fact label="Layout" value={group.room_layout} />
        <Fact label="Seen" value={formatDate(group.last_seen_at)} />
        <Fact label="Sources" value={sources.map(sourceLabel).join(' + ')} />
        <Fact label="Best match" value={formatGroupedPriceMatch(group)} />
        <Fact label="Insights" value={group.insight_count ? String(group.insight_count) : '-'} />
      </div>
      <div className="search-listing-actions">
        <Link to={`/target/offering/${group.offering_id}`}>Grouped offering</Link>
        {group.housing_company_id && <Link to={`/target/housing_company/${group.housing_company_id}`}>Housing company</Link>}
        {group.price_match_transaction_id && <Link to={`/matches?transaction=${group.price_match_transaction_id}`}>Price match</Link>}
        {addressPath && <Link to={addressPath}>Grouped by address</Link>}
      </div>
    </article>
  )
}

function Pagination({ body, urlParams, onChange }: { body: { page: number; page_size: number; total: number }; urlParams: URLSearchParams; onChange: (params: URLSearchParams) => void }) {
  const totalPages = Math.max(1, Math.ceil(body.total / body.page_size))
  function go(page: number) {
    const next = new URLSearchParams(urlParams)
    if (page <= 1) next.delete('page')
    else next.set('page', String(page))
    onChange(next)
  }
  return (
    <div className="search-pagination">
      <button className="search-page-btn" disabled={body.page <= 1} type="button" onClick={() => go(body.page - 1)}>Previous</button>
      <span className="search-page-info">Page {body.page} / {totalPages}</span>
      <button className="search-page-btn" disabled={body.page >= totalPages} type="button" onClick={() => go(body.page + 1)}>Next</button>
    </div>
  )
}

function searchParamsFromURL(params: URLSearchParams): SearchParams {
  return {
    q: valueOrUndefined(params.get('q')),
    source: valueOrUndefined(params.get('source')),
    kind: valueOrUndefined(params.get('kind')),
    grouping: valueOrUndefined(params.get('grouping')),
    city: valueOrUndefined(params.get('city')),
    postal: valueOrUndefined(params.get('postal')),
    min_price: numberOrUndefined(params.get('min_price')),
    max_price: numberOrUndefined(params.get('max_price')),
    min_area: numberOrUndefined(params.get('min_area')),
    max_area: numberOrUndefined(params.get('max_area')),
    sort: params.get('sort') || 'seen_desc',
    page: numberOrUndefined(params.get('page')) ?? 1,
    page_size: 50,
  }
}

function searchViewPath(params: URLSearchParams, view: 'raw' | 'grouped') {
  const next = new URLSearchParams(params)
  if (view === 'raw') {
    next.delete('view')
  } else {
    next.set('view', 'grouped')
    next.set('grouping', 'grouped')
  }
  next.delete('page')
  const query = next.toString()
  return query ? `/search?${query}` : '/search'
}

function setIfPresent(params: URLSearchParams, key: string, value: string) {
  const trimmed = value.trim()
  if (trimmed) params.set(key, trimmed)
}

function numberOrUndefined(value: string | null): number | undefined {
  if (!value?.trim()) return undefined
  const parsed = Number(value)
  return Number.isFinite(parsed) ? parsed : undefined
}

function valueOrUndefined(value: string | null): string | undefined {
  return value?.trim() || undefined
}

function filterSummary(params: SearchParams) {
  const parts = [
    params.q ? `Query "${params.q}"` : 'All source listings',
    params.source ? sourceLabel(params.source) : '',
    params.kind ? params.kind : '',
    params.grouping ? params.grouping : '',
    params.city,
    params.postal,
  ].filter(Boolean)
  return parts.join(' / ')
}

function addressPathFromParams(params: SearchParams) {
  return buildAddressLookupPath({ address: params.q, city: params.city, postal: params.postal, source: params.source })
}

function sourceLabel(source?: string) {
  if (source === 'shortcut') return 'Shortcut'
  if (source === 'frontdoor') return 'Frontdoor'
  return source || 'Source'
}

function formatEUR(value?: number) {
  if (value == null) return '-'
  return `${new Intl.NumberFormat('fi-FI').format(value)} EUR`
}

function formatArea(value?: number) {
  if (value == null) return '-'
  return `${new Intl.NumberFormat('fi-FI', { maximumFractionDigits: 1 }).format(value)} m2`
}

function formatDate(value?: string) {
  if (!value) return '-'
  return new Intl.DateTimeFormat('fi-FI').format(new Date(value))
}

function formatScore(value?: number) {
  return value == null ? '-' : String(value)
}

function formatPriceMatch(row: SearchResultRow) {
  if (!row.price_match_status) return '-'
  return [row.price_match_status, formatEUR(row.price_match_price_eur)].filter(value => value && value !== '-').join(' / ')
}

function formatGroupedPriceMatch(row: GroupedOfferingSearchRow) {
  if (!row.price_match_status) return '-'
  return [row.price_match_status, formatEUR(row.price_match_price_eur)].filter(value => value && value !== '-').join(' / ')
}

function formatInsights(row: SearchResultRow) {
  if (!row.insight_count) return '-'
  return [String(row.insight_count), row.insight_top_severity].filter(Boolean).join(' / ')
}

function Fact({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value || '-'}</strong>
    </div>
  )
}
