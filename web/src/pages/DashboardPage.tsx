import { useEffect, useState, useMemo } from 'react'
import { Link, useSearchParams } from 'react-router-dom'
import Nav from '../components/Nav'
import {
  useAvailabilityLocations,
  useAvailabilityCategories,
  useAvailabilityTypes,
  usePricesTransactionsFiltered,
  type PricesTransaction,
  type AvailableMunicipality,
  type AvailablePostalCode,
  type TranslatedValue,
} from '../api/koditon'
import PasskeyManager from '../components/PasskeyManager'

const LIMIT_OPTIONS = [50, 100, 200, 500]

type PricesTransactionRow = PricesTransaction & {
  is_matched?: boolean
  matched_listing_count?: number
  matched_offering_count?: number
}

type SortKey = 'is_matched' | 'created_at' | 'updated_at' | 'neighborhood_name' | 'description' | 'category' | 'type' | 'area' | 'price' | 'price_per_square_meter' | 'build_year' | 'floor' | 'elevator' | 'condition' | 'plot' | 'energy_class' | 'period_identifier' | 'postal_code_code' | 'municipality_name_fi'

type SortState = {
  key: SortKey
  direction: 'asc' | 'desc'
}

const PRICES_QUERY_KEYS = ['municipality', 'postals', 'categories', 'types', 'min_area', 'max_area', 'limit', 'sort', 'sort_dir']

function fmt(n: number) {
  return new Intl.NumberFormat('fi-FI').format(n)
}

function fmtDate(value: string) {
  return new Intl.DateTimeFormat('fi-FI', { year: 'numeric', month: '2-digit', day: '2-digit' }).format(new Date(value))
}

function avg(arr: number[]) {
  if (!arr.length) return 0
  return arr.reduce((a, b) => a + b, 0) / arr.length
}

export default function DashboardPage({ onSignOut }: { onSignOut: () => void }) {
  const [urlParams, setUrlParams] = useSearchParams()
  const [selectedMunicipality, setSelectedMunicipality] = useState(() => urlParams.get('municipality') ?? '')
  const [showPasskeyManager, setShowPasskeyManager] = useState(false)
  const [selectedPostalCodes, setSelectedPostalCodes] = useState<Set<string>>(() => parseSetParam(urlParams.get('postals')))
  const [selectedCategories, setSelectedCategories] = useState<Set<string>>(() => parseSetParam(urlParams.get('categories')))
  const [selectedTypes, setSelectedTypes] = useState<Set<string>>(() => parseSetParam(urlParams.get('types')))
  const [minArea, setMinArea] = useState(() => urlParams.get('min_area') ?? '')
  const [maxArea, setMaxArea] = useState(() => urlParams.get('max_area') ?? '')
  const [limit, setLimit] = useState(() => parseLimit(urlParams.get('limit')))
  const [sort, setSort] = useState<SortState>(() => parseSort(urlParams.get('sort'), urlParams.get('sort_dir')))

  const { data: locRes } = useAvailabilityLocations()
  const { data: catRes } = useAvailabilityCategories()
  const { data: typeRes } = useAvailabilityTypes()

  const municipalities: AvailableMunicipality[] = useMemo(
    () => locRes?.status === 200 ? (locRes.data.municipalities ?? []) : [],
    [locRes],
  )
  const allPostalCodes: AvailablePostalCode[] = useMemo(
    () => locRes?.status === 200 ? (locRes.data.postal_codes ?? []) : [],
    [locRes],
  )
  const categories: TranslatedValue[] = useMemo(
    () => catRes?.status === 200 ? (catRes.data.categories ?? []) : [],
    [catRes],
  )
  const types: TranslatedValue[] = useMemo(
    () => typeRes?.status === 200 ? (typeRes.data.types ?? []) : [],
    [typeRes],
  )

  const postalCodesForMunicipality = useMemo(() =>
    selectedMunicipality
      ? allPostalCodes.filter(pc => pc.municipality_id === selectedMunicipality)
      : [],
    [allPostalCodes, selectedMunicipality],
  )

  useEffect(() => {
    setUrlParams(prev => {
      const next = new URLSearchParams(prev)
      for (const key of PRICES_QUERY_KEYS) next.delete(key)
      if (selectedMunicipality) next.set('municipality', selectedMunicipality)
      setJoinedParam(next, 'postals', selectedPostalCodes)
      setJoinedParam(next, 'categories', selectedCategories)
      setJoinedParam(next, 'types', selectedTypes)
      if (minArea) next.set('min_area', minArea)
      if (maxArea) next.set('max_area', maxArea)
      if (limit !== 100) next.set('limit', String(limit))
      if (sort.key !== 'created_at') next.set('sort', sort.key)
      if (sort.direction !== defaultDirection(sort.key)) next.set('sort_dir', sort.direction)
      return next.toString() === prev.toString() ? prev : next
    }, { replace: true })
  }, [selectedMunicipality, selectedPostalCodes, selectedCategories, selectedTypes, minArea, maxArea, limit, sort, setUrlParams])

  const params = useMemo(() => ({
    municipality_ids: selectedMunicipality || undefined,
    postal_code_ids: selectedPostalCodes.size > 0 ? [...selectedPostalCodes].join(',') : undefined,
    categories: selectedCategories.size > 0 ? [...selectedCategories].join(',') : undefined,
    types: selectedTypes.size > 0 ? [...selectedTypes].join(',') : undefined,
    min_area: minArea ? parseFloat(minArea) : undefined,
    max_area: maxArea ? parseFloat(maxArea) : undefined,
    limit: selectedPostalCodes.size > 0 ? undefined : limit,
  }), [selectedMunicipality, selectedPostalCodes, selectedCategories, selectedTypes, minArea, maxArea, limit])

  const { data: txRes, isPending, isError } = usePricesTransactionsFiltered(params, {
    query: { enabled: !!selectedMunicipality },
  })

  const transactions: PricesTransactionRow[] = useMemo(
    () => txRes?.status === 200 ? (txRes.data.transactions ?? []) : [],
    [txRes],
  )

  const stats = useMemo(() => ({
    count: transactions.length,
    avgPrice: avg(transactions.map(t => t.price)),
    avgPricePerSqm: avg(transactions.map(t => t.price_per_square_meter)),
    avgArea: avg(transactions.map(t => t.area)),
    matched: transactions.filter(t => t.is_matched).length,
  }), [transactions])

  function toggleSet<T>(set: Set<T>, value: T): Set<T> {
    const next = new Set(set)
    if (next.has(value)) {
      next.delete(value)
    } else {
      next.add(value)
    }
    return next
  }

  function onMunicipalityChange(id: string) {
    setSelectedMunicipality(id)
    setSelectedPostalCodes(new Set())
  }

  return (
    <div className="app-layout">
      <Nav actions={
        <>
          <button className="header-action-btn" onClick={() => setShowPasskeyManager(true)} title="Manage passkeys">
            <KeyIcon />
          </button>
          <button className="header-action-btn" onClick={onSignOut} title="Sign out">
            <SignOutIcon />
          </button>
        </>
      } />

      {showPasskeyManager && <PasskeyManager onClose={() => setShowPasskeyManager(false)} />}

      <div className="main-content">
        <aside className="sidebar">
          <div className="sidebar-section">
            <div className="sidebar-label">Municipality</div>
            <select
              className="filter-select"
              value={selectedMunicipality}
              onChange={e => onMunicipalityChange(e.target.value)}
            >
              <option value="">Select…</option>
              {municipalities.map(m => (
                <option key={m.id} value={m.id}>{m.name_fi}</option>
              ))}
            </select>
          </div>

          {postalCodesForMunicipality.length > 0 && (
            <div className="sidebar-section">
              <div className="sidebar-label">Postal Codes</div>
              <div className="checkbox-list">
                {postalCodesForMunicipality.map(pc => (
                  <label key={pc.id} className="checkbox-item">
                    <input
                      type="checkbox"
                      checked={selectedPostalCodes.has(pc.id)}
                      onChange={() => setSelectedPostalCodes(prev => toggleSet(prev, pc.id))}
                    />
                    <span className="checkbox-item-label">{pc.code} {pc.name_fi}</span>
                  </label>
                ))}
              </div>
            </div>
          )}

          {categories.length > 0 && (
            <div className="sidebar-section">
              <div className="sidebar-label">Category</div>
              <div className="checkbox-list">
                {categories.map(c => (
                  <label key={c.value} className="checkbox-item">
                    <input
                      type="checkbox"
                      checked={selectedCategories.has(c.value)}
                      onChange={() => setSelectedCategories(prev => toggleSet(prev, c.value))}
                    />
                    <span className="checkbox-item-label">{c.translation}</span>
                  </label>
                ))}
              </div>
            </div>
          )}

          {types.length > 0 && (
            <div className="sidebar-section">
              <div className="sidebar-label">Type</div>
              <div className="checkbox-list">
                {types.map(t => (
                  <label key={t.value} className="checkbox-item">
                    <input
                      type="checkbox"
                      checked={selectedTypes.has(t.value)}
                      onChange={() => setSelectedTypes(prev => toggleSet(prev, t.value))}
                    />
                    <span className="checkbox-item-label">{t.translation}</span>
                  </label>
                ))}
              </div>
            </div>
          )}

          <div className="sidebar-section">
            <div className="sidebar-label">Area (m²)</div>
            <div className="filter-row">
              <input
                type="number"
                className="filter-input"
                placeholder="Min"
                value={minArea}
                onChange={e => setMinArea(e.target.value)}
                min={0}
              />
              <span className="filter-row-sep">–</span>
              <input
                type="number"
                className="filter-input"
                placeholder="Max"
                value={maxArea}
                onChange={e => setMaxArea(e.target.value)}
                min={0}
              />
            </div>
          </div>

          <div className="sidebar-section">
            <div className="sidebar-label">Results</div>
            <select
              className="filter-select"
              value={limit}
              onChange={e => setLimit(Number(e.target.value))}
              disabled={selectedPostalCodes.size > 0}
            >
              {LIMIT_OPTIONS.map(n => (
                <option key={n} value={n}>{n}</option>
              ))}
            </select>
            {selectedPostalCodes.size > 0 && <div className="filter-help">All rows for selected postal codes</div>}
          </div>
        </aside>

        <main className="dashboard-body">
          {selectedMunicipality && (
            <div className="stats-grid">
              <div className="stat-card">
                <div className="stat-label">Transactions</div>
                <div className="stat-value accent">{fmt(stats.count)}</div>
                <div className="stat-sub">in results</div>
              </div>
              <div className="stat-card">
                <div className="stat-label">Avg Price</div>
                <div className="stat-value">
                  {stats.avgPrice > 0 ? fmt(Math.round(stats.avgPrice)) : '—'}
                </div>
                <div className="stat-sub">€</div>
              </div>
              <div className="stat-card">
                <div className="stat-label">Avg €/m²</div>
                <div className="stat-value">
                  {stats.avgPricePerSqm > 0 ? fmt(Math.round(stats.avgPricePerSqm)) : '—'}
                </div>
                <div className="stat-sub">per square meter</div>
              </div>
              <div className="stat-card">
                <div className="stat-label">Matched</div>
                <div className="stat-value">
                  {fmt(stats.matched)}
                </div>
                <div className="stat-sub">{stats.count > 0 ? `${Math.round((stats.matched / stats.count) * 100)}% of results` : 'of results'}</div>
              </div>
            </div>
          )}

          <div className="section-header">
            <span className="section-title">Transactions</span>
            {transactions.length > 0 && (
              <span className="section-count">{transactions.length} rows</span>
            )}
          </div>

          <div className="table-container">
            {!selectedMunicipality ? (
              <div className="empty-state">
                Select a municipality to explore price transactions
              </div>
            ) : isPending ? (
              <div className="loading-state"><div className="spinner" /></div>
            ) : isError ? (
              <div className="error-state">Failed to load transactions</div>
            ) : transactions.length === 0 ? (
              <div className="empty-state">No transactions found for the selected filters</div>
            ) : (
              <TransactionsTable transactions={transactions} sort={sort} onSort={setSort} />
            )}
          </div>
        </main>
      </div>
    </div>
  )
}

function KeyIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="7.5" cy="15.5" r="5.5" />
      <path d="m21 2-9.6 9.6" />
      <path d="m15.5 7.5 3 3L22 7l-3-3" />
    </svg>
  )
}

function SignOutIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M9 21H5a2 2 0 0 1-2-2V5a2 2 0 0 1 2-2h4" />
      <polyline points="16 17 21 12 16 7" />
      <line x1="21" y1="12" x2="9" y2="12" />
    </svg>
  )
}

function TransactionsTable({ transactions, sort, onSort }: { transactions: PricesTransactionRow[]; sort: SortState; onSort: (sort: SortState | ((prev: SortState) => SortState)) => void }) {
  const sortedTransactions = useMemo(() => [...transactions].sort((left, right) => compareTransactions(left, right, sort)), [transactions, sort])
  function sortBy(key: SortKey) {
    onSort(prev => prev.key === key ? { key, direction: prev.direction === 'asc' ? 'desc' : 'asc' } : { key, direction: defaultDirection(key) })
  }
  function th(label: string, key: SortKey, align?: 'right') {
    const active = sort.key === key
    return (
      <th className={align === 'right' ? 'right' : undefined}>
        <button className={`sortable-th ${active ? 'is-active' : ''}`} onClick={() => sortBy(key)} type="button">
          <span>{label}</span>
          <span className="sort-indicator">{active ? (sort.direction === 'asc' ? '↑' : '↓') : '↕'}</span>
        </button>
      </th>
    )
  }
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            {th('Match', 'is_matched')}
            {th('Seen', 'created_at')}
            {th('Updated', 'updated_at')}
            {th('Neighbourhood', 'neighborhood_name')}
            {th('Description', 'description')}
            {th('Category', 'category')}
            {th('Type', 'type')}
            {th('Area m²', 'area', 'right')}
            {th('Price €', 'price', 'right')}
            {th('€/m²', 'price_per_square_meter', 'right')}
            {th('Built', 'build_year')}
            {th('Floor', 'floor')}
            {th('Elevator', 'elevator')}
            {th('Condition', 'condition')}
            {th('Plot', 'plot')}
            {th('Energy', 'energy_class')}
            {th('Period', 'period_identifier')}
            {th('Postal Code', 'postal_code_code')}
            {th('Municipality', 'municipality_name_fi')}
          </tr>
        </thead>
        <tbody>
          {sortedTransactions.map(t => {
            const matchURL = `/matches?postal=${encodeURIComponent(t.postal_code_code)}&transaction=${encodeURIComponent(t.id)}`
            const matchedCount = (t.matched_listing_count ?? 0) + (t.matched_offering_count ?? 0)
            return (
              <tr key={t.id} className={`clickable-row ${t.is_matched ? 'matched-row' : ''}`}>
                <td>
                  <Link className="transaction-row-link" to={matchURL}>
                    <span className={`badge ${t.is_matched ? 'badge-match' : 'badge-default'}`}>{t.is_matched ? `Matched${matchedCount > 1 ? ` ${matchedCount}` : ''}` : 'Open'}</span>
                  </Link>
                </td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{fmtDate(t.created_at)}</Link></td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{fmtDate(t.updated_at)}</Link></td>
                <td><Link className="transaction-row-link" to={matchURL}>{t.neighborhood_name || '—'}</Link></td>
                <td style={{ maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                  <Link className="transaction-row-link" to={matchURL}>{t.description || '—'}</Link>
                </td>
                <td>
                  <Link className="transaction-row-link" to={matchURL}><span className="badge badge-default">{t.category}</span></Link>
                </td>
                <td className="muted"><Link className="transaction-row-link muted" to={matchURL}>{t.type}</Link></td>
                <td className="right mono"><Link className="transaction-row-link mono" to={matchURL}>{t.area.toFixed(1)}</Link></td>
                <td className="right mono"><Link className="transaction-row-link mono" to={matchURL}>{fmt(t.price)}</Link></td>
                <td className="right mono">
                  <Link className="transaction-row-link mono accent-link" to={matchURL}>{fmt(t.price_per_square_meter)}</Link>
                </td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{t.build_year > 0 ? t.build_year : '—'}</Link></td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{t.floor || '—'}</Link></td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{t.elevator ? 'Yes' : 'No'}</Link></td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{t.condition || '—'}</Link></td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{t.plot || '—'}</Link></td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{t.energy_class || '—'}</Link></td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{t.period_identifier}</Link></td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{t.postal_code_code} {t.postal_code_name_fi}</Link></td>
                <td className="dim"><Link className="transaction-row-link dim" to={matchURL}>{t.municipality_name_fi}</Link></td>
              </tr>
            )
          })}
        </tbody>
      </table>
    </div>
  )
}

function defaultDirection(key: SortKey): SortState['direction'] {
  return ['created_at', 'updated_at', 'price', 'price_per_square_meter', 'area', 'build_year', 'is_matched'].includes(key) ? 'desc' : 'asc'
}

function parseSetParam(value: string | null) {
  return new Set((value ?? '').split(',').map(item => item.trim()).filter(Boolean))
}

function setJoinedParam(params: URLSearchParams, key: string, values: Set<string>) {
  if (values.size > 0) params.set(key, [...values].sort().join(','))
}

function parseLimit(value: string | null) {
  const parsed = Number(value)
  return LIMIT_OPTIONS.includes(parsed) ? parsed : 100
}

function parseSort(key: string | null, direction: string | null): SortState {
  const sortKey = isSortKey(key) ? key : 'created_at'
  return { key: sortKey, direction: direction === 'asc' || direction === 'desc' ? direction : defaultDirection(sortKey) }
}

function isSortKey(value: string | null): value is SortKey {
  return value === 'is_matched' || value === 'created_at' || value === 'updated_at' || value === 'neighborhood_name' || value === 'description' || value === 'category' || value === 'type' || value === 'area' || value === 'price' || value === 'price_per_square_meter' || value === 'build_year' || value === 'floor' || value === 'elevator' || value === 'condition' || value === 'plot' || value === 'energy_class' || value === 'period_identifier' || value === 'postal_code_code' || value === 'municipality_name_fi'
}

function compareTransactions(left: PricesTransactionRow, right: PricesTransactionRow, sort: SortState) {
  const leftValue = sortValue(left, sort.key)
  const rightValue = sortValue(right, sort.key)
  const result = compareValues(leftValue, rightValue)
  return sort.direction === 'asc' ? result : -result
}

function sortValue(row: PricesTransactionRow, key: SortKey) {
  if (key === 'is_matched') return row.is_matched ? 1 : 0
  if (key === 'created_at' || key === 'updated_at') return Date.parse(row[key])
  return row[key] ?? ''
}

function compareValues(left: string | number | boolean, right: string | number | boolean) {
  if (typeof left === 'number' && typeof right === 'number') return left - right
  if (typeof left === 'boolean' && typeof right === 'boolean') return Number(left) - Number(right)
  return String(left).localeCompare(String(right), 'fi', { numeric: true, sensitivity: 'base' })
}
