import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { keepPreviousData } from '@tanstack/react-query'
import Nav from '../components/Nav'
import { useSaleListingsSearch, type PageSaleListingSummary, type SaleListingSummary, type SaleListingsSearchParams } from '../api/koditon'

const PAGE_SIZE = 25

const SOURCE_OPTIONS = [
  { value: '', label: 'All sources' },
  { value: 'shortcut', label: 'Shortcut' },
  { value: 'frontdoor', label: 'Frontdoor' },
]

const SORT_OPTIONS = [
  { value: 'seen_desc', label: 'Recently seen' },
  { value: 'price_asc', label: 'Price ↑' },
  { value: 'price_desc', label: 'Price ↓' },
  { value: 'area_asc', label: 'Area ↑' },
  { value: 'area_desc', label: 'Area ↓' },
]

function useDebounce<T>(value: T, delay: number): T {
  const [debounced, setDebounced] = useState(value)
  useEffect(() => {
    const id = setTimeout(() => setDebounced(value), delay)
    return () => clearTimeout(id)
  }, [value, delay])
  return debounced
}

function fmtPrice(n: number) {
  return new Intl.NumberFormat('fi-FI').format(n) + ' €'
}

export default function SearchPage() {
  const [urlParams, setUrlParams] = useSearchParams()

  // Read initial state from URL
  const [query, setQuery]     = useState(() => urlParams.get('q') ?? '')
  const [city, setCity]       = useState(() => urlParams.get('city') ?? '')
  const [postal, setPostal]   = useState(() => urlParams.get('postal') ?? '')
  const [source, setSource]   = useState(() => urlParams.get('source') ?? '')
  const [minPrice, setMinPrice] = useState(() => urlParams.get('min_price') ?? '')
  const [maxPrice, setMaxPrice] = useState(() => urlParams.get('max_price') ?? '')
  const [minArea, setMinArea]   = useState(() => urlParams.get('min_area') ?? '')
  const [maxArea, setMaxArea]   = useState(() => urlParams.get('max_area') ?? '')
  const [sort, setSort]       = useState(() => urlParams.get('sort') ?? 'seen_desc')
  const [page, setPage]       = useState(() => Number(urlParams.get('page') ?? '1'))

  const dQuery  = useDebounce(query, 300)
  const dCity   = useDebounce(city, 300)
  const dPostal = useDebounce(postal, 300)

  // Sync URL
  useEffect(() => {
    const p: Record<string, string> = {}
    if (dQuery)   p.q       = dQuery
    if (dCity)    p.city    = dCity
    if (dPostal)  p.postal  = dPostal
    if (source)   p.source  = source
    if (minPrice) p.min_price = minPrice
    if (maxPrice) p.max_price = maxPrice
    if (minArea)  p.min_area  = minArea
    if (maxArea)  p.max_area  = maxArea
    if (sort !== 'seen_desc') p.sort = sort
    if (page > 1) p.page = String(page)
    setUrlParams(p, { replace: true })
  }, [dQuery, dCity, dPostal, source, minPrice, maxPrice, minArea, maxArea, sort, page, setUrlParams])

  const hasFilters = !!(dQuery || dCity || dPostal || source || minPrice || maxPrice || minArea || maxArea)

  const params: SaleListingsSearchParams = {
    q:         dQuery   || undefined,
    city:      dCity    || undefined,
    postal:    dPostal  || undefined,
    source:    source   || undefined,
    min_price: minPrice ? Number(minPrice) : undefined,
    max_price: maxPrice ? Number(maxPrice) : undefined,
    min_area:  minArea  ? Number(minArea)  : undefined,
    max_area:  maxArea  ? Number(maxArea)  : undefined,
    sort,
    page,
    page_size: PAGE_SIZE,
  }

  const { data, isPending, isPlaceholderData } = useSaleListingsSearch(params, {
    query: {
      enabled: hasFilters,
      placeholderData: keepPreviousData,
      staleTime: 30_000,
    },
  })

  const pageData = data?.data as PageSaleListingSummary | undefined
  const rows = pageData?.rows ?? []
  const totalPages = pageData ? Math.ceil(pageData.total / PAGE_SIZE) : 0

  function clearAll() {
    setQuery(''); setCity(''); setPostal('')
    setSource('')
    setMinPrice(''); setMaxPrice(''); setMinArea(''); setMaxArea('')
    setSort('seen_desc'); setPage(1)
  }

  function updateFilter(setter: (value: string) => void, value: string) {
    setter(value)
    setPage(1)
  }

  return (
    <div className="search-layout">
      <Nav actions={
        pageData && hasFilters
          ? <span className="search-total">{pageData.total.toLocaleString('fi-FI')} results</span>
          : undefined
      } />

      <div className="search-body">
        {/* Sidebar filters */}
        <aside className="search-sidebar">
          <div className="search-filter-group">
            <label className="search-filter-label">Search</label>
            <input
              className="search-input"
              type="text"
              placeholder="Address, area, description…"
              value={query}
              onChange={e => updateFilter(setQuery, e.target.value)}
              autoFocus
            />
          </div>

          <div className="search-filter-row">
            <div className="search-filter-group">
              <label className="search-filter-label">City</label>
              <input
                className="search-input"
                type="text"
                placeholder="Helsinki"
                value={city}
                onChange={e => updateFilter(setCity, e.target.value)}
              />
            </div>
            <div className="search-filter-group">
              <label className="search-filter-label">Postal</label>
              <input
                className="search-input"
                type="text"
                placeholder="001…"
                value={postal}
                onChange={e => updateFilter(setPostal, e.target.value)}
              />
            </div>
          </div>

          <div className="search-filter-divider" />

          <div className="search-filter-row">
            <div className="search-filter-group">
              <label className="search-filter-label">Source</label>
              <select className="search-select" value={source} onChange={e => updateFilter(setSource, e.target.value)}>
                {SOURCE_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
              </select>
            </div>
          </div>

          <div className="search-filter-divider" />

          <div className="search-filter-group">
            <label className="search-filter-label">Price (€)</label>
            <div className="search-range-row">
              <input
                className="search-input"
                type="number"
                placeholder="Min"
                value={minPrice}
                onChange={e => updateFilter(setMinPrice, e.target.value)}
                min={0}
              />
              <span className="search-range-sep">–</span>
              <input
                className="search-input"
                type="number"
                placeholder="Max"
                value={maxPrice}
                onChange={e => updateFilter(setMaxPrice, e.target.value)}
                min={0}
              />
            </div>
          </div>

          <div className="search-filter-group">
            <label className="search-filter-label">Area (m²)</label>
            <div className="search-range-row">
              <input
                className="search-input"
                type="number"
                placeholder="Min"
                value={minArea}
                onChange={e => updateFilter(setMinArea, e.target.value)}
                min={0}
              />
              <span className="search-range-sep">–</span>
              <input
                className="search-input"
                type="number"
                placeholder="Max"
                value={maxArea}
                onChange={e => updateFilter(setMaxArea, e.target.value)}
                min={0}
              />
            </div>
          </div>

          <div className="search-filter-divider" />

          <div className="search-filter-group">
            <label className="search-filter-label">Sort by</label>
            <select className="search-select" value={sort} onChange={e => updateFilter(setSort, e.target.value)}>
              {SORT_OPTIONS.map(o => <option key={o.value} value={o.value}>{o.label}</option>)}
            </select>
          </div>

          {hasFilters && (
            <button className="search-clear-btn" onClick={clearAll}>
              Clear all filters
            </button>
          )}
        </aside>

        {/* Results */}
        <main className="search-results">
          {!hasFilters ? (
            <div className="search-empty-hint">
              <SearchIllustration />
              <p>Start typing to search listings</p>
              <p className="search-empty-sub">Search by address, description, city, or postal code</p>
            </div>
          ) : isPending && !isPlaceholderData ? (
            <div className="search-loading">
              <div className="spinner" />
            </div>
          ) : !pageData || rows.length === 0 ? (
            <div className="search-no-results">No listings found</div>
          ) : (
            <>
              <div className={`search-grid${isPlaceholderData ? ' search-grid--faded' : ''}`}>
                {rows.map(row => (
                  <SearchCard key={row.id} row={row} />
                ))}
              </div>

              {totalPages > 1 && (
                <div className="search-pagination">
                  <button
                    className="search-page-btn"
                    onClick={() => setPage(p => Math.max(1, p - 1))}
                    disabled={page <= 1}
                  >
                    ← Prev
                  </button>
                  <span className="search-page-info">
                    {page} / {totalPages}
                  </span>
                  <button
                    className="search-page-btn"
                    onClick={() => setPage(p => Math.min(totalPages, p + 1))}
                    disabled={page >= totalPages}
                  >
                    Next →
                  </button>
                </div>
              )}
            </>
          )}
        </main>
      </div>
    </div>
  )
}

function SearchCard({ row }: { row: SaleListingSummary }) {
  const navigate = useNavigate()
  const locationData = row.unit.location
  const title = locationData.street_address || row.headline || row.id
  const sub = [row.unit.room_layout, row.unit.area_m2 != null ? `${row.unit.area_m2.toFixed(1)} m²` : null]
    .filter(Boolean).join(' · ')
  const location = [locationData.postal, locationData.city].filter(Boolean).join(' ')
  const image = row.media?.main_image?.variants?.card || row.media?.main_image?.url

  return (
    <div
      className="search-card"
      onClick={() => navigate('/detail?' + new URLSearchParams({ id: row.id }))}
      role="button"
      tabIndex={0}
      onKeyDown={e => e.key === 'Enter' && navigate('/detail?' + new URLSearchParams({ id: row.id }))}
    >
      {image && <img className="search-card-image" src={image} alt="" loading="lazy" />}
      <div className="search-card-header">
        <div className="search-card-title">{title}</div>
        <div className="search-card-badges">
          <span className={`search-badge search-badge--${row.source.provider}`}>{row.source.provider}</span>
          {row.source.kind !== 'ad' && <span className="search-badge search-badge--kind">{row.source.kind}</span>}
        </div>
      </div>
      {sub && <div className="search-card-sub">{sub}</div>}
      {location && <div className="search-card-location">{location}</div>}
      {row.commercial.asking_price != null && (
        <div className="search-card-price">{fmtPrice(row.commercial.asking_price)}</div>
      )}
    </div>
  )
}

function SearchIllustration() {
  return (
    <svg className="search-empty-icon" width="40" height="40" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="1.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="11" cy="11" r="8" />
      <path d="m21 21-4.35-4.35" />
    </svg>
  )
}
