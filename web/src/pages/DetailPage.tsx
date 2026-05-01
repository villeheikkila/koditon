import { Link, useParams } from 'react-router-dom'
import {
  useBuildingsDetail,
  useRentalsDetail,
  useSaleListingsDetail,
  type Building,
  type Rental,
  type SaleListing,
} from '../api/koditon'

type ListingDetail = SaleListing | Rental

interface DetailPageProps {
  kind: 'listing' | 'rental' | 'building'
}

export default function DetailPage({ kind }: DetailPageProps) {
  const params = useParams()
  const id = params.id ? decodeURIComponent(params.id) : ''
  const sale = useSaleListingsDetail(id, { query: { enabled: !!id && kind === 'listing', retry: false } })
  const rental = useRentalsDetail(id, { query: { enabled: !!id && kind === 'rental', retry: false } })
  const building = useBuildingsDetail(id, { query: { enabled: !!id && kind === 'building', retry: false } })

  if (!id) return <div className="error-screen">Missing entity ID</div>
  if (kind === 'building') {
    if (building.isPending) return <Loading />
    if (building.isError || !building.data?.data) return <ErrorMessage error={building.error} />
    return <BuildingView building={building.data.data as Building} />
  }
  if (kind === 'rental') {
    if (rental.isPending) return <Loading />
    if (rental.isError || !rental.data?.data) return <ErrorMessage error={rental.error} />
    return <ListingView detail={rental.data.data as Rental} kind="rental" />
  }
  if (sale.isPending) return <Loading />
  if (sale.isError || !sale.data?.data) return <ErrorMessage error={sale.error} />
  return <ListingView detail={sale.data.data as SaleListing} kind="listing" />
}

function Loading() {
  return (
    <div className="loading-screen">
      <div className="spinner" />
      <span>Loading…</span>
    </div>
  )
}

function ErrorMessage({ error }: { error: unknown }) {
  return (
    <div className="error-screen">
      <span>{(error as Error)?.message ?? 'Failed to load entity'}</span>
      <Link to="/" style={{ marginTop: 12, fontSize: 13, color: 'var(--text-2)' }}>← Back</Link>
    </div>
  )
}

function ListingView({ detail: d, kind }: { detail: ListingDetail; kind: 'listing' | 'rental' }) {
  const unit = d.unit
  const building = d.building
  const commercial = d.commercial
  const matchedTransaction = kind === 'listing' ? commercial.matched_transaction : undefined
  const texts = d.texts
  const location = unit.location
  const charges = commercial.charges
  const price = commercial.asking_price ?? commercial.rent
  const isRental = kind === 'rental'
  const title = [unit.room_layout, unit.area_m2 != null && `${unit.area_m2.toFixed(1)} m²`]
    .filter(Boolean).join(', ')
  const mainImage = d.media?.main_image?.variants?.gallery || d.media?.main_image?.variants?.large || d.media?.main_image?.url

  return (
    <div className="listing-layout">
      <div className="listing-container">
        <div className="listing-nav">
          <Link to="/" className="listing-back">← Back</Link>
          {d.source.url && (
            <a href={d.source.url} target="_blank" rel="noopener noreferrer" className="listing-source-link">
              View on {providerLabel(d.source.provider)} ↗
            </a>
          )}
        </div>
        {mainImage && <img className="listing-main-image" src={mainImage} alt="" />}
        <div className="listing-header">
          <div className="listing-header-main">
            <div className={`listing-type-pill listing-type-pill--${kind}`}>{isRental ? 'Rental' : 'Sale listing'}</div>
            <h1 className="listing-title">{location.street_address || d.headline}</h1>
            {title && <div className="listing-subtitle">{title}</div>}
            <div className="listing-location">
              {[location.postal, location.city].filter(Boolean).join(' ')}
            </div>
          </div>
          <div className="listing-header-price">
            {price != null && (
              <div className="listing-price-main">{fmtPrice(price)}</div>
            )}
            {isRental && commercial.rent_period && (
              <div className="listing-price-sub">
                Rent period {commercial.rent_period}
              </div>
            )}
            {commercial.debt_free_price != null && commercial.debt_free_price !== commercial.asking_price && (
              <div className="listing-price-sub">
                Debt-free {fmtPrice(commercial.debt_free_price)}
              </div>
            )}
            {commercial.price_per_m2 != null && (
              <div className="listing-price-sqm">{fmtEur(commercial.price_per_m2)} / m²</div>
            )}
            {matchedTransaction?.price != null && (
              <div className="listing-price-transaction">
                Sold {fmtPrice(matchedTransaction.price)}
                {matchedTransaction.period_identifier ? ` · ${matchedTransaction.period_identifier}` : ''}
              </div>
            )}
          </div>
        </div>
        <div className="listing-facts">
          {unit.rooms_count != null && <Fact label="Rooms" value={String(unit.rooms_count)} />}
          {unit.area_m2 != null && <Fact label="Area" value={`${unit.area_m2.toFixed(1)} m²`} />}
          {unit.floor_level != null && <Fact label="Floor" value={building.floor_count != null ? `${unit.floor_level} / ${building.floor_count}` : String(unit.floor_level)} />}
          {building.build_year != null && <Fact label="Built" value={String(building.build_year)} />}
          {unit.condition && <Fact label="Condition" value={unit.condition} />}
          {building.energy_class && <Fact label="Energy" value={building.energy_class} />}
          {building.elevator != null && <Fact label="Elevator" value={building.elevator ? 'Yes' : 'No'} />}
          {unit.sauna != null && <Fact label="Sauna" value={unit.sauna ? 'Yes' : 'No'} />}
          {d.site?.plot_type && <Fact label="Plot" value={d.site.plot_type} />}
        </div>
        <div className="listing-body">
          {texts?.description && (
            <Section title="Description">
              <TextBlock text={texts.description} />
            </Section>
          )}
          {(commercial.debt_share_amount != null || charges?.maintenance_monthly != null ||
            charges?.total_monthly != null || charges?.water != null || commercial.security_deposit ||
            commercial.minimum_term_months != null || commercial.pets_allowed != null || texts?.charges) && (
            <Section title="Pricing & Charges">
              <div className="listing-table">
                {commercial.rent != null && <Row label="Rent" value={`${fmtPrice(commercial.rent)}${commercial.rent_period ? ` / ${commercial.rent_period}` : ''}`} highlight />}
                {commercial.asking_price != null && <Row label="Asking price" value={fmtPrice(commercial.asking_price)} highlight={!isRental} />}
                {commercial.debt_free_price != null && <Row label="Debt-free price" value={fmtPrice(commercial.debt_free_price)} />}
                {matchedTransaction?.price != null && <Row label="Matched sale price" value={fmtPrice(matchedTransaction.price)} highlight />}
                {matchedTransaction?.price_per_m2 != null && <Row label="Matched sale / m²" value={fmtEur(matchedTransaction.price_per_m2)} />}
                {matchedTransaction?.period_identifier && <Row label="Transaction period" value={matchedTransaction.period_identifier} />}
                {matchedTransaction?.description && <Row label="Transaction layout" value={matchedTransaction.description} />}
                {matchedTransaction?.match_score != null && <Row label="Match confidence" value={[matchedTransaction.match_confidence, String(matchedTransaction.match_score)].filter(Boolean).join(' · ')} />}
                {commercial.debt_share_amount != null && <Row label="Debt share" value={fmtPrice(commercial.debt_share_amount)} />}
                {charges?.maintenance_monthly != null && <Row label="Maintenance charge" value={`${fmtEur(charges.maintenance_monthly)} / mo`} />}
                {charges?.total_monthly != null && charges.total_monthly !== charges.maintenance_monthly && <Row label="Total monthly charge" value={`${fmtEur(charges.total_monthly)} / mo`} />}
                {charges?.water != null && <Row label="Water charge" value={`${fmtEur(charges.water)} / mo`} />}
                {commercial.security_deposit && <Row label="Security deposit" value={commercial.security_deposit} />}
                {commercial.minimum_term_months != null && <Row label="Minimum term" value={`${commercial.minimum_term_months} months`} />}
                {commercial.pets_allowed != null && <Row label="Pets allowed" value={commercial.pets_allowed ? 'Yes' : 'No'} />}
              </div>
              {texts?.charges && <TextBlock text={texts.charges} muted />}
            </Section>
          )}
          <Section title="Unit">
            <div className="listing-table">
              {unit.room_layout && <Row label="Room layout" value={unit.room_layout} />}
              {unit.rooms_count != null && <Row label="Rooms" value={String(unit.rooms_count)} />}
              {unit.area_m2 != null && <Row label="Area" value={`${unit.area_m2.toFixed(1)} m²`} />}
              {commercial.price_per_m2 != null && <Row label="Price / m²" value={fmtEur(commercial.price_per_m2)} />}
              {unit.floor_level != null && <Row label="Floor" value={building.floor_count != null ? `${unit.floor_level} / ${building.floor_count}` : String(unit.floor_level)} />}
              {unit.condition && <Row label="Condition" value={unit.condition} />}
              {unit.parking && <Row label="Parking" value={unit.parking} />}
              {unit.kitchen_description && <Row label="Kitchen" value={unit.kitchen_description} />}
              {unit.bathroom_description && <Row label="Bathroom" value={unit.bathroom_description} />}
              {unit.storage_description && <Row label="Storage" value={unit.storage_description} />}
            </div>
          </Section>
          <Section title="Building">
            <div className="listing-table">
              {building.housing_company && <Row label="Housing company" value={building.housing_company} />}
              {building.build_year != null && <Row label="Year built" value={String(building.build_year)} />}
              {building.energy_class && <Row label="Energy class" value={building.energy_class} />}
              {building.heating && <Row label="Heating" value={building.heating} />}
              {building.apartment_count != null && <Row label="Apartments" value={String(building.apartment_count)} />}
              {building.elevator != null && <Row label="Elevator" value={building.elevator ? 'Yes' : 'No'} />}
              {building.common_areas && <Row label="Common areas" value={building.common_areas} />}
            </div>
          </Section>
          {texts?.availability && (
            <Section title="Availability">
              <TextBlock text={texts.availability} />
            </Section>
          )}
          {(texts?.renovations_done || texts?.renovations_planned) && (
            <Section title="Renovations">
              {texts.renovations_done && <TextBlock text={texts.renovations_done} />}
              {texts.renovations_planned && <TextBlock text={texts.renovations_planned} />}
            </Section>
          )}
          {texts?.additional_info && (
            <Section title="Additional Information">
              <TextBlock text={texts.additional_info} />
            </Section>
          )}
          <div className="listing-meta-footer">
            <span className="badge badge-default">{d.id}</span>
            {commercial.last_seen_at && (
              <span className="listing-meta-date">
                Last seen {new Date(commercial.last_seen_at).toLocaleDateString('fi-FI')}
              </span>
            )}
          </div>
        </div>
      </div>
    </div>
  )
}

function BuildingView({ building }: { building: Building }) {
  const details = building.details
  return (
    <div className="listing-layout">
      <div className="listing-container">
        <div className="listing-nav">
          <Link to="/" className="listing-back">← Back</Link>
          {building.source_records?.[0]?.url && (
            <a href={building.source_records[0].url} target="_blank" rel="noopener noreferrer" className="listing-source-link">
              View on {providerLabel(building.source_records[0].provider)} ↗
            </a>
          )}
        </div>
        <div className="listing-header">
          <div className="listing-header-main">
            <h1 className="listing-title">{details.housing_company || details.location.street_address || building.id}</h1>
            <div className="listing-location">
              {[details.location.postal, details.location.city].filter(Boolean).join(' ')}
            </div>
          </div>
        </div>
        <Section title="Building">
          <div className="listing-table">
            {details.location.street_address && <Row label="Address" value={details.location.street_address} />}
            {details.business_id && <Row label="Business ID" value={details.business_id} />}
            {details.build_year != null && <Row label="Year built" value={String(details.build_year)} />}
            {details.apartment_count != null && <Row label="Apartments" value={String(details.apartment_count)} />}
            {details.floor_count != null && <Row label="Floors" value={String(details.floor_count)} />}
            {details.heating && <Row label="Heating" value={details.heating} />}
            {details.energy_class && <Row label="Energy class" value={details.energy_class} />}
          </div>
        </Section>
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

function providerLabel(provider: string): string {
  return provider === 'shortcut' ? 'Shortcut' : 'Frontdoor'
}

function fmtPrice(n: number): string {
  return new Intl.NumberFormat('fi-FI').format(n) + ' €'
}

function fmtEur(n: number): string {
  return new Intl.NumberFormat('fi-FI', { maximumFractionDigits: 0 }).format(n) + ' €'
}
