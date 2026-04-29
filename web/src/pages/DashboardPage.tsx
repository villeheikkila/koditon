import { useState, useMemo } from 'react'
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

function fmt(n: number) {
  return new Intl.NumberFormat('fi-FI').format(n)
}

function avg(arr: number[]) {
  if (!arr.length) return 0
  return arr.reduce((a, b) => a + b, 0) / arr.length
}

export default function DashboardPage({ onSignOut }: { onSignOut: () => void }) {
  const [selectedMunicipality, setSelectedMunicipality] = useState('')
  const [showPasskeyManager, setShowPasskeyManager] = useState(false)
  const [selectedPostalCodes, setSelectedPostalCodes] = useState<Set<string>>(new Set())
  const [selectedCategories, setSelectedCategories] = useState<Set<string>>(new Set())
  const [selectedTypes, setSelectedTypes] = useState<Set<string>>(new Set())
  const [minArea, setMinArea] = useState('')
  const [maxArea, setMaxArea] = useState('')
  const [limit, setLimit] = useState(100)

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

  const params = useMemo(() => ({
    municipality_ids: selectedMunicipality || undefined,
    postal_code_ids: selectedPostalCodes.size > 0 ? [...selectedPostalCodes].join(',') : undefined,
    categories: selectedCategories.size > 0 ? [...selectedCategories].join(',') : undefined,
    types: selectedTypes.size > 0 ? [...selectedTypes].join(',') : undefined,
    min_area: minArea ? parseFloat(minArea) : undefined,
    max_area: maxArea ? parseFloat(maxArea) : undefined,
    limit,
  }), [selectedMunicipality, selectedPostalCodes, selectedCategories, selectedTypes, minArea, maxArea, limit])

  const { data: txRes, isPending, isError } = usePricesTransactionsFiltered(params, {
    query: { enabled: !!selectedMunicipality },
  })

  const transactions: PricesTransaction[] = useMemo(
    () => txRes?.status === 200 ? (txRes.data.transactions ?? []) : [],
    [txRes],
  )

  const stats = useMemo(() => ({
    count: transactions.length,
    avgPrice: avg(transactions.map(t => t.price)),
    avgPricePerSqm: avg(transactions.map(t => t.price_per_square_meter)),
    avgArea: avg(transactions.map(t => t.area)),
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
            >
              {LIMIT_OPTIONS.map(n => (
                <option key={n} value={n}>{n}</option>
              ))}
            </select>
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
                <div className="stat-label">Avg Area</div>
                <div className="stat-value">
                  {stats.avgArea > 0 ? stats.avgArea.toFixed(1) : '—'}
                </div>
                <div className="stat-sub">m²</div>
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
              <TransactionsTable transactions={transactions} />
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

function TransactionsTable({ transactions }: { transactions: PricesTransaction[] }) {
  return (
    <div className="table-wrap">
      <table>
        <thead>
          <tr>
            <th>Description</th>
            <th>Category</th>
            <th>Type</th>
            <th className="right">Area m²</th>
            <th className="right">Price €</th>
            <th className="right">€/m²</th>
            <th>Built</th>
            <th>Period</th>
            <th>Postal Code</th>
          </tr>
        </thead>
        <tbody>
          {transactions.map(t => (
            <tr key={t.id}>
              <td style={{ maxWidth: 220, overflow: 'hidden', textOverflow: 'ellipsis' }}>
                {t.description || <span className="dim">—</span>}
              </td>
              <td>
                <span className="badge badge-default">{t.category}</span>
              </td>
              <td className="muted">{t.type}</td>
              <td className="right mono">{t.area.toFixed(1)}</td>
              <td className="right mono">{fmt(t.price)}</td>
              <td className="right mono" style={{ color: 'var(--accent)' }}>
                {fmt(t.price_per_square_meter)}
              </td>
              <td className="dim">{t.build_year > 0 ? t.build_year : '—'}</td>
              <td className="dim">{t.period_identifier}</td>
              <td className="dim">{t.postal_code_code} {t.postal_code_name_fi}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  )
}
