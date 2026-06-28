import { Link, useParams, useSearchParams } from 'react-router-dom'
import LiveSourceLink from '../components/LiveSourceLink'
import Nav from '../components/Nav'
import { useEntityDetail, type DetailFieldOutput, type EntityDetailOutputBody, type EntityInsightOutput, type EntityPriceMatchOutput } from '../api/koditon'
import { addressLookupInputFromParams, buildAddressLookupPath, sourceEntityPath, withAddressLookupContext, type AddressLookupInput } from '../lib/address-lookup'

type SourceEntityKind = 'listing' | 'rental' | 'housingCompany'

export default function SourceEntityPage({ kind }: { kind: SourceEntityKind }) {
  const { id = '' } = useParams()
  const [params] = useSearchParams()
  const detailQuery = useEntityDetail({ id }, { query: { enabled: Boolean(id) } })
  const detail = detailQuery.data?.status === 200 ? detailQuery.data.data : undefined
  const contextLookup = addressLookupInputFromParams(params)
  const contextLookupPath = buildAddressLookupPath(contextLookup)
  const lookupPath = contextLookupPath || (detail ? buildAddressLookupPath({ address: detail.street_address, city: detail.city, postal: detail.postal, source: detail.source }) : '')
  return (
    <main className="model-page">
      <Nav />
      <div className="model-shell">
        <header className="model-header">
          <div>
            <Link className="model-back" to={lookupPath || '/address'}>Address lookup</Link>
            <h1>{labelKind(kind)}</h1>
            <p>{id}</p>
          </div>
          <div className="source-entity-actions">
            {detail?.offering_id && <Link to={withAddressLookupContext(`/target/offering/${detail.offering_id}`, contextLookup)}>Grouped offering</Link>}
            {lookupPath && <Link to={lookupPath}>Address lookup</Link>}
            <LiveSourceLink available={detail?.external_url_available} url={detail?.url} />
          </div>
        </header>
        {detailQuery.isLoading && <div className="loading-state">Loading source detail</div>}
        {detailQuery.isError && <div className="error-state">Source detail could not be loaded.</div>}
        {detail && <SourceEntityDetail detail={detail} contextLookup={contextLookup} />}
      </div>
    </main>
  )
}

function SourceEntityDetail({ detail, contextLookup }: { detail: EntityDetailOutputBody; contextLookup: AddressLookupInput }) {
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
    ['Plot', detail.plot_type],
    ['Elevator', formatBool(detail.elevator)],
    ['Sauna', formatBool(detail.sauna)],
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
        <GroupedContext detail={detail} contextLookup={contextLookup} />
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
        <ComputedMetadata priceMatch={detail.price_match} insights={detail.insights} />
        <FieldGroup title="Canonical extra" fields={detail.canonical_extra ?? []} />
        <FieldGroup title="Source specific" fields={detail.source_specific ?? []} />
        <FieldGroup title="Related" fields={detail.related ?? []} />
      </aside>
    </div>
  )
}

function GroupedContext({ detail, contextLookup }: { detail: EntityDetailOutputBody; contextLookup: AddressLookupInput }) {
  const records = detail.source_records ?? []
  if (!detail.offering_id && records.length === 0) return null
  return (
    <section className="model-panel">
      <header>
        <h2>Grouped Offering</h2>
        {detail.source_count != null && <span>{detail.source_count}</span>}
      </header>
      <div className="source-group-context">
        {detail.offering_id && (
          <Link className="source-group-target" to={withAddressLookupContext(`/target/offering/${detail.offering_id}`, contextLookup)}>
            <span>Canonical offering</span>
            <strong>{detail.offering_id}</strong>
          </Link>
        )}
        {records.length > 0 && (
          <div className="source-group-records">
            {records.map(record => {
              const path = withAddressLookupContext(sourceEntityPath({ canonicalId: record.canonical_id, kind: record.kind }), contextLookup)
              return (
                <div className="source-group-record" key={record.listing_id}>
                  <div>
                    <span>{record.source} / {record.kind}</span>
                    <strong>{record.headline || record.native_id}</strong>
                    <small>{[record.address, record.city, record.postal].filter(Boolean).join(' / ')}</small>
                  </div>
                  <div>
                    <strong>{formatCurrency(record.asking_price)}</strong>
                    <small>{[formatNumber(record.area, 'm2'), formatDate(record.last_seen_at), record.link_status, record.price_match?.status].filter(Boolean).join(' / ')}</small>
                    {record.insights?.length ? <small>{record.insights.length} insights</small> : null}
                  </div>
                  <div className="source-group-record-actions">
                    {record.price_match && <Link to={`/matches?transaction=${record.price_match.transaction_id}`}>Price match</Link>}
                    {path && <Link to={path}>Source detail</Link>}
                    <LiveSourceLink available={record.external_url_available} url={record.url} />
                  </div>
                </div>
              )
            })}
          </div>
        )}
      </div>
    </section>
  )
}

function ComputedMetadata({ priceMatch, insights }: { priceMatch?: EntityPriceMatchOutput; insights?: EntityInsightOutput[] | null }) {
  const visibleInsights = insights?.slice(0, 5) ?? []
  if (!priceMatch && visibleInsights.length === 0) {
    return null
  }
  return (
    <section className="model-panel">
      <header><h2>Computed metadata</h2></header>
      {priceMatch && (
        <div className="source-computed-match">
          <div>
            <span>{[priceMatch.scope, priceMatch.status, priceMatch.method].filter(Boolean).join(' / ')}</span>
            <strong>{formatCurrency(priceMatch.price_eur)}</strong>
            <small>{[priceMatch.description, priceMatch.category, priceMatch.type, formatDate(priceMatch.updated_at)].filter(Boolean).join(' / ')}</small>
          </div>
          <Link to={`/matches?transaction=${priceMatch.transaction_id}`}>Price match</Link>
        </div>
      )}
      {visibleInsights.length > 0 && <InsightList insights={visibleInsights} total={insights?.length ?? visibleInsights.length} />}
    </section>
  )
}

function InsightList({ insights, total }: { insights: EntityInsightOutput[]; total: number }) {
  return (
    <div className="source-insight-list">
      {insights.map(insight => (
        <span key={`${insight.source_field}:${insight.key}:${insight.value}`}>
          <strong>{[insight.severity, insight.key].filter(Boolean).join(' / ')}</strong>
          {insight.value}
        </span>
      ))}
      {total > insights.length && <span>{total - insights.length} more</span>}
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

function formatBool(value?: boolean) {
  if (value == null) return ''
  return value ? 'Yes' : 'No'
}

function formatBytes(value?: number) {
  if (value == null) return ''
  return `${new Intl.NumberFormat('fi-FI').format(value)} bytes`
}
