import { useSearchParams, Link } from 'react-router-dom'
import { useEntityDetail } from '../api/entity'

export default function DetailPage() {
  const [searchParams] = useSearchParams()
  const id = searchParams.get('id') ?? ''
  const { data, isPending, isError, error } = useEntityDetail(id ?? '')

  if (!id) return <div className="error-screen">Missing entity ID</div>

  if (isPending) {
    return (
      <div className="loading-screen">
        <div className="spinner" />
        <span>Loading&hellip;</span>
      </div>
    )
  }

  if (isError) {
    const msg = (error as Error)?.message ?? 'Failed to load entity'
    return (
      <div className="error-screen">
        <span>{msg}</span>
        <Link to="/" style={{ marginTop: 12, fontSize: 13, color: 'var(--text-2)' }}>← Back</Link>
      </div>
    )
  }

  const d = data!

  return (
    <div className="detail-layout">
      <div className="detail-container">
        <Link to="/" className="detail-back">← Back</Link>

        <h1 className="detail-headline">{d.headline}</h1>
        <div className="detail-meta">
          <span className="badge badge-default">{d.source}</span>
          <span className="badge badge-default">{d.kind}</span>
          <span className="detail-meta-id">{d.canonical_id}</span>
          {d.url && (
            <a href={d.url} target="_blank" rel="noopener noreferrer" className="detail-source-link">
              Source ↗
            </a>
          )}
        </div>

        <div className="detail-cards">
          <DetailCard title="Overview">
            {d.address && <DetailRow label="Address" value={d.address} />}
            {d.city && <DetailRow label="City" value={d.city} />}
            {d.postal && <DetailRow label="Postal" value={d.postal} />}
            {d.price != null && <DetailRow label="Price" value={formatPrice(d.price)} accent />}
            {d.area != null && <DetailRow label="Area" value={formatArea(d.area)} />}
            {d.room_layout && <DetailRow label="Room Layout" value={d.room_layout} />}
            {d.last_seen_at && <DetailRow label="Last Seen" value={formatDate(d.last_seen_at)} />}
          </DetailCard>

          {d.canonical_extra && d.canonical_extra.length > 0 && (
            <DetailCard title="Details">
              {d.canonical_extra.map((f, i) => (
                <DetailRow key={i} label={f.label} value={f.value} />
              ))}
            </DetailCard>
          )}

          {d.source_specific && d.source_specific.length > 0 && (
            <DetailCard title="Source Specific">
              {d.source_specific.map((f, i) => (
                <DetailRow key={i} label={f.label} value={f.value} />
              ))}
            </DetailCard>
          )}

          {d.related && d.related.length > 0 && (
            <DetailCard title="Related">
              {d.related.map((f, i) => (
                <DetailRow key={i} label={f.label} value={f.value} />
              ))}
            </DetailCard>
          )}

          {d.raw_json && (
            <details className="detail-raw">
              <summary className="detail-raw-summary">Raw JSON</summary>
              <pre className="detail-raw-pre">{d.raw_json}</pre>
            </details>
          )}
        </div>
      </div>
    </div>
  )
}

function DetailCard({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="detail-card">
      <div className="detail-card-title">{title}</div>
      {children}
    </div>
  )
}

function DetailRow({ label, value, accent }: { label: string; value: string; accent?: boolean }) {
  return (
    <div className="detail-row">
      <span className="detail-row-label">{label}</span>
      <span className={`detail-row-value${accent ? ' accent' : ''}`}>{value}</span>
    </div>
  )
}

function formatPrice(p: number): string {
  return new Intl.NumberFormat('fi-FI').format(p) + ' €'
}

function formatArea(a: number): string {
  return a.toFixed(1) + ' m²'
}

function formatDate(s: string): string {
  return new Date(s).toLocaleDateString('fi-FI', { year: 'numeric', month: '2-digit', day: '2-digit' })
}
