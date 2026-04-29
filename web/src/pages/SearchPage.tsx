import { useState, useEffect } from 'react'
import { useNavigate, useSearchParams } from 'react-router-dom'
import { keepPreviousData } from '@tanstack/react-query'
import Nav from '../components/Nav'
import {
  useSaleListingsSearch,
  type PageSaleListingSummary,
  type SaleListingSummary,
  type SaleListingsSearchParams,
} from '../api/koditon'

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
  { value: 'price_m2_asc', label: 'Price / m² ↑' },
  { value: 'price_m2_desc', label: 'Price / m² ↓' },
  { value: 'build_year_desc', label: 'Year built ↓' },
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

function fmtPricePerM2(n: number) {
  return new Intl.NumberFormat('fi-FI', { maximumFractionDigits: 0 }).format(n) + ' €/m²'
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
  const [minPriceM2, setMinPriceM2] = useState(() => urlParams.get('min_price_per_m2') ?? '')
  const [maxPriceM2, setMaxPriceM2] = useState(() => urlParams.get('max_price_per_m2') ?? '')
  const [rooms, setRooms] = useState(() => urlParams.get('rooms') ?? '')
  const [floor, setFloor] = useState(() => urlParams.get('floor') ?? '')
  const [minBuildYear, setMinBuildYear] = useState(() => urlParams.get('min_build_year') ?? '')
  const [maxBuildYear, setMaxBuildYear] = useState(() => urlParams.get('max_build_year') ?? '')
  const [condition, setCondition] = useState(() => urlParams.get('condition') ?? '')
  const [energyClass, setEnergyClass] = useState(() => urlParams.get('energy_class') ?? '')
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
    if (minPriceM2) p.min_price_per_m2 = minPriceM2
    if (maxPriceM2) p.max_price_per_m2 = maxPriceM2
    if (rooms) p.rooms = rooms
    if (floor) p.floor = floor
    if (minBuildYear) p.min_build_year = minBuildYear
    if (maxBuildYear) p.max_build_year = maxBuildYear
    if (condition) p.condition = condition
    if (energyClass) p.energy_class = energyClass
    if (sort !== 'seen_desc') p.sort = sort
    if (page > 1) p.page = String(page)
    setUrlParams(p, { replace: true })
  }, [dQuery, dCity, dPostal, source, minPrice, maxPrice, minArea, maxArea, minPriceM2, maxPriceM2, rooms, floor, minBuildYear, maxBuildYear, condition, energyClass, sort, page, setUrlParams])

  const hasFilters = !!(dQuery || dCity || dPostal || source || minPrice || maxPrice || minArea || maxArea || minPriceM2 || maxPriceM2 || rooms || floor || minBuildYear || maxBuildYear || condition || energyClass)

  const params: SaleListingsSearchParams = {
    q:         dQuery   || undefined,
    city:      dCity    || undefined,
    postal:    dPostal  || undefined,
    source:    source   || undefined,
    min_price: minPrice ? Number(minPrice) : undefined,
    max_price: maxPrice ? Number(maxPrice) : undefined,
    min_area:  minArea  ? Number(minArea)  : undefined,
    max_area:  maxArea  ? Number(maxArea)  : undefined,
    min_price_per_m2: minPriceM2 ? Number(minPriceM2) : undefined,
    max_price_per_m2: maxPriceM2 ? Number(maxPriceM2) : undefined,
    rooms: rooms ? Number(rooms) : undefined,
    floor: floor ? Number(floor) : undefined,
    min_build_year: minBuildYear ? Number(minBuildYear) : undefined,
    max_build_year: maxBuildYear ? Number(maxBuildYear) : undefined,
    condition: condition || undefined,
    energy_class: energyClass || undefined,
    sort,
    page,
    page_size: PAGE_SIZE,
  }

  const saleSearch = useSaleListingsSearch(params, {
    query: {
      enabled: hasFilters,
      placeholderData: keepPreviousData,
      staleTime: 30_000,
    },
  })

  const pageData = saleSearch.data?.data as PageSaleListingSummary | undefined
  const rows = pageData?.rows ?? []
  const totalPages = pageData ? Math.ceil(pageData.total / PAGE_SIZE) : 0
  const isPending = saleSearch.isPending
  const isPlaceholderData = saleSearch.isPlaceholderData

  function clearAll() {
    setQuery(''); setCity(''); setPostal('')
    setSource('')
    setMinPrice(''); setMaxPrice(''); setMinArea(''); setMaxArea('')
    setMinPriceM2(''); setMaxPriceM2(''); setRooms(''); setFloor('')
    setMinBuildYear(''); setMaxBuildYear(''); setCondition(''); setEnergyClass('')
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

          <div className="search-filter-group">
            <label className="search-filter-label">Price / m²</label>
            <div className="search-range-row">
              <input
                className="search-input"
                type="number"
                placeholder="Min"
                value={minPriceM2}
                onChange={e => updateFilter(setMinPriceM2, e.target.value)}
                min={0}
              />
              <span className="search-range-sep">–</span>
              <input
                className="search-input"
                type="number"
                placeholder="Max"
                value={maxPriceM2}
                onChange={e => updateFilter(setMaxPriceM2, e.target.value)}
                min={0}
              />
            </div>
          </div>

          <div className="search-filter-row">
            <div className="search-filter-group">
              <label className="search-filter-label">Rooms</label>
              <input
                className="search-input"
                type="number"
                placeholder="Any"
                value={rooms}
                onChange={e => updateFilter(setRooms, e.target.value)}
                min={0}
              />
            </div>
            <div className="search-filter-group">
              <label className="search-filter-label">Floor</label>
              <input
                className="search-input"
                type="number"
                placeholder="Any"
                value={floor}
                onChange={e => updateFilter(setFloor, e.target.value)}
                min={0}
              />
            </div>
          </div>

          <div className="search-filter-group">
            <label className="search-filter-label">Year built</label>
            <div className="search-range-row">
              <input
                className="search-input"
                type="number"
                placeholder="Min"
                value={minBuildYear}
                onChange={e => updateFilter(setMinBuildYear, e.target.value)}
                min={0}
              />
              <span className="search-range-sep">–</span>
              <input
                className="search-input"
                type="number"
                placeholder="Max"
                value={maxBuildYear}
                onChange={e => updateFilter(setMaxBuildYear, e.target.value)}
                min={0}
              />
            </div>
          </div>

          <div className="search-filter-row">
            <div className="search-filter-group">
              <label className="search-filter-label">Condition</label>
              <input
                className="search-input"
                type="text"
                placeholder="Good"
                value={condition}
                onChange={e => updateFilter(setCondition, e.target.value)}
              />
            </div>
            <div className="search-filter-group">
              <label className="search-filter-label">Energy</label>
              <input
                className="search-input"
                type="text"
                placeholder="C"
                value={energyClass}
                onChange={e => updateFilter(setEnergyClass, e.target.value)}
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
              <p>Start typing to search sale listings</p>
              <p className="search-empty-sub">Search by address, description, city, or postal code</p>
            </div>
          ) : isPending && !isPlaceholderData ? (
            <div className="search-loading">
              <div className="spinner" />
            </div>
          ) : !pageData || rows.length === 0 ? (
            <div className="search-no-results">No sale listings found</div>
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
  const sub = [row.unit.room_layout, row.unit.area_m2 != null ? `${row.unit.area_m2.toFixed(1)} m²` : null, row.unit.rooms_count != null ? `${row.unit.rooms_count} rooms` : null]
    .filter(Boolean).join(' · ')
  const facts = [
    row.commercial.price_per_m2 != null ? fmtPricePerM2(row.commercial.price_per_m2) : null,
    row.unit.floor_level != null ? `Floor ${row.unit.floor_level}` : null,
    row.building.build_year != null ? `Built ${row.building.build_year}` : null,
    row.unit.condition || null,
    row.building.energy_class ? `Energy ${row.building.energy_class}` : null,
  ].filter(Boolean).join(' · ')
  const location = [locationData.postal, locationData.city].filter(Boolean).join(' ')
  const image = row.media?.main_image?.variants?.card || row.media?.main_image?.url

  const price = row.commercial.asking_price
  const path = `/listing/${encodeURIComponent(row.id)}`

  return (
    <div
      className="search-card"
      onClick={() => navigate(path)}
      role="button"
      tabIndex={0}
      onKeyDown={e => e.key === 'Enter' && navigate(path)}
    >
      {image && <img className="search-card-image" src={image} alt="" loading="lazy" />}
      <div className="search-card-header">
        <div className="search-card-title">{title}</div>
        <div className="search-card-badges">
          <span className={`search-badge search-badge--${row.source.provider}`}>{row.source.provider}</span>
          <span className="search-badge search-badge--mode-listings">sale</span>
          {row.source.kind !== 'ad' && <span className="search-badge search-badge--kind">{row.source.kind}</span>}
        </div>
      </div>
      {sub && <div className="search-card-sub">{sub}</div>}
      {facts && <div className="search-card-sub">{facts}</div>}
      {location && <div className="search-card-location">{location}</div>}
      {price != null && (
        <div className="search-card-price">{fmtPrice(price)}</div>
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
