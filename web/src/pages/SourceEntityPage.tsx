import { Link, useParams } from 'react-router-dom'
import Nav from '../components/Nav'
import { useEntityDetail, type DetailFieldOutput, type EntityDetailOutputBody } from '../api/koditon'
import { buildAddressLookupPath } from '../lib/address-lookup'

type SourceEntityKind = 'listing' | 'rental' | 'housingCompany'

export default function SourceEntityPage({ kind }: { kind: SourceEntityKind }) {
  const { id = '' } = useParams()
  const detailQuery = useEntityDetail({ id }, { query: { enabled: Boolean(id) } })
  const detail = detailQuery.data?.status === 200 ? detailQuery.data.data : undefined
  const lookupPath = detail ? buildAddressLookupPath({ address: detail.street_address, city: detail.city, postal: detail.postal }) : ''
  return (
    <main className="model-page">
      <Nav />
      <div className="model-shell">
        <header className="model-header">
          <div>
            <Link className="model-back" to="/address">Address lookup</Link>
            <h1>{labelKind(kind)}</h1>
            <p>{id}</p>
          </div>
          <div className="source-entity-actions">
            {lookupPath && <Link to={lookupPath}>Address lookup</Link>}
            {detail?.external_url_available && detail.url && <a href={detail.url} target="_blank" rel="noreferrer">Source page</a>}
          </div>
        </header>
        {detailQuery.isLoading && <div className="loading-state">Loading source detail</div>}
        {detailQuery.isError && <div className="error-state">Source detail could not be loaded.</div>}
        {detail && <SourceEntityDetail detail={detail} />}
      </div>
    </main>
  )
}

function SourceEntityDetail({ detail }: { detail: EntityDetailOutputBody }) {
  const primaryRows: Array<[string, string | number | undefined]> = [
    ['Canonical ID', detail.canonical_id],
    ['Source', detail.source],
    ['Kind', detail.kind],
    ['Native ID', detail.native_id],
    ['Address', detail.street_address],
    ['City', detail.city],
    ['Postal', detail.postal],
    ['Last seen', formatDate(detail.last_seen_at)],
  ]
  const propertyRows: Array<[string, string | number | undefined]> = [
    ['Room layout', detail.room_layout],
    ['Area', formatNumber(detail.area_m2, 'm2')],
    ['Rooms', detail.rooms_count],
    ['Floor', detail.floor_level],
    ['Floors', detail.total_floors],
    ['Build year', detail.build_year],
    ['Condition', detail.condition],
    ['Energy class', detail.energy_class],
  ]
  const priceRows: Array<[string, string | number | undefined]> = [
    ['Asking price', formatCurrency(detail.asking_price)],
    ['Debt-free price', formatCurrency(detail.debt_free_price)],
    ['Debt share', formatCurrency(detail.debt_share_amount)],
    ['Price per m2', formatNumber(detail.price_per_m2, 'EUR/m2')],
    ['Maintenance', formatNumber(detail.maintenance_charge_monthly, 'EUR/mo')],
    ['Total charge', formatNumber(detail.total_charge_monthly, 'EUR/mo')],
    ['Water charge', formatNumber(detail.water_charge, 'EUR/mo')],
  ]
  return (
    <div className="model-grid">
      <section className="model-column model-column--main">
        <section className="model-panel">
          <header><h2>{detail.headline || detail.canonical_id}</h2></header>
          <DetailRows rows={primaryRows} />
        </section>
        <section className="model-panel">
          <header><h2>Property</h2></header>
          <DetailRows rows={propertyRows} />
        </section>
        <TextSections detail={detail} />
        <RawPayload detail={detail} />
      </section>
      <aside className="model-column">
        <section className="model-panel">
          <header><h2>Pricing</h2></header>
          <DetailRows rows={priceRows} />
        </section>
        <FieldGroup title="Canonical extra" fields={detail.canonical_extra ?? []} />
        <FieldGroup title="Source specific" fields={detail.source_specific ?? []} />
        <FieldGroup title="Related" fields={detail.related ?? []} />
      </aside>
    </div>
  )
}

function RawPayload({ detail }: { detail: EntityDetailOutputBody }) {
  if (!detail.raw?.pretty) return null
  return (
    <section className="model-panel">
      <header><h2>Raw source payload</h2><span>{formatBytes(detail.raw.original_bytes)}</span></header>
      <pre className="source-raw-payload">{detail.raw.pretty}</pre>
    </section>
  )
}

function FieldGroup({ title, fields }: { title: string; fields: DetailFieldOutput[] }) {
  return (
    <section className="model-panel">
      <header><h2>{title}</h2><span>{fields.length}</span></header>
      {fields.length === 0 ? <p className="model-empty">No fields.</p> : <DetailRows rows={fields.map(field => [field.label, field.value])} />}
    </section>
  )
}

function TextSections({ detail }: { detail: EntityDetailOutputBody }) {
  const sections = [
    ['Description', detail.description_text],
    ['Availability', detail.availability_text],
    ['Renovations done', detail.renovations_done_text],
    ['Renovations planned', detail.renovations_planned_text],
    ['Additional info', detail.additional_info_text],
    ['Charges', detail.charges_text],
  ].filter(([, value]) => typeof value === 'string' && value.trim())
  if (sections.length === 0) return null
  return (
    <section className="model-panel">
      <header><h2>Source texts</h2></header>
      <div className="source-text-section-list">
        {sections.map(([label, value]) => (
          <div key={label}>
            <strong>{label}</strong>
            <p>{value}</p>
          </div>
        ))}
      </div>
    </section>
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

function labelKind(kind: SourceEntityKind) {
  if (kind === 'housingCompany') return 'Source housing company'
  if (kind === 'rental') return 'Source rental'
  return 'Source listing'
}

function formatCurrency(value?: number) {
  if (value == null) return ''
  return `${new Intl.NumberFormat('fi-FI').format(value)} EUR`
}

function formatNumber(value?: number, unit?: string) {
  if (value == null) return ''
  return `${new Intl.NumberFormat('fi-FI', { maximumFractionDigits: 1 }).format(value)}${unit ? ` ${unit}` : ''}`
}

function formatDate(value?: string) {
  if (!value) return ''
  return new Intl.DateTimeFormat('fi-FI').format(new Date(value))
}

function formatBytes(value?: number) {
  if (value == null) return ''
  return `${new Intl.NumberFormat('fi-FI').format(value)} bytes`
}
