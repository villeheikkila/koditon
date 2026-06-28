import { useMemo, useState, type ReactNode } from 'react'
import { Link, useParams, useSearchParams } from 'react-router-dom'
import LiveSourceLink from '../components/LiveSourceLink'
import Nav from '../components/Nav'
import {
  usePropertyDocumentsManagerCertificatesUpload,
  usePropertyTargetsClaims,
  usePropertyTargetsDetail,
  type CanonicalTargetResource,
  type PropertyDocumentSummary,
  type RenovationEvent,
  type ResolvedValue,
  type SourceClaim,
  type TargetBuildingSummary,
  type TargetOfferingSummary,
  type TargetSourceListing,
  type TargetUnitSummary,
  type TargetOverview,
} from '../api/koditon'
import { addressLookupInputFromParams, addressLookupPathFromOverviewFields, buildAddressLookupPath, sourceEntityPath, withAddressLookupContext } from '../lib/address-lookup'

type DetailKind = 'listing' | 'rental' | 'housingCompany'

function targetTypeForKind(kind: DetailKind): string {
  if (kind === 'housingCompany') return 'housing_company'
  return 'offering'
}

export default function DetailPage({ kind }: { kind?: DetailKind }) {
  const { id = '', targetType: targetTypeParam = '' } = useParams()
  const [params] = useSearchParams()
  const targetType = targetTypeParam || targetTypeForKind(kind ?? 'listing')
  const detailQuery = usePropertyTargetsDetail(targetType, id, { query: { enabled: Boolean(id) } })
  const claimsQuery = usePropertyTargetsClaims(targetType, id, { query: { enabled: Boolean(id) } })
  const uploadMutation = usePropertyDocumentsManagerCertificatesUpload()
  const detail = detailQuery.data?.data as CanonicalTargetResource | undefined
  const claimsBody = claimsQuery.data?.data as { claims?: SourceClaim[] } | undefined
  const claims = claimsBody?.claims ?? []
  const values = detail?.resolved_values ?? []
  const renovations = useMemo(() => detail?.renovation_events ?? [], [detail?.renovation_events])
  const documents = detail?.documents ?? []
  const sourceListings = detail?.source_listings ?? []
  const buildings = detail?.buildings ?? []
  const units = detail?.units ?? []
  const offerings = detail?.offerings ?? []
  const contextLookup = addressLookupInputFromParams(params)
  const contextLookupPath = buildAddressLookupPath(contextLookup)
  const [selectedRenovationID, setSelectedRenovationID] = useState<string>('')
  const selectedRenovation = useMemo(() => renovations.find(item => item.id === selectedRenovationID) ?? renovations[0], [renovations, selectedRenovationID])
  async function uploadCertificate(file: File | undefined) {
    if (!file) return
    await uploadMutation.mutateAsync({ data: { file }, params: { target_type: targetType, target_id: id } })
    await detailQuery.refetch()
  }
  return (
    <main className="model-page">
      <Nav />
      <div className="model-shell">
        <header className="model-header">
          <div>
            <Link className="model-back" to={contextLookupPath || '/search'}>{contextLookupPath ? 'Address lookup' : 'Targets'}</Link>
            <h1>{labelTargetType(targetType)}</h1>
            <p>{id}</p>
          </div>
          <label className="model-upload">
            Upload certificate
            <input type="file" accept="application/pdf" onChange={event => uploadCertificate(event.target.files?.[0])} />
          </label>
        </header>
        {detailQuery.isLoading && <div className="loading-state">Loading target</div>}
        {detailQuery.isError && <div className="error-state">Target could not be loaded.</div>}
        {detail && (
          <>
            <TargetSummary values={values} renovations={renovations} documents={documents} claims={claims} />
            <div className="model-grid">
              <section className="model-column model-column--main">
                <Overview overview={detail.overview} fallbackID={id} contextLookup={contextLookup} contextLookupPath={contextLookupPath} />
                <TargetChildren buildings={buildings} units={units} offerings={offerings} contextLookup={contextLookup} />
                <SourceListings listings={sourceListings} contextLookup={contextLookup} />
                <ResolvedValues values={values} />
                <Renovations events={renovations} selectedID={selectedRenovation?.id ?? ''} onSelect={setSelectedRenovationID} />
                <Documents documents={documents} />
              </section>
              <aside className="model-column">
                <RenovationDetail event={selectedRenovation} />
                <Claims claims={claims} />
              </aside>
            </div>
          </>
        )}
      </div>
    </main>
  )
}

function SourceListings({ listings, contextLookup }: { listings: TargetSourceListing[]; contextLookup: ReturnType<typeof addressLookupInputFromParams> }) {
  const [mode, setMode] = useState<'grouped' | 'raw'>('grouped')
  const groups = useMemo(() => {
    const map = new Map<string, TargetSourceListing[]>()
    for (const listing of listings) {
      const key = listing.offering_target.id
      map.set(key, [...(map.get(key) ?? []), listing])
    }
    return Array.from(map.entries()).map(([offeringID, items]) => ({ offeringID, items })).sort((a, b) => compareListingGroups(a.items, b.items))
  }, [listings])
  return (
    <Panel title="Source Listings" count={listings.length}>
      {groups.length === 0 ? <Empty text="No linked provider listings." /> : (
        <div className="source-listing-groups">
          {listings.length > 1 && (
            <div className="address-view-tabs source-listing-tabs">
              <button className={mode === 'grouped' ? 'is-active' : ''} type="button" onClick={() => setMode('grouped')}>Grouped offerings</button>
              <button className={mode === 'raw' ? 'is-active' : ''} type="button" onClick={() => setMode('raw')}>Raw source records</button>
            </div>
          )}
          {mode === 'grouped' ? (
            groups.map(group => <SourceListingGroup group={group} contextLookup={contextLookup} key={group.offeringID} />)
          ) : (
            <div className="source-listing-list">
              {listings.map(listing => <SourceListingRow listing={listing} contextLookup={contextLookup} key={listing.target.id} />)}
            </div>
          )}
        </div>
      )}
    </Panel>
  )
}

function SourceListingGroup({ group, contextLookup }: { group: { offeringID: string; items: TargetSourceListing[] }; contextLookup: ReturnType<typeof addressLookupInputFromParams> }) {
  const representative = group.items[0]
  const providers = Array.from(new Set(group.items.map(item => item.provider).filter(Boolean)))
  const priceMatches = group.items.filter(item => item.price_match).length
  const insightCount = group.items.reduce((count, item) => count + (item.insights?.length ?? 0), 0)
  if (!representative) return null
  return (
    <section className="source-listing-group">
      <div className="source-listing-group-head">
        <div>
          <span>Grouped offering</span>
          <h3>{representative.title || representative.street_address || shortID(group.offeringID)}</h3>
          <p>{[representative.room_layout, formatOptionalArea(representative.area_m2), formatOptionalCurrency(representative.asking_price_eur)].filter(Boolean).join(' / ')}</p>
        </div>
        <Link to={withAddressLookupContext(`/target/offering/${group.offeringID}`, contextLookup)}>Open grouped detail</Link>
      </div>
      <div className="source-listing-group-meta">
        <Fact label="Sources" value={providers.join(', ') || '-'} />
        <Fact label="Records" value={String(group.items.length)} />
        <Fact label="Price matches" value={priceMatches ? String(priceMatches) : '-'} />
        <Fact label="Insights" value={insightCount ? String(insightCount) : '-'} />
        <Fact label="Latest seen" value={formatDate(latestListingDate(group.items))} />
        <Fact label="Link score" value={bestLinkScore(group.items)} />
      </div>
      <div className="source-listing-list">
        {group.items.map(listing => <SourceListingRow listing={listing} contextLookup={contextLookup} key={listing.target.id} />)}
      </div>
    </section>
  )
}

function SourceListingRow({ listing, contextLookup }: { listing: TargetSourceListing; contextLookup: ReturnType<typeof addressLookupInputFromParams> }) {
  const detailPath = sourceEntityPath({ canonicalId: listing.canonical_id, kind: listing.kind })
  return (
    <article className="source-listing-row">
      <div className="source-listing-main">
        <span>{listing.provider} / {listing.kind}</span>
        <strong>{listing.title || listing.native_id || shortID(listing.target.id)}</strong>
        <small>{[listing.street_address, listing.postal, listing.city].filter(Boolean).join(' ')}</small>
      </div>
      <div className="source-listing-facts">
        <Fact label="Price" value={formatOptionalCurrency(listing.asking_price_eur)} />
        <Fact label="Debt-free" value={formatOptionalCurrency(listing.debt_free_price_eur)} />
        <Fact label="Area" value={formatOptionalArea(listing.area_m2)} />
        <Fact label="Layout" value={listing.room_layout} />
        <Fact label="Seen" value={formatDate(listing.last_seen_at)} />
        <Fact label="Match" value={formatPriceMatch(listing)} />
      </div>
      {(listing.insights?.length ?? 0) > 0 && (
        <div className="source-listing-insights">
          {listing.insights?.slice(0, 4).map(insight => (
            <span key={`${insight.source_field}:${insight.key}`}>
              <strong>{insight.key}</strong>
              {insight.value}
            </span>
          ))}
        </div>
      )}
      <div className="source-listing-actions">
        {detailPath && <Link to={withAddressLookupContext(detailPath, contextLookup)}>Source detail</Link>}
        <Link to={withAddressLookupContext(`/target/offering/${listing.offering_target.id}`, contextLookup)}>Grouped detail</Link>
        {listing.price_match && <Link to={withAddressLookupContext(`/matches?transaction=${listing.price_match.target.id}`, contextLookup)}>Price match</Link>}
        <LiveSourceLink available={listing.external_url_available} url={listing.url} />
      </div>
    </article>
  )
}

function TargetChildren({ buildings, units, offerings, contextLookup }: { buildings: TargetBuildingSummary[]; units: TargetUnitSummary[]; offerings: TargetOfferingSummary[]; contextLookup: ReturnType<typeof addressLookupInputFromParams> }) {
  const count = buildings.length + units.length + offerings.length
  if (count === 0) return null
  return (
    <Panel title="Canonical Children" count={count}>
      <div className="target-children">
        {offerings.length > 0 && <TargetOfferingList offerings={offerings} contextLookup={contextLookup} />}
        {units.length > 0 && <TargetUnitList units={units} contextLookup={contextLookup} />}
        {buildings.length > 0 && <TargetBuildingList buildings={buildings} contextLookup={contextLookup} />}
      </div>
    </Panel>
  )
}

function TargetOfferingList({ offerings, contextLookup }: { offerings: TargetOfferingSummary[]; contextLookup: ReturnType<typeof addressLookupInputFromParams> }) {
  return (
    <section className="target-child-group">
      <h3>Grouped offerings</h3>
      <div className="target-child-list">
        {offerings.map(offering => (
          <Link className="target-child-card" to={withAddressLookupContext(`/target/offering/${offering.target.id}`, contextLookup)} key={offering.target.id}>
            <span>{targetOfferingEyebrow(offering)}</span>
            <strong>{offering.title}</strong>
            <small>{[formatOptionalArea(offering.area_m2), formatOptionalCurrency(offering.asking_price_eur), formatDate(offering.last_seen_at)].filter(Boolean).join(' / ')}</small>
            <small>{targetOfferingMetadata(offering)}</small>
          </Link>
        ))}
      </div>
    </section>
  )
}

function TargetUnitList({ units, contextLookup }: { units: TargetUnitSummary[]; contextLookup: ReturnType<typeof addressLookupInputFromParams> }) {
  return (
    <section className="target-child-group">
      <h3>Units</h3>
      <div className="target-child-list">
        {units.map(unit => (
          <Link className="target-child-card" to={withAddressLookupContext(`/target/unit/${unit.target.id}`, contextLookup)} key={unit.target.id}>
            <span>{unit.layout || 'Unit'}</span>
            <strong>{unit.title}</strong>
            <small>{[unit.address, formatOptionalArea(unit.area_m2), countLabel(unit.offering_count, 'offering')].filter(Boolean).join(' / ')}</small>
          </Link>
        ))}
      </div>
    </section>
  )
}

function TargetBuildingList({ buildings, contextLookup }: { buildings: TargetBuildingSummary[]; contextLookup: ReturnType<typeof addressLookupInputFromParams> }) {
  return (
    <section className="target-child-group">
      <h3>Buildings</h3>
      <div className="target-child-list">
        {buildings.map(building => (
          <Link className="target-child-card" to={withAddressLookupContext(`/target/building/${building.target.id}`, contextLookup)} key={building.target.id}>
            <span>{building.postal || 'Building'}</span>
            <strong>{building.title}</strong>
            <small>{[building.address, countLabel(building.unit_count, 'unit'), countLabel(building.offering_count, 'offering')].filter(Boolean).join(' / ')}</small>
          </Link>
        ))}
      </div>
    </section>
  )
}

function Overview({ overview, fallbackID, contextLookup, contextLookupPath }: { overview?: TargetOverview; fallbackID: string; contextLookup: ReturnType<typeof addressLookupInputFromParams>; contextLookupPath: string }) {
  if (!overview) {
    return (
      <Panel title="Overview">
        <Empty text="No canonical overview for this target." />
      </Panel>
    )
  }
  const fields = (overview.fields ?? []).filter(field => field.value)
  const related = overview.related ?? []
  const lookupPath = contextLookupPath || addressLookupPathFromOverviewFields(fields)
  return (
    <Panel title="Overview">
      <div className="target-overview">
        <div className="target-overview-head">
          <div>
            <h2>{overview.title || fallbackID}</h2>
            {overview.subtitle && <p>{overview.subtitle}</p>}
          </div>
          {lookupPath && <Link to={lookupPath}>Address lookup</Link>}
        </div>
        {fields.length > 0 && (
          <div className="target-overview-fields">
            {fields.map(field => (
              <div key={field.label}>
                <span>{field.label}</span>
                <strong>{field.value}</strong>
              </div>
            ))}
          </div>
        )}
        <RelatedTargets items={related} contextLookup={contextLookup} />
        <SourceLinks overview={overview} contextLookup={contextLookup} />
      </div>
    </Panel>
  )
}

function RelatedTargets({ items, contextLookup }: { items: NonNullable<TargetOverview['related']>; contextLookup: ReturnType<typeof addressLookupInputFromParams> }) {
  const groups = useMemo(() => {
    const map = new Map<string, NonNullable<TargetOverview['related']>>()
    for (const item of items) {
      const key = item.label || item.target.type
      map.set(key, [...(map.get(key) ?? []), item])
    }
    return Array.from(map.entries()).map(([label, items]) => ({ label, items }))
  }, [items])
  if (groups.length === 0) return null
  return (
    <div className="target-related-groups">
      {groups.map(group => (
        <section className="target-related-group" key={group.label}>
          <h3>{group.label}</h3>
          <div className="target-overview-related">
            {group.items.map(item => (
              <Link key={`${item.target.type}:${item.target.id}`} to={withAddressLookupContext(`/target/${item.target.type}/${item.target.id}`, contextLookup)}>
                <span>{item.target.type.replaceAll('_', ' ')}</span>
                <strong>{item.title}</strong>
              </Link>
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}

function SourceLinks({ overview, contextLookup }: { overview: TargetOverview; contextLookup: ReturnType<typeof addressLookupInputFromParams> }) {
  const groups = useMemo(() => {
    const map = new Map<string, NonNullable<TargetOverview['sources']>>()
    for (const source of overview.sources ?? []) {
      const key = source.label || source.kind || 'Source'
      map.set(key, [...(map.get(key) ?? []), source])
    }
    return Array.from(map.entries()).map(([label, sources]) => ({ label, sources }))
  }, [overview.sources])
  if (groups.length === 0) return null
  return (
    <div className="target-source-groups">
      {groups.map(group => (
        <section className="target-source-group" key={group.label}>
          <h3>{group.label}</h3>
          <div className="target-source-list">
            {group.sources.map((source, index) => (
              <SourceLink source={source} index={index} contextLookup={contextLookup} key={`${source.provider}:${source.kind}:${source.source_id || index}`} />
            ))}
          </div>
        </section>
      ))}
    </div>
  )
}

function SourceLink({ source, index, contextLookup }: { source: NonNullable<TargetOverview['sources']>[number]; index: number; contextLookup: ReturnType<typeof addressLookupInputFromParams> }) {
  const detailPath = sourceEntityPath({ canonicalId: source.canonical_id, kind: source.kind })
  return (
    <div className="target-source-link" data-source-index={index}>
      <span>{source.provider} / {source.kind}</span>
      <strong>{source.title || source.source_id || source.external_id || 'Source'}</strong>
      <small>{[source.external_id || source.source_id, formatDate(source.last_seen_at)].filter(Boolean).join(' / ')}</small>
      <div className="target-source-actions">
        {detailPath && <Link to={withAddressLookupContext(detailPath, contextLookup)}>Source detail</Link>}
        <LiveSourceLink available={source.external_url_available} url={source.url} />
      </div>
    </div>
  )
}

function TargetSummary({ values, renovations, documents, claims }: { values: ResolvedValue[]; renovations: RenovationEvent[]; documents: PropertyDocumentSummary[]; claims: SourceClaim[] }) {
  const conflicting = values.filter(value => value.conflict_status && value.conflict_status !== 'none').length
  const certificateCount = documents.filter(document => document.type === 'manager_certificate').length
  return (
    <section className="model-summary">
      <Metric label="Resolved values" value={values.length} />
      <Metric label="Conflicts" value={conflicting} tone={conflicting > 0 ? 'warn' : 'muted'} />
      <Metric label="Renovations" value={renovations.length} />
      <Metric label="Certificates" value={certificateCount} />
      <Metric label="Claims" value={claims.length} />
    </section>
  )
}

function Metric({ label, value, tone }: { label: string; value: number | string; tone?: 'warn' | 'muted' }) {
  return (
    <div className={`model-metric${tone ? ` model-metric--${tone}` : ''}`}>
      <span>{label}</span>
      <strong>{value}</strong>
    </div>
  )
}

function ResolvedValues({ values }: { values: ResolvedValue[] }) {
  const groups = useMemo(() => groupByNamespace(values), [values])
  return (
    <Panel title="Resolved Values" count={values.length}>
      {groups.length === 0 ? <Empty text="No resolved values." /> : (
        <div className="value-groups">
          {groups.map(group => (
            <section className="value-group" key={group.name}>
              <h3>{group.name}</h3>
              <div className="value-table">
                {group.items.map(value => (
                  <div className="value-row" key={value.dimension_key}>
                    <div>
                      <span>{value.dimension_key}</span>
                      <small>{value.selected_reason}</small>
                    </div>
                    <strong>{formatValue(value.value, value.unit)}</strong>
                    <Badge>{value.conflict_status || 'none'}</Badge>
                  </div>
                ))}
              </div>
            </section>
          ))}
        </div>
      )}
    </Panel>
  )
}

function Renovations({ events, selectedID, onSelect }: { events: RenovationEvent[]; selectedID: string; onSelect: (id: string) => void }) {
  const groups = useMemo(() => groupRenovations(events), [events])
  return (
    <Panel title="Renovations" count={events.length}>
      {groups.length === 0 ? <Empty text="No renovation events." /> : (
        <div className="renovation-section-list">
          {groups.map(group => (
            <section className="renovation-section" key={group.status}>
              <h3>{labelStatus(group.status)}</h3>
              {group.items.map(event => (
                <button className={`renovation-event-row${event.id === selectedID ? ' is-selected' : ''}`} key={event.id} onClick={() => onSelect(event.id)} type="button">
                  <span className="renovation-event-year">{event.year ?? event.start_year ?? 'n/a'}</span>
                  <span className="renovation-event-main">
                    <strong>{labelCategory(event.category)}</strong>
                    <small>{[event.component, event.stage, event.responsibility].filter(Boolean).join(' / ') || event.source_field || event.source_table}</small>
                  </span>
                  <Badge>{Math.round(event.confidence * 100)}%</Badge>
                </button>
              ))}
            </section>
          ))}
        </div>
      )}
    </Panel>
  )
}

function RenovationDetail({ event }: { event?: RenovationEvent }) {
  if (!event) {
    return (
      <Panel title="Renovation Detail">
        <Empty text="Select a renovation." />
      </Panel>
    )
  }
  return (
    <Panel title="Renovation Detail">
      <div className="renovation-detail">
        <div className="renovation-detail-head">
          <strong>{labelCategory(event.category)}</strong>
          <Badge>{labelStatus(event.status)}</Badge>
        </div>
        {event.summary && <p>{event.summary}</p>}
        <DetailRows rows={[
          ['Year', yearRange(event)],
          ['Component', event.component],
          ['Stage', event.stage],
          ['Scope', event.scope],
          ['Responsibility', event.responsibility],
          ['Cost estimate', event.cost_estimate_eur ? formatCurrency(event.cost_estimate_eur) : ''],
          ['Observed', formatDate(event.observed_at)],
          ['Source', [event.source_table, event.source_field].filter(Boolean).join(' / ')],
          ['Source id', event.source_id],
          ['Projection', event.projection_version],
          ['Confidence', `${Math.round(event.confidence * 100)}%`],
          ['Reliability', `${Math.round(event.source_reliability * 100)}%`],
        ]} />
        {event.evidence != null && (
          <pre className="model-json">{JSON.stringify(event.evidence, null, 2)}</pre>
        )}
      </div>
    </Panel>
  )
}

function Documents({ documents }: { documents: PropertyDocumentSummary[] }) {
  return (
    <Panel title="Documents" count={documents.length}>
      {documents.length === 0 ? <Empty text="No documents attached." /> : (
        <div className="document-list model-document-list">
          {documents.map(document => (
            <a className="model-document-row" href={document.download_url} key={document.id}>
              <div>
                <strong>{document.filename}</strong>
                <small>{document.id}</small>
              </div>
              <span>{document.extraction_status}</span>
            </a>
          ))}
        </div>
      )}
    </Panel>
  )
}

function Claims({ claims }: { claims: SourceClaim[] }) {
  return (
    <Panel title="Claims" count={claims.length}>
      {claims.length === 0 ? <Empty text="No claims attached directly to this target." /> : (
        <div className="claim-list">
          {claims.slice(0, 80).map(claim => (
            <div className="claim-row" key={claim.id}>
              <div>
                <strong>{claim.dimension_key}</strong>
                <small>{claim.source_table} / {claim.source_field || 'source'}</small>
              </div>
              <span>{formatValue(claim.value, claim.unit)}</span>
            </div>
          ))}
        </div>
      )}
    </Panel>
  )
}

function Panel({ title, count, children }: { title: string; count?: number; children: ReactNode }) {
  return (
    <section className="model-panel">
      <header>
        <h2>{title}</h2>
        {typeof count === 'number' && <span>{count}</span>}
      </header>
      {children}
    </section>
  )
}

function DetailRows({ rows }: { rows: Array<[string, string | number | undefined]> }) {
  return (
    <div className="detail-rows">
      {rows.filter(([, value]) => value != null && value !== '').map(([label, value]) => (
        <div className="detail-row" key={label}>
          <span>{label}</span>
          <strong>{value}</strong>
        </div>
      ))}
    </div>
  )
}

function Empty({ text }: { text: string }) {
  return <p className="model-empty">{text}</p>
}

function Badge({ children }: { children: ReactNode }) {
  return <span className="model-badge">{children}</span>
}

function Fact({ label, value }: { label: string; value?: string }) {
  return (
    <div>
      <span>{label}</span>
      <strong>{value || '-'}</strong>
    </div>
  )
}

function groupByNamespace(values: ResolvedValue[]) {
  const map = new Map<string, ResolvedValue[]>()
  for (const value of values) {
    const namespace = value.dimension_key.split('.')[0] || 'other'
    map.set(namespace, [...(map.get(namespace) ?? []), value])
  }
  return Array.from(map.entries()).map(([name, items]) => ({ name, items })).sort((a, b) => a.name.localeCompare(b.name))
}

function groupRenovations(events: RenovationEvent[]) {
  const order = ['planned', 'suspected', 'forecast', 'done', 'unknown', 'cancelled']
  const map = new Map<string, RenovationEvent[]>()
  for (const event of events) {
    map.set(event.status, [...(map.get(event.status) ?? []), event])
  }
  return Array.from(map.entries()).map(([status, items]) => ({
    status,
    items: items.slice().sort((a, b) => (a.year ?? a.start_year ?? 9999) - (b.year ?? b.start_year ?? 9999)),
  })).sort((a, b) => order.indexOf(a.status) - order.indexOf(b.status))
}

function formatValue(value: unknown, unit?: string): string {
  if (value == null) return ''
  const rendered = typeof value === 'object' ? JSON.stringify(value) : String(value)
  return unit ? `${rendered} ${unit}` : rendered
}

function formatCurrency(value: number): string {
  return new Intl.NumberFormat('fi-FI', { style: 'currency', currency: 'EUR', maximumFractionDigits: 0 }).format(value)
}

function formatOptionalCurrency(value?: number): string {
  return value == null ? '' : formatCurrency(value)
}

function formatOptionalArea(value?: number): string {
  return value == null ? '' : `${value.toFixed(1)} m2`
}

function formatPriceMatch(listing: TargetSourceListing): string {
  if (listing.price_match) {
    return [listing.price_match.status, formatOptionalCurrency(listing.price_match.price_eur)].filter(Boolean).join(' / ')
  }
  return listing.link_status || ''
}

function targetOfferingEyebrow(offering: TargetOfferingSummary): string {
  const sources = offering.sources?.map(sourceLabel).join(' + ')
  return [offering.layout || 'Offering', sources].filter(Boolean).join(' / ')
}

function sourceLabel(source?: string): string {
  if (source === 'shortcut') return 'Shortcut'
  if (source === 'frontdoor') return 'Frontdoor'
  return source || ''
}

function targetOfferingMetadata(offering: TargetOfferingSummary): string {
  return [
    countLabel(offering.source_count ?? 0, 'source'),
    offering.price_match_status ? `Price ${[offering.price_match_status, formatOptionalCurrency(offering.price_match_price_eur)].filter(Boolean).join(' ')}` : '',
    offering.insight_count ? `${offering.insight_count} insights` : '',
  ].filter(Boolean).join(' / ')
}

function latestListingDate(listings: TargetSourceListing[]): string | undefined {
  return listings.reduce<string | undefined>((latest, listing) => {
    if (!listing.last_seen_at) return latest
    if (!latest) return listing.last_seen_at
    return new Date(listing.last_seen_at).getTime() > new Date(latest).getTime() ? listing.last_seen_at : latest
  }, undefined)
}

function bestLinkScore(listings: TargetSourceListing[]): string {
  const score = listings.reduce<number | undefined>((best, listing) => {
    if (listing.link_score == null) return best
    if (best == null) return listing.link_score
    return Math.max(best, listing.link_score)
  }, undefined)
  return score == null ? '' : String(score)
}

function compareListingGroups(a: TargetSourceListing[], b: TargetSourceListing[]): number {
  return listingGroupTime(b) - listingGroupTime(a)
}

function listingGroupTime(listings: TargetSourceListing[]): number {
  const date = latestListingDate(listings)
  return date ? new Date(date).getTime() : 0
}

function countLabel(value: number, noun: string): string {
  return `${value} ${noun}${value === 1 ? '' : 's'}`
}

function formatDate(value?: string): string {
  if (!value) return ''
  return new Intl.DateTimeFormat('fi-FI').format(new Date(value))
}

function yearRange(event: RenovationEvent): string {
  if (event.year) return String(event.year)
  if (event.start_year && event.end_year) return `${event.start_year}-${event.end_year}`
  if (event.start_year) return String(event.start_year)
  if (event.end_year) return String(event.end_year)
  return ''
}

function labelTargetType(value: string): string {
  return value.replaceAll('_', ' ')
}

function labelStatus(value: string): string {
  return value.replaceAll('_', ' ')
}

function labelCategory(value: string): string {
  return value.replaceAll('_', ' ')
}

function shortID(value: string): string {
  return value.slice(0, 8)
}
