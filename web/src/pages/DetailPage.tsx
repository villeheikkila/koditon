import { useMemo, useState, type ReactNode } from 'react'
import { Link, useParams } from 'react-router-dom'
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
  type TargetOverview,
} from '../api/koditon'

type DetailKind = 'listing' | 'rental' | 'housingCompany'

function targetTypeForKind(kind: DetailKind): string {
  if (kind === 'housingCompany') return 'housing_company'
  return 'offering'
}

export default function DetailPage({ kind }: { kind?: DetailKind }) {
  const { id = '', targetType: targetTypeParam = '' } = useParams()
  const targetType = targetTypeParam || targetTypeForKind(kind ?? 'listing')
  const detailQuery = usePropertyTargetsDetail(targetType, id, { query: { enabled: Boolean(id) } })
  const claimsQuery = usePropertyTargetsClaims(targetType, id, { query: { enabled: Boolean(id) } })
  const uploadMutation = usePropertyDocumentsManagerCertificatesUpload()
  const detail = detailQuery.data?.data as CanonicalTargetResource | undefined
  const claimsBody = claimsQuery.data?.data as { claims?: SourceClaim[] } | undefined
  const claims = claimsBody?.claims ?? []
  const values = detail?.resolved_values ?? []
  const renovations = detail?.renovation_events ?? []
  const documents = detail?.documents ?? []
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
            <Link className="model-back" to="/search">Targets</Link>
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
                <Overview overview={detail.overview} fallbackID={id} />
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

function Overview({ overview, fallbackID }: { overview?: TargetOverview; fallbackID: string }) {
  if (!overview) {
    return (
      <Panel title="Overview">
        <Empty text="No canonical overview for this target." />
      </Panel>
    )
  }
  const fields = (overview.fields ?? []).filter(field => field.value)
  const related = overview.related ?? []
  return (
    <Panel title="Overview">
      <div className="target-overview">
        <div className="target-overview-head">
          <div>
            <h2>{overview.title || fallbackID}</h2>
            {overview.subtitle && <p>{overview.subtitle}</p>}
          </div>
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
        <RelatedTargets items={related} />
        <SourceLinks overview={overview} />
      </div>
    </Panel>
  )
}

function RelatedTargets({ items }: { items: NonNullable<TargetOverview['related']> }) {
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
              <Link key={`${item.target.type}:${item.target.id}`} to={`/target/${item.target.type}/${item.target.id}`}>
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

function SourceLinks({ overview }: { overview: TargetOverview }) {
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
              <a key={`${source.provider}:${source.kind}:${source.source_id || index}`} href={source.url || '#'} target="_blank" rel="noreferrer" aria-disabled={!source.url}>
                <span>{source.provider} / {source.kind}</span>
                <strong>{source.title || source.source_id || 'Source page'}</strong>
                <small>{[source.source_id, formatDate(source.last_seen_at)].filter(Boolean).join(' / ')}</small>
              </a>
            ))}
          </div>
        </section>
      ))}
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
