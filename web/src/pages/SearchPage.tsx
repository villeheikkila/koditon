import { useState, useEffect } from 'react'
import { Link, useNavigate, useSearchParams } from 'react-router-dom'
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

const KIND_OPTIONS = [
  { value: '', label: 'All listing kinds' },
  { value: 'ad', label: 'Full ads only' },
  { value: 'announcement', label: 'Announcements only' },
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

type SearchFilterValues = {
  query: string
  city: string
  postal: string
  source: string
  listingKind: string
  minPrice: string
  maxPrice: string
  minArea: string
  maxArea: string
  minPriceM2: string
  maxPriceM2: string
  rooms: string
  floor: string
  minBuildYear: string
  maxBuildYear: string
  condition: string
  energyClass: string
  sort: string
}

const EMPTY_SEARCH_FILTERS: SearchFilterValues = {
  query: '',
  city: '',
  postal: '',
  source: '',
  listingKind: '',
  minPrice: '',
  maxPrice: '',
  minArea: '',
  maxArea: '',
  minPriceM2: '',
  maxPriceM2: '',
  rooms: '',
  floor: '',
  minBuildYear: '',
  maxBuildYear: '',
  condition: '',
  energyClass: '',
  sort: 'seen_desc',
}

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
  const [query, setQuery]     = useState(() => urlParams.get('q') ?? '')
  const [city, setCity]       = useState(() => urlParams.get('city') ?? '')
  const [postal, setPostal]   = useState(() => urlParams.get('postal') ?? '')
  const [source, setSource]   = useState(() => urlParams.get('source') ?? '')
  const [listingKind, setListingKind] = useState(() => urlParams.get('kind') ?? '')
  const [minPrice, setMinPrice] = useState(() => urlParams.get('min_price') ?? '')
  const [maxPrice, setMaxPrice] = useState(() => urlParams.get('max_price') ?? '')
  const [minArea, setMinArea]   = useState(() => urlParams.get('min_area') ?? '')
  const [maxArea, setMaxArea]   = useState(() => urlParams.get('max_area') ?? '')
  const [minPriceM2, setMinPriceM2] = useState(() => urlParams.get('min_price_m2') ?? urlParams.get('min_price_per_m2') ?? '')
  const [maxPriceM2, setMaxPriceM2] = useState(() => urlParams.get('max_price_m2') ?? urlParams.get('max_price_per_m2') ?? '')
  const [rooms, setRooms] = useState(() => urlParams.get('rooms') ?? '')
  const [floor, setFloor] = useState(() => urlParams.get('floor') ?? '')
  const [minBuildYear, setMinBuildYear] = useState(() => urlParams.get('min_build_year') ?? '')
  const [maxBuildYear, setMaxBuildYear] = useState(() => urlParams.get('max_build_year') ?? '')
  const [condition, setCondition] = useState(() => urlParams.get('condition') ?? '')
  const [energyClass, setEnergyClass] = useState(() => urlParams.get('energy_class') ?? '')
  const [sort, setSort]       = useState(() => urlParams.get('sort') ?? 'seen_desc')
  const [page, setPage]       = useState(() => Number(urlParams.get('page') ?? '1'))
  const [filtersOpen, setFiltersOpen] = useState(false)

  const dQuery  = useDebounce(query, 300)
  const dCity   = useDebounce(city, 300)
  const dPostal = useDebounce(postal, 300)

  useEffect(() => {
    const p: Record<string, string> = {}
    if (dQuery)   p.q       = dQuery
    if (dCity)    p.city    = dCity
    if (dPostal)  p.postal  = dPostal
    if (source)   p.source  = source
    if (listingKind) p.kind = listingKind
    if (minPrice) p.min_price = minPrice
    if (maxPrice) p.max_price = maxPrice
    if (minArea)  p.min_area  = minArea
    if (maxArea)  p.max_area  = maxArea
    if (minPriceM2) p.min_price_m2 = minPriceM2
    if (maxPriceM2) p.max_price_m2 = maxPriceM2
    if (rooms) p.rooms = rooms
    if (floor) p.floor = floor
    if (minBuildYear) p.min_build_year = minBuildYear
    if (maxBuildYear) p.max_build_year = maxBuildYear
    if (condition) p.condition = condition
    if (energyClass) p.energy_class = energyClass
    if (sort !== 'seen_desc') p.sort = sort
    if (page > 1) p.page = String(page)
    setUrlParams(p, { replace: true })
  }, [dQuery, dCity, dPostal, source, listingKind, minPrice, maxPrice, minArea, maxArea, minPriceM2, maxPriceM2, rooms, floor, minBuildYear, maxBuildYear, condition, energyClass, sort, page, setUrlParams])
  useEffect(() => {
    if (!filtersOpen) return
    const onKeyDown = (event: KeyboardEvent) => {
      if (event.key === 'Escape') setFiltersOpen(false)
    }
    window.addEventListener('keydown', onKeyDown)
    return () => window.removeEventListener('keydown', onKeyDown)
  }, [filtersOpen])

  const hasFilters = !!(dQuery || dCity || dPostal || source || listingKind || minPrice || maxPrice || minArea || maxArea || minPriceM2 || maxPriceM2 || rooms || floor || minBuildYear || maxBuildYear || condition || energyClass)
  const activeFilterCount = [dQuery, dCity, dPostal, source, listingKind, minPrice, maxPrice, minArea, maxArea, minPriceM2, maxPriceM2, rooms, floor, minBuildYear, maxBuildYear, condition, energyClass].filter(Boolean).length
  const activeFilterSummary = [
    dQuery && `Search "${dQuery}"`,
    dCity && `City ${dCity}`,
    dPostal && `Postal ${dPostal}`,
    source && SOURCE_OPTIONS.find(option => option.value === source)?.label,
    listingKind && KIND_OPTIONS.find(option => option.value === listingKind)?.label,
    minPrice || maxPrice ? `Price ${minPrice || '0'}-${maxPrice || 'any'}` : null,
    minArea || maxArea ? `Area ${minArea || '0'}-${maxArea || 'any'}` : null,
    rooms && `${rooms} rooms`,
  ].filter(Boolean).join(' · ')

  const params: SaleListingsSearchParams = {
    q:         dQuery   || undefined,
    city:      dCity    || undefined,
    postal:    dPostal  || undefined,
    source:    source   || undefined,
    kind:      listingKind || undefined,
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
  const mapPath = `/map${urlParams.toString() ? `?${urlParams.toString()}` : ''}`
  const currentFilters: SearchFilterValues = {
    query,
    city,
    postal,
    source,
    listingKind,
    minPrice,
    maxPrice,
    minArea,
    maxArea,
    minPriceM2,
    maxPriceM2,
    rooms,
    floor,
    minBuildYear,
    maxBuildYear,
    condition,
    energyClass,
    sort,
  }

  function clearAll() {
    setQuery(''); setCity(''); setPostal('')
    setSource(''); setListingKind('')
    setMinPrice(''); setMaxPrice(''); setMinArea(''); setMaxArea('')
    setMinPriceM2(''); setMaxPriceM2(''); setRooms(''); setFloor('')
    setMinBuildYear(''); setMaxBuildYear(''); setCondition(''); setEnergyClass('')
    setSort('seen_desc'); setPage(1)
  }

  function applyFilters(next: SearchFilterValues) {
    setQuery(next.query)
    setCity(next.city)
    setPostal(next.postal)
    setSource(next.source)
    setListingKind(next.listingKind)
    setMinPrice(next.minPrice)
    setMaxPrice(next.maxPrice)
    setMinArea(next.minArea)
    setMaxArea(next.maxArea)
    setMinPriceM2(next.minPriceM2)
    setMaxPriceM2(next.maxPriceM2)
    setRooms(next.rooms)
    setFloor(next.floor)
    setMinBuildYear(next.minBuildYear)
    setMaxBuildYear(next.maxBuildYear)
    setCondition(next.condition)
    setEnergyClass(next.energyClass)
    setSort(next.sort || 'seen_desc')
    setPage(1)
    setFiltersOpen(false)
  }

  return (
    <div className="search-layout">
      <Nav actions={
        pageData && hasFilters
          ? <span className="search-total">{pageData.total.toLocaleString('fi-FI')} results</span>
          : undefined
      } />
      {filtersOpen && (
        <SearchFiltersModal
          initialFilters={currentFilters}
          hasFilters={hasFilters}
          onClose={() => setFiltersOpen(false)}
          onApply={applyFilters}
        />
      )}
      <div className="search-body">
        <main className="search-results">
          <div className="search-filter-toolbar">
            <div className="search-view-switch">
              <span className="search-view-tab search-view-tab--active">List</span>
              <Link className="search-view-tab" to={mapPath}>Map</Link>
            </div>
            <button className="search-filter-trigger" type="button" onClick={() => setFiltersOpen(true)}>
              Filters{activeFilterCount > 0 ? ` (${activeFilterCount})` : ''}
            </button>
            <div className="search-filter-summary">
              {activeFilterSummary || 'No filters selected'}
            </div>
            {hasFilters && (
              <button className="search-clear-btn search-clear-btn--inline" type="button" onClick={clearAll}>
                Clear
              </button>
            )}
          </div>
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

type SearchFiltersModalProps = {
  initialFilters: SearchFilterValues
  hasFilters: boolean
  onClose: () => void
  onApply: (filters: SearchFilterValues) => void
}

function SearchFiltersModal(props: SearchFiltersModalProps) {
  const [draft, setDraft] = useState<SearchFilterValues>(props.initialFilters)
  const field = (key: keyof SearchFilterValues) => (event: { target: { value: string } }) => setDraft(current => ({ ...current, [key]: event.target.value }))
  const hasDraftFilters = Object.entries(draft).some(([key, value]) => key !== 'sort' && Boolean(value))
  function clearDraft() {
    setDraft(EMPTY_SEARCH_FILTERS)
  }
  return (
    <div className="modal-overlay" onClick={event => { if (event.target === event.currentTarget) props.onClose() }}>
      <div className="search-filter-modal" role="dialog" aria-modal="true" aria-labelledby="search-filter-title">
        <div className="search-filter-modal-head">
          <div>
            <h2 id="search-filter-title" className="search-filter-modal-title">Search filters</h2>
            <div className="search-filter-modal-subtitle">Shared by list and map views.</div>
          </div>
          <button className="transaction-modal-close" type="button" onClick={props.onClose} aria-label="Close">×</button>
        </div>
        <div className="search-filter-modal-body">
          <div className="search-filter-group search-filter-group--wide">
            <label className="search-filter-label">Search</label>
            <input className="search-input" type="text" placeholder="Address, area, description…" value={draft.query} onChange={field('query')} autoFocus />
          </div>
          <div className="search-filter-row">
            <div className="search-filter-group">
              <label className="search-filter-label">City</label>
              <input className="search-input" type="text" placeholder="Helsinki" value={draft.city} onChange={field('city')} />
            </div>
            <div className="search-filter-group">
              <label className="search-filter-label">Postal</label>
              <input className="search-input" type="text" placeholder="00100" value={draft.postal} onChange={field('postal')} />
            </div>
          </div>
          <div className="search-filter-row">
            <div className="search-filter-group">
              <label className="search-filter-label">Source</label>
              <select className="search-select" value={draft.source} onChange={field('source')}>
                {SOURCE_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </div>
            <div className="search-filter-group">
              <label className="search-filter-label">Kind</label>
              <select className="search-select" value={draft.listingKind} onChange={field('listingKind')}>
                {KIND_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </div>
          </div>
          <SearchRange label="Price (€)" min={draft.minPrice} max={draft.maxPrice} onMin={field('minPrice')} onMax={field('maxPrice')} />
          <SearchRange label="Area (m²)" min={draft.minArea} max={draft.maxArea} onMin={field('minArea')} onMax={field('maxArea')} />
          <SearchRange label="Price / m²" min={draft.minPriceM2} max={draft.maxPriceM2} onMin={field('minPriceM2')} onMax={field('maxPriceM2')} />
          <div className="search-filter-row">
            <div className="search-filter-group">
              <label className="search-filter-label">Rooms</label>
              <input className="search-input" type="number" placeholder="Any" value={draft.rooms} onChange={field('rooms')} min={0} />
            </div>
            <div className="search-filter-group">
              <label className="search-filter-label">Floor</label>
              <input className="search-input" type="number" placeholder="Any" value={draft.floor} onChange={field('floor')} min={0} />
            </div>
          </div>
          <SearchRange label="Year built" min={draft.minBuildYear} max={draft.maxBuildYear} onMin={field('minBuildYear')} onMax={field('maxBuildYear')} />
          <div className="search-filter-row">
            <div className="search-filter-group">
              <label className="search-filter-label">Condition</label>
              <input className="search-input" type="text" placeholder="Good" value={draft.condition} onChange={field('condition')} />
            </div>
            <div className="search-filter-group">
              <label className="search-filter-label">Energy</label>
              <input className="search-input" type="text" placeholder="C" value={draft.energyClass} onChange={field('energyClass')} />
            </div>
          </div>
          <div className="search-filter-group search-filter-group--wide">
            <label className="search-filter-label">Sort by</label>
            <select className="search-select" value={draft.sort} onChange={field('sort')}>
              {SORT_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
            </select>
          </div>
        </div>
        <div className="search-filter-modal-actions">
          <button className="search-clear-btn search-clear-btn--inline" type="button" onClick={clearDraft} disabled={!props.hasFilters && !hasDraftFilters}>Clear all</button>
          <button className="search-filter-trigger" type="button" onClick={() => props.onApply(draft)}>Done</button>
        </div>
      </div>
    </div>
  )
}

function SearchRange({ label, min, max, onMin, onMax }: { label: string; min: string; max: string; onMin: (event: { target: { value: string } }) => void; onMax: (event: { target: { value: string } }) => void }) {
  return (
    <div className="search-filter-group search-filter-group--wide">
      <label className="search-filter-label">{label}</label>
      <div className="search-range-row">
        <input className="search-input" type="number" placeholder="Min" value={min} onChange={onMin} min={0} />
        <span className="search-range-sep">–</span>
        <input className="search-input" type="number" placeholder="Max" value={max} onChange={onMax} min={0} />
      </div>
    </div>
  )
}

function SearchCard({ row }: { row: SaleListingSummary }) {
  const navigate = useNavigate()
  const sourceProviders = ((row as SaleListingSummary & { source_providers?: string[] }).source_providers?.length ? (row as SaleListingSummary & { source_providers?: string[] }).source_providers : [row.source.provider]) ?? [row.source.provider]
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
          {sourceProviders.map(provider => (
            <span key={provider} className={`search-badge search-badge--${provider}`}>{provider}</span>
          ))}
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
