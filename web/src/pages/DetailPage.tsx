import { useSearchParams, Link } from 'react-router-dom'
import { useEntityDetail, type EntityDetail } from '../api/entity'

export default function DetailPage() {
  const [searchParams] = useSearchParams()
  const id = searchParams.get('id') ?? ''
  const { data, isPending, isError, error } = useEntityDetail(id)

  if (!id) return <div className="error-screen">Missing entity ID</div>

  if (isPending) {
    return (
      <div className="loading-screen">
        <div className="spinner" />
        <span>Loading…</span>
      </div>
    )
  }

  if (isError) {
    return (
      <div className="error-screen">
        <span>{(error as Error)?.message ?? 'Failed to load entity'}</span>
        <Link to="/" style={{ marginTop: 12, fontSize: 13, color: 'var(--text-2)' }}>← Back</Link>
      </div>
    )
  }

  return <ListingView detail={data!} />
}

function ListingView({ detail: d }: { detail: EntityDetail }) {
  const title = [d.room_layout, d.area_m2 != null && `${d.area_m2.toFixed(1)} m²`]
    .filter(Boolean).join(', ')

  return (
    <div className="listing-layout">
      <div className="listing-container">

        {/* Top nav */}
        <div className="listing-nav">
          <Link to="/" className="listing-back">← Back</Link>
          {d.url && (
            <a href={d.url} target="_blank" rel="noopener noreferrer" className="listing-source-link">
              View on {d.source === 'shortcut' ? 'Shortcut' : 'Frontdoor'} ↗
            </a>
          )}
        </div>

        {/* Header */}
        <div className="listing-header">
          <div className="listing-header-main">
            <h1 className="listing-title">{d.street_address || d.headline}</h1>
            {title && <div className="listing-subtitle">{title}</div>}
            <div className="listing-location">
              {[d.postal, d.city].filter(Boolean).join(' ')}
            </div>
          </div>
          <div className="listing-header-price">
            {d.asking_price != null && (
              <div className="listing-price-main">{fmtPrice(d.asking_price)}</div>
            )}
            {d.debt_free_price != null && d.debt_free_price !== d.asking_price && (
              <div className="listing-price-sub">
                Debt-free {fmtPrice(d.debt_free_price)}
              </div>
            )}
            {d.price_per_m2 != null && (
              <div className="listing-price-sqm">{fmtEur(d.price_per_m2)} / m²</div>
            )}
          </div>
        </div>

        {/* Key facts strip */}
        <div className="listing-facts">
          {d.rooms_count != null && (
            <Fact label="Rooms" value={String(d.rooms_count)} />
          )}
          {d.area_m2 != null && (
            <Fact label="Area" value={`${d.area_m2.toFixed(1)} m²`} />
          )}
          {d.floor_level != null && (
            <Fact
              label="Floor"
              value={d.total_floors != null ? `${d.floor_level} / ${d.total_floors}` : String(d.floor_level)}
            />
          )}
          {d.build_year != null && (
            <Fact label="Built" value={String(d.build_year)} />
          )}
          {d.condition && <Fact label="Condition" value={d.condition} />}
          {d.energy_class && <Fact label="Energy" value={d.energy_class} />}
          {d.elevator != null && <Fact label="Elevator" value={d.elevator ? 'Yes' : 'No'} />}
          {d.sauna != null && <Fact label="Sauna" value={d.sauna ? 'Yes' : 'No'} />}
          {d.plot_type && <Fact label="Plot" value={d.plot_type} />}
        </div>

        <div className="listing-body">

          {/* Description */}
          {d.description_text && (
            <Section title="Description">
              <TextBlock text={d.description_text} />
            </Section>
          )}

          {/* Pricing breakdown */}
          {(d.debt_share_amount != null || d.maintenance_charge_monthly != null ||
            d.total_charge_monthly != null || d.water_charge != null || d.charges_text) && (
            <Section title="Pricing & Charges">
              <div className="listing-table">
                {d.asking_price != null && (
                  <Row label="Asking price" value={fmtPrice(d.asking_price)} highlight />
                )}
                {d.debt_free_price != null && (
                  <Row label="Debt-free price" value={fmtPrice(d.debt_free_price)} />
                )}
                {d.debt_share_amount != null && (
                  <Row label="Debt share" value={fmtPrice(d.debt_share_amount)} />
                )}
                {d.maintenance_charge_monthly != null && (
                  <Row label="Maintenance charge" value={`${fmtEur(d.maintenance_charge_monthly)} / mo`} />
                )}
                {d.total_charge_monthly != null &&
                  d.total_charge_monthly !== d.maintenance_charge_monthly && (
                    <Row label="Total monthly charge" value={`${fmtEur(d.total_charge_monthly)} / mo`} />
                )}
                {d.water_charge != null && (
                  <Row label="Water charge" value={`${fmtEur(d.water_charge)} / mo`} />
                )}
              </div>
              {d.charges_text && <TextBlock text={d.charges_text} muted />}
            </Section>
          )}

          {/* Property details */}
          <Section title="Property Details">
            <div className="listing-table">
              {d.room_layout && <Row label="Room layout" value={d.room_layout} />}
              {d.rooms_count != null && <Row label="Rooms" value={String(d.rooms_count)} />}
              {d.area_m2 != null && <Row label="Area" value={`${d.area_m2.toFixed(1)} m²`} />}
              {d.price_per_m2 != null && <Row label="Price / m²" value={fmtEur(d.price_per_m2)} />}
              {d.floor_level != null && (
                <Row
                  label="Floor"
                  value={d.total_floors != null ? `${d.floor_level} / ${d.total_floors}` : String(d.floor_level)}
                />
              )}
              {d.build_year != null && <Row label="Year built" value={String(d.build_year)} />}
              {d.condition && <Row label="Condition" value={d.condition} />}
              {d.energy_class && <Row label="Energy class" value={d.energy_class} />}
              {d.plot_type && <Row label="Plot type" value={d.plot_type} />}
              {d.elevator != null && <Row label="Elevator" value={d.elevator ? 'Yes' : 'No'} />}
              {d.sauna != null && <Row label="Sauna" value={d.sauna ? 'Yes' : 'No'} />}
            </div>
          </Section>

          {/* Availability */}
          {d.availability_text && (
            <Section title="Availability">
              <TextBlock text={d.availability_text} />
            </Section>
          )}

          {/* Renovations */}
          {(d.renovations_done_text || d.renovations_planned_text) && (
            <Section title="Renovations">
              {d.renovations_done_text && (
                <>
                  <div className="listing-subsection-label">Completed</div>
                  <TextBlock text={d.renovations_done_text} />
                </>
              )}
              {d.renovations_planned_text && (
                <>
                  <div className="listing-subsection-label" style={{ marginTop: 12 }}>Planned</div>
                  <TextBlock text={d.renovations_planned_text} />
                </>
              )}
            </Section>
          )}

          {/* Additional info */}
          {d.additional_info_text && (
            <Section title="Additional Information">
              <TextBlock text={d.additional_info_text} />
            </Section>
          )}

          {/* Source-specific fields */}
          {d.source_specific && d.source_specific.length > 0 && (
            <Section title="Building">
              <div className="listing-table">
                {d.source_specific.map((f, i) => (
                  f.value && f.value !== '00000000-0000-0000-0000-000000000000' && (
                    <Row key={i} label={f.label} value={f.value} />
                  )
                ))}
              </div>
            </Section>
          )}

          {/* Related */}
          {d.related && d.related.length > 0 && (
            <Section title="Building Listings">
              <div className="listing-table">
                {d.related.map((f, i) => (
                  <Row key={i} label={f.label} value={f.value} />
                ))}
              </div>
            </Section>
          )}

          {/* Footer meta */}
          <div className="listing-meta-footer">
            <span className="badge badge-default">{d.canonical_id}</span>
            {d.last_seen_at && (
              <span className="listing-meta-date">
                Last seen {new Date(d.last_seen_at).toLocaleDateString('fi-FI')}
              </span>
            )}
          </div>

        </div>
      </div>
    </div>
  )
}

function Section({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <section className="listing-section">
      <h2 className="listing-section-title">{title}</h2>
      {children}
    </section>
  )
}

function Fact({ label, value }: { label: string; value: string }) {
  return (
    <div className="listing-fact">
      <span className="listing-fact-value">{value}</span>
      <span className="listing-fact-label">{label}</span>
    </div>
  )
}

function Row({ label, value, highlight }: { label: string; value: string; highlight?: boolean }) {
  return (
    <div className="listing-row">
      <span className="listing-row-label">{label}</span>
      <span className={`listing-row-value${highlight ? ' listing-row-value--highlight' : ''}`}>
        {value}
      </span>
    </div>
  )
}

function TextBlock({ text, muted }: { text: string; muted?: boolean }) {
  return (
    <p className={`listing-text${muted ? ' listing-text--muted' : ''}`}>
      {text}
    </p>
  )
}

function fmtPrice(n: number): string {
  return new Intl.NumberFormat('fi-FI').format(n) + ' €'
}

function fmtEur(n: number): string {
  return new Intl.NumberFormat('fi-FI', { maximumFractionDigits: 0 }).format(n) + ' €'
}
