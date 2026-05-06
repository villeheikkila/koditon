import { useState } from 'react'
import { useMutation } from '@tanstack/react-query'
import { Link, useParams } from 'react-router-dom'
import {
  useHousingCompaniesDetail,
  useRentalsDetail,
  useSaleListingsDetail,
  type Building,
  type Rental,
  type SaleListing,
} from '../api/koditon'
import ListingLocationMap from '../components/ListingLocationMap'
import { customInstance } from '../lib/axios-instance'

type ListingDetail = SaleListing | Rental
type RenovationStatus = 'done' | 'planned'
const RENOVATION_EXTRACTION_MODEL = '~google/gemini-flash-latest'
type RenovationItem = {
  kind: string
  status: RenovationStatus
  year?: number
}
type RenovationExtractionResult = {
  sale_listing_id: string
  model: string
  items?: RenovationExtractionResultItem[] | null
}
type RenovationExtractionResultItem = {
  category: string
  status: string
  year?: number
  text?: string
  confidence: number
  source_field: string
}
type RenovationExtractionResponse = {
  data: RenovationExtractionResult
  status: number
  headers: Headers
}
type DescriptionExtractionResult = {
  sale_listing_id: string
  model: string
  items?: Array<{ key: string; value: string; direction: string; severity: string; confidence: number; text?: string; source_field: string }> | null
}
type DescriptionExtractionResponse = {
  data: DescriptionExtractionResult
  status: number
  headers: Headers
}
type ValuationFact = {
  section: string
  key: string
  value_kind: string
  value_text?: string
  value_number?: number
  value_bool?: boolean
  confidence?: number
  source?: string
  evidence?: string
  model?: string
  prompt?: string
}
type ValuationInputExtractionResult = {
  sale_listing_id: string
  model: string
  facts?: ValuationFact[] | null
}
type ValuationInputExtractionResponse = {
  data: ValuationInputExtractionResult
  status: number
  headers: Headers
}
type ApartmentProfile = {
  housing_company_id?: string
  property_unit_id?: string
  area_m2?: number
  living_area_m2?: number
  room_layout?: string
  room_count?: number
  bedroom_count?: number
  floor_level?: number
  total_floors?: number
  kitchen_type?: string
  layout_quality?: string
  awkward_layout?: boolean
  condition?: string
  kitchen_condition?: string
  bathroom_condition?: string
  surface_renovation_need?: boolean
  modernization_need?: boolean
  sauna?: boolean
  balcony?: boolean
  balcony_glazing?: boolean
  parking_type?: string
  storage_quality?: string
  view_quality?: string
  noise_risk?: boolean
  accessibility?: string
  confidence?: string
  updated_at?: string
}
type ApartmentProfileProjectionResult = {
  sale_listing_id: string
  apartment_profile?: ApartmentProfile
}
type ApartmentProfileProjectionResponse = {
  data: ApartmentProfileProjectionResult
  status: number
  headers: Headers
}
type HouseOverview = {
  headline?: string
  summary?: string
  renovation_readiness?: string
  expensive_window?: string
  key_strengths?: string[] | null
  key_risks?: string[] | null
  evidence_gaps?: string[] | null
  confidence?: string
  generated_at?: string
  model?: string
}
type HouseOverviewGenerationResult = {
  sale_listing_id: string
  model: string
  overview: HouseOverview
}
type HouseOverviewGenerationResponse = {
  data: HouseOverviewGenerationResult
  status: number
  headers: Headers
}
type ApartmentProfileFact = {
  key: string
  label: string
  value: string
  tone?: 'positive' | 'negative'
}
type ApartmentProfileGroup = {
  key: string
  title: string
  facts: ApartmentProfileFact[]
}
type ValuationInput = {
  unit?: Record<string, unknown>
  layout?: Record<string, unknown>
  floor?: Record<string, unknown>
  building?: Record<string, unknown>
  site?: Record<string, unknown>
  charges?: Record<string, unknown>
  market?: Record<string, unknown>
  renovations?: Record<string, unknown>
  documents?: Record<string, unknown>
  facts?: ValuationFact[] | null
  extra_facts?: ValuationFact[] | null
  conflicts?: Array<{ path: string; reason: string }> | null
  missing?: string[] | null
}
type RenovationForecastItem = {
  category: string
  component?: string
  status: string
  scope?: string
  stage?: string
  responsibility?: string
  year?: number
  year_range?: string
  window_start_year?: number
  window_end_year?: number
  basis_year?: number
  cycle_years?: number
  severity: string
  confidence?: string
  cost_estimate_eur?: number
  price_effect: string
  source: string
  depends_on?: string[] | null
  price_mechanisms?: string[] | null
  explanation: string
}
type KeyRenovationGridItem = {
  category: string
  done?: RenovationItem
  planned?: RenovationItem
  forecast?: RenovationForecastItem
}
type ValueRange = {
  low?: number
  high?: number
}
type OfferAssessment = {
  verdict: string
  asking_price?: number
  debt_free_price?: number
  market_value_range?: ValueRange
  risk_adjusted_value_range?: ValueRange
  recommended_offer_range?: ValueRange
  renovation_risk_reserve?: ValueRange
  renovation_risk_reserve_per_m2?: ValueRange
  confidence: string
  main_reasons?: Array<{ key: string; direction: string; severity: string; explanation: string }> | null
  missing?: string[] | null
  explanation: string
}
type OwnershipCostWindow = {
  start_year?: number
  end_year?: number
  severity: string
  label: string
  reasons?: string[] | null
}
type KeyRenovationStatus = {
  category: string
  status: string
  year?: number
  window_start_year?: number
  window_end_year?: number
  severity?: string
  confidence?: string
  explanation?: string
}
type BriefSignal = {
  key: string
  label: string
  severity: string
  direction: string
  explanation?: string
}
type ValuationBrief = {
  verdict: string
  label?: string
  building_risk?: string
  expensive_windows?: OwnershipCostWindow[] | null
  key_renovations?: KeyRenovationStatus[] | null
  top_risks?: BriefSignal[] | null
  top_positives?: BriefSignal[] | null
  missing_evidence?: string[] | null
  confidence: string
  explanation?: string
}
type SaleListingWithValuation = SaleListing & {
  apartment_profile?: ApartmentProfile
  house_overview?: HouseOverview
  valuation_inputs?: {
    facts?: ValuationFact[] | null
  }
  valuation?: {
    input?: {
      facts?: ValuationFact[] | null
    } & ValuationInput
    brief?: ValuationBrief
    renovations?: {
      next_40_years?: RenovationForecastItem[] | null
      forecast_start_year?: number
      forecast_horizon_years?: number
    }
    offer_assessment?: OfferAssessment
  }
}

interface DetailPageProps {
  kind: 'listing' | 'rental' | 'housingCompany'
}

export default function DetailPage({ kind }: DetailPageProps) {
  const params = useParams()
  const id = params.id ? decodeURIComponent(params.id) : ''
  const sale = useSaleListingsDetail(id, { query: { enabled: !!id && kind === 'listing', retry: false } })
  const rental = useRentalsDetail(id, { query: { enabled: !!id && kind === 'rental', retry: false } })
  const housingCompany = useHousingCompaniesDetail(id, { query: { enabled: !!id && kind === 'housingCompany', retry: false } })

  if (!id) return <div className="error-screen">Missing entity ID</div>
  if (kind === 'housingCompany') {
    if (housingCompany.isPending) return <Loading />
    if (housingCompany.isError || !housingCompany.data?.data) return <ErrorMessage error={housingCompany.error} />
    return <BuildingView building={housingCompany.data.data as Building} />
  }
  if (kind === 'rental') {
    if (rental.isPending) return <Loading />
    if (rental.isError || !rental.data?.data) return <ErrorMessage error={rental.error} />
    return <ListingView detail={rental.data.data as Rental} kind="rental" />
  }
  if (sale.isPending) return <Loading />
  if (sale.isError || !sale.data?.data) return <ErrorMessage error={sale.error} />
  return <ListingView detail={sale.data.data as SaleListing} kind="listing" onRefresh={() => sale.refetch().then(() => undefined)} />
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

function ListingView({ detail: d, kind, onRefresh }: { detail: ListingDetail; kind: 'listing' | 'rental'; onRefresh?: () => Promise<void> }) {
  const [transactionOpen, setTransactionOpen] = useState(false)
  const [renovationExtractionResult, setRenovationExtractionResult] = useState<RenovationExtractionResult | null>(null)
  const [descriptionExtractionResult, setDescriptionExtractionResult] = useState<DescriptionExtractionResult | null>(null)
  const [valuationInputExtractionResult, setValuationInputExtractionResult] = useState<ValuationInputExtractionResult | null>(null)
  const [apartmentProfileProjectionResult, setApartmentProfileProjectionResult] = useState<ApartmentProfileProjectionResult | null>(null)
  const [houseOverviewResult, setHouseOverviewResult] = useState<HouseOverviewGenerationResult | null>(null)
  const unit = d.unit
  const building = d.building
  const commercial = d.commercial
  const matchedTransaction = kind === 'listing' ? commercial.matched_transaction : undefined
  const texts = d.texts
  const location = unit.location
  const charges = commercial.charges
  const price = commercial.asking_price ?? commercial.rent
  const isRental = kind === 'rental'
  const saleDetail = kind === 'listing' ? d as SaleListing : undefined
  const site = d.site
  const siteRows = site ? [
    site.plot_type,
    site.plot_ownership_type,
    site.plot_area_m2,
    site.lot_redemption_info,
    site.lot_rental_agreement,
    site.yard,
    site.shore,
    site.zoning,
    site.road_access,
    site.water_supply,
    site.water_supply_types?.length,
    site.sewer,
    site.services,
    site.transport,
    site.driving_directions,
  ] : []
  const hasSiteDetails = siteRows.some(value => value != null && value !== '')
  const title = [unit.room_layout, unit.area_m2 != null && `${unit.area_m2.toFixed(1)} m²`]
    .filter(Boolean).join(', ')
  const mainImage = d.media?.main_image?.variants?.large || d.media?.main_image?.variants?.gallery || d.media?.main_image?.url
  const mainImageURLs = new Set([d.media?.main_image?.url, d.media?.main_image?.variants?.large, d.media?.main_image?.variants?.gallery].filter(Boolean))
  const images = d.media?.images?.filter(image => !mainImageURLs.has(image.url)) ?? []
  const renovationRows = renovationItems(building.renovations, texts?.renovations_done, texts?.renovations_planned)
  const valuationRenovations = (saleDetail as SaleListingWithValuation | undefined)?.valuation?.renovations
  const valuationBrief = (saleDetail as SaleListingWithValuation | undefined)?.valuation?.brief
  const offerAssessment = (saleDetail as SaleListingWithValuation | undefined)?.valuation?.offer_assessment
  const apartmentProfile = apartmentProfileProjectionResult?.apartment_profile ?? (saleDetail as SaleListingWithValuation | undefined)?.apartment_profile
  const apartmentProfileGroups = apartmentProfile ? groupApartmentProfile(apartmentProfile) : []
  const apartmentProfileCompleteness = apartmentProfile ? profileCompleteness(apartmentProfile) : { completed: 0, total: PROFILE_COMPLETENESS_FIELDS.length }
  const valuationInput = (saleDetail as SaleListingWithValuation | undefined)?.valuation?.input
  const valuationFacts = valuationInput?.facts ?? (saleDetail as SaleListingWithValuation | undefined)?.valuation_inputs?.facts ?? []
  const valuationDisplayFacts = valuationInputDisplayFacts(valuationInput, valuationFacts)
  const renovationForecastRows = valuationRenovations?.next_40_years ?? []
  const keyRenovationGrid = keyRenovationGridItems(renovationRows, renovationForecastRows)
  const houseOverview = houseOverviewResult?.overview ?? (saleDetail as SaleListingWithValuation | undefined)?.house_overview
  const transactionDate = matchedTransaction?.first_seen_at ? fmtDate(matchedTransaction.first_seen_at) : undefined
  const mapLatitude = location.latitude ?? building.location.latitude
  const mapLongitude = location.longitude ?? building.location.longitude
  const mapLabel = [location.street_address || building.location.street_address || d.headline, location.postal || building.location.postal, location.city || building.location.city].filter(Boolean).join(', ')
  const extractRenovations = useMutation({
    mutationFn: async () => {
      if (!saleDetail) throw new Error('Renovation extraction is only available for sale listings')
      return extractSaleListingRenovations(saleDetail.id)
    },
    onSuccess: async response => {
      setRenovationExtractionResult(response.data)
      await onRefresh?.()
    },
  })
  const renovationExtractionMessage = extractRenovations.isError
    ? (extractRenovations.error as Error)?.message ?? 'Renovation extraction failed'
    : renovationExtractionResult
      ? `${renovationExtractionResult.items?.length ?? 0} extracted · ${renovationExtractionResult.model}`
      : undefined
  const renovationActions = saleDetail ? (
    <div className="listing-section-actions">
      {renovationExtractionMessage && (
        <span className={`listing-section-status${extractRenovations.isError ? ' listing-section-status--error' : ''}`}>
          {renovationExtractionMessage}
        </span>
      )}
      <button type="button" className="listing-action-button" onClick={() => extractRenovations.mutate()} disabled={extractRenovations.isPending}>
        {extractRenovations.isPending ? 'Running…' : 'Run extraction'}
      </button>
    </div>
  ) : undefined
  const extractDescription = useMutation({
    mutationFn: async () => {
      if (!saleDetail) throw new Error('Description extraction is only available for sale listings')
      return extractSaleListingDescription(saleDetail.id)
    },
    onSuccess: async response => {
      setDescriptionExtractionResult(response.data)
      await onRefresh?.()
    },
  })
  const descriptionExtractionMessage = extractDescription.isError
    ? (extractDescription.error as Error)?.message ?? 'Description extraction failed'
    : descriptionExtractionResult
      ? `${descriptionExtractionResult.items?.length ?? 0} extracted · ${descriptionExtractionResult.model}`
      : undefined
  const extractValuationInputs = useMutation({
    mutationFn: async () => {
      if (!saleDetail) throw new Error('Valuation input extraction is only available for sale listings')
      return extractSaleListingValuationInputs(saleDetail.id)
    },
    onSuccess: async response => {
      setValuationInputExtractionResult(response.data)
      await onRefresh?.()
    },
  })
  const valuationInputExtractionMessage = extractValuationInputs.isError
    ? (extractValuationInputs.error as Error)?.message ?? 'Valuation input extraction failed'
    : valuationInputExtractionResult
      ? `${valuationInputExtractionResult.facts?.length ?? 0} facts · ${valuationInputExtractionResult.model}`
      : undefined
  const projectApartmentProfile = useMutation({
    mutationFn: async () => {
      if (!saleDetail) throw new Error('Apartment profile projection is only available for sale listings')
      return projectSaleListingApartmentProfile(saleDetail.id)
    },
    onSuccess: async response => {
      setApartmentProfileProjectionResult(response.data)
      await onRefresh?.()
    },
  })
  const generateHouseOverview = useMutation({
    mutationFn: async () => {
      if (!saleDetail) throw new Error('House overview generation is only available for sale listings')
      return generateSaleListingHouseOverview(saleDetail.id)
    },
    onSuccess: response => {
      setHouseOverviewResult(response.data)
    },
  })
  const populateEverything = useMutation({
    mutationFn: async () => {
      if (!saleDetail) throw new Error('Profile population is only available for sale listings')
      const renovations = await extractSaleListingRenovations(saleDetail.id)
      const description = await extractSaleListingDescription(saleDetail.id)
      const valuationInputs = await extractSaleListingValuationInputs(saleDetail.id)
      const profile = await projectSaleListingApartmentProfile(saleDetail.id)
      const overview = await generateSaleListingHouseOverview(saleDetail.id)
      return { renovations, description, valuationInputs, profile, overview }
    },
    onSuccess: async result => {
      setRenovationExtractionResult(result.renovations.data)
      setDescriptionExtractionResult(result.description.data)
      setValuationInputExtractionResult(result.valuationInputs.data)
      setApartmentProfileProjectionResult(result.profile.data)
      setHouseOverviewResult(result.overview.data)
      await onRefresh?.()
    },
  })
  const apartmentProfileMessage = populateEverything.isError
    ? (populateEverything.error as Error)?.message ?? 'Populate all failed'
    : projectApartmentProfile.isError
      ? (projectApartmentProfile.error as Error)?.message ?? 'Profile projection failed'
      : generateHouseOverview.isError
        ? (generateHouseOverview.error as Error)?.message ?? 'House overview generation failed'
      : populateEverything.data
        ? 'profile and overview populated from provider fields and AI extraction'
        : apartmentProfileProjectionResult
          ? 'profile projected'
          : undefined
  const offerActions = saleDetail ? (
    <div className="listing-section-actions">
      {(descriptionExtractionMessage || valuationInputExtractionMessage || apartmentProfileMessage) && (
        <span className={`listing-section-status${extractDescription.isError || extractValuationInputs.isError || projectApartmentProfile.isError || generateHouseOverview.isError || populateEverything.isError ? ' listing-section-status--error' : ''}`}>
          {[descriptionExtractionMessage, valuationInputExtractionMessage, apartmentProfileMessage].filter(Boolean).join(' · ')}
        </span>
      )}
      <button type="button" className="listing-action-button" onClick={() => populateEverything.mutate()} disabled={populateEverything.isPending || extractValuationInputs.isPending || extractDescription.isPending || extractRenovations.isPending || projectApartmentProfile.isPending || generateHouseOverview.isPending}>
        {populateEverything.isPending ? 'Running…' : 'Populate all'}
      </button>
      <button type="button" className="listing-action-button" onClick={() => generateHouseOverview.mutate()} disabled={generateHouseOverview.isPending}>
        {generateHouseOverview.isPending ? 'Generating…' : 'Generate overview'}
      </button>
      <button type="button" className="listing-action-button" onClick={() => projectApartmentProfile.mutate()} disabled={projectApartmentProfile.isPending}>
        {projectApartmentProfile.isPending ? 'Projecting…' : 'Build profile'}
      </button>
      <button type="button" className="listing-action-button" onClick={() => extractValuationInputs.mutate()} disabled={extractValuationInputs.isPending}>
        {extractValuationInputs.isPending ? 'Running…' : 'Extract inputs'}
      </button>
      <button type="button" className="listing-action-button" onClick={() => extractDescription.mutate()} disabled={extractDescription.isPending}>
        {extractDescription.isPending ? 'Running…' : 'Extract description'}
      </button>
    </div>
  ) : undefined

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
              <button type="button" className="listing-price-transaction" onClick={() => setTransactionOpen(true)}>
                Sold {fmtPrice(matchedTransaction.price)}
                {transactionDate ? ` · ${transactionDate}` : ''}
              </button>
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
          {unit.balcony != null && <Fact label="Balcony" value={unit.balcony ? 'Yes' : 'No'} />}
          {unit.sauna != null && <Fact label="Sauna" value={unit.sauna ? 'Yes' : 'No'} />}
          {(site?.plot_ownership_type || site?.plot_type) && <Fact label="Plot" value={plotSummary(site)} />}
        </div>
        {mapLatitude != null && mapLongitude != null && (
          <ListingLocationMap latitude={mapLatitude} longitude={mapLongitude} label={mapLabel || 'Listing location'} />
        )}
        <div className="listing-body">
          {saleDetail && (
            <Section title="Profile Workbench" actions={offerActions}>
              <div className="profile-workbench">
                <div className="profile-workbench-summary">
                  <div>
                    <div className="profile-workbench-title">{apartmentProfile ? 'Typed apartment profile' : 'No typed apartment profile yet'}</div>
                    <p className="profile-workbench-copy">
                      {apartmentProfile
                        ? 'Canonical apartment facts are projected from provider fields and targeted AI extraction. Use this as the main valuation input surface.'
                        : 'Populate the profile to canonicalize apartment-level facts before reviewing value, renovation exposure, and source evidence.'}
                    </p>
                  </div>
                  <div className="profile-workbench-metrics">
                    <span>{apartmentProfileCompleteness.completed} / {apartmentProfileCompleteness.total} fields</span>
                    {apartmentProfile?.confidence && <span>{apartmentProfile.confidence} confidence</span>}
                    {apartmentProfile?.updated_at && <span>{fmtDateTime(apartmentProfile.updated_at)}</span>}
                  </div>
                </div>
                <div className="profile-workbench-steps">
                  <ProfileStep label="Source rows linked" done={(saleDetail.canonical.source_count ?? 0) > 0} detail={saleDetail.canonical.source_count != null ? `${saleDetail.canonical.source_count} source row${saleDetail.canonical.source_count === 1 ? '' : 's'}` : undefined} />
                  <ProfileStep label="Description parsed" done={(valuationFacts?.length ?? 0) > 0 || !!descriptionExtractionResult} detail={(valuationFacts?.length ?? 0) > 0 ? `${valuationFacts.length} valuation facts` : descriptionExtractionMessage} />
                  <ProfileStep label="Renovations structured" done={renovationRows.length > 0 || !!renovationExtractionResult} detail={renovationRows.length > 0 ? `${renovationRows.length} renovation facts` : renovationExtractionMessage} />
                  <ProfileStep label="Apartment profile projected" done={!!apartmentProfile} detail={apartmentProfile?.updated_at ? fmtDateTime(apartmentProfile.updated_at) : apartmentProfileMessage} />
                </div>
              </div>
            </Section>
          )}
          {saleDetail && (
            <Section title="Key Renovation Dates" actions={!valuationBrief && !offerAssessment ? renovationActions : undefined}>
              {keyRenovationGrid.length ? (
                <div className="renovation-date-grid">
                  {keyRenovationGrid.map(item => (
                    <div className={`renovation-date-card renovation-date-card--${renovationGridTone(item)}`} key={item.category}>
                      <div className="renovation-date-card-head">
                        <strong>{renovationStatusLabel(item.category)}</strong>
                        <span>{renovationGridStatus(item)}</span>
                      </div>
                      <div className="renovation-date-card-body">
                        <RenovationDate label="Done" value={item.done?.year} detail={item.done?.kind} />
                        <RenovationDate label="Planned" value={item.planned?.year} detail={item.planned?.kind} />
                        <RenovationDate label="Expected" value={renovationForecastDate(item.forecast)} detail={item.forecast?.severity && `${item.forecast.severity} impact`} />
                      </div>
                      {renovationGridExplanation(item) && <p>{renovationGridExplanation(item)}</p>}
                    </div>
                  ))}
                </div>
              ) : (
                <div className="listing-empty-state">Run renovation extraction to parse key dates into structured facts</div>
              )}
            </Section>
          )}
          {saleDetail && (
            <Section title="House Overview" actions={!houseOverview ? offerActions : undefined}>
              {houseOverview ? (
                <div className="house-overview">
                  <div className={`house-overview-hero house-overview-hero--${houseOverview.renovation_readiness || 'unclear'}`}>
                    <div>
                      <div className="house-overview-title">{houseOverview.headline || 'Generated building overview'}</div>
                      {houseOverview.summary && <p>{houseOverview.summary}</p>}
                    </div>
                    <div className="house-overview-meta">
                      {houseOverview.renovation_readiness && <span>{houseOverview.renovation_readiness} readiness</span>}
                      {houseOverview.confidence && <span>{houseOverview.confidence} confidence</span>}
                      {houseOverview.generated_at && <span>{fmtDateTime(houseOverview.generated_at)}</span>}
                    </div>
                  </div>
                  {houseOverview.expensive_window && (
                    <div className="house-overview-window">
                      <span>Expensive window</span>
                      <strong>{houseOverview.expensive_window}</strong>
                    </div>
                  )}
                  <div className="house-overview-columns">
                    <HouseOverviewList title="Strengths" items={houseOverview.key_strengths} empty="No strong positives extracted" />
                    <HouseOverviewList title="Risks" items={houseOverview.key_risks} empty="No major risks extracted" />
                    <HouseOverviewList title="Evidence gaps" items={houseOverview.evidence_gaps} empty="No major gaps listed" />
                  </div>
                </div>
              ) : (
                <div className="listing-empty-state">Generate an overview after the structured renovation and profile facts are populated</div>
              )}
            </Section>
          )}
          <Section title="Provenance & Timing">
            <div className="listing-table">
              {saleDetail?.canonical.offering_id && <Row label="Offering ID" value={saleDetail.canonical.offering_id} />}
              {saleDetail?.canonical.source_count != null && <Row label="Combined sources" value={`${saleDetail.canonical.source_count} linked source row${saleDetail.canonical.source_count === 1 ? '' : 's'}`} highlight={saleDetail.canonical.source_count > 1} />}
              {saleDetail?.canonical.merge_decision_count ? <Row label="Merged duplicate offerings" value={String(saleDetail.canonical.merge_decision_count)} highlight /> : null}
              {saleDetail?.canonical.merged_from?.length ? <Row label="Merged from" value={saleDetail.canonical.merged_from.join(', ')} /> : null}
              <Row label="Provider" value={providerLabel(d.source.provider)} />
              <Row label="Source kind" value={d.source.kind} />
              {d.source.external_id && <Row label="External ID" value={d.source.external_id} />}
              {d.source.friendly_id && <Row label="Friendly ID" value={d.source.friendly_id} />}
              {commercial.status && <Row label="Status" value={commercial.status} />}
              {commercial.booking_status && <Row label="Booking status" value={commercial.booking_status} />}
              {commercial.published_at && <Row label="Published" value={fmtDateTime(commercial.published_at)} />}
              {commercial.first_seen_at && <Row label="First seen" value={fmtDateTime(commercial.first_seen_at)} />}
              {commercial.last_seen_at && <Row label="Last seen" value={fmtDateTime(commercial.last_seen_at)} />}
              {commercial.unpublished_at && <Row label="Unpublished" value={fmtDateTime(commercial.unpublished_at)} />}
              {commercial.days_on_market != null && <Row label="Days on market" value={String(commercial.days_on_market)} />}
              {commercial.can_receive_leads != null && <Row label="Can receive leads" value={fmtBool(commercial.can_receive_leads)} />}
              {commercial.map_visible != null && <Row label="Map visible" value={fmtBool(commercial.map_visible)} />}
              {d.source.original_url && <Row label="Original URL" value={<a href={d.source.original_url} target="_blank" rel="noopener noreferrer">{d.source.original_url}</a>} />}
            </div>
          </Section>
          {saleDetail?.source_records?.length ? (
            <Section title="Linked Sources">
              <div className="listing-table">
                {saleDetail.source_records.map(record => (
                  <Row
                    key={record.id}
                    label={`${providerLabel(record.provider)} ${record.kind}`}
                    value={(
                      <div className="source-row-value">
                        <span>
                          {[
                            record.headline,
                            record.last_seen_at && `last seen ${fmtDateTime(record.last_seen_at)}`,
                            record.link_score > 0 && `${record.link_status} ${record.link_score}`,
                          ].filter(Boolean).join(' · ')}
                        </span>
                        <a
                          href={`/api/v1/sale-listings/${encodeURIComponent(saleDetail.id)}/source-records/${encodeURIComponent(record.id)}/raw`}
                          target="_blank"
                          rel="noopener noreferrer"
                          className="source-json-link"
                        >
                          JSON
                        </a>
                      </div>
                    )}
                  />
                ))}
              </div>
            </Section>
          ) : null}
          {texts?.description && (
            <Section title="Description">
              <TextBlock text={texts.description} />
            </Section>
          )}
          {valuationBrief && (
            <Section title="At a Glance" actions={offerActions}>
              <div className="valuation-brief">
                <div className={`brief-verdict brief-verdict--${valuationBrief.verdict}`}>
                  <div>
                    <div className="brief-verdict-label">{valuationBrief.label || renovationStatusLabel(valuationBrief.verdict)}</div>
                    {valuationBrief.explanation && <div className="brief-verdict-text">{valuationBrief.explanation}</div>}
                  </div>
                  <div className="brief-meta">
                    {valuationBrief.building_risk && <span>{valuationBrief.building_risk} building risk</span>}
                    <span>{valuationBrief.confidence} confidence</span>
                  </div>
                </div>
                <div className="brief-grid">
                  <BriefPanel title="Cost window">
                    {valuationBrief.expensive_windows?.length ? (
                      <div className={`brief-window brief-window--${valuationBrief.expensive_windows[0].severity}`}>
                        <strong>{valuationBrief.expensive_windows[0].label}</strong>
                        <span>{valuationBrief.expensive_windows[0].severity} impact</span>
                        {valuationBrief.expensive_windows[0].reasons?.length ? <p>{valuationBrief.expensive_windows[0].reasons.map(renovationStatusLabel).join(', ')}</p> : null}
                      </div>
                    ) : (
                      <span className="brief-empty">No near expensive window</span>
                    )}
                  </BriefPanel>
                  <BriefPanel title="Top risks">
                    {valuationBrief.top_risks?.length ? valuationBrief.top_risks.slice(0, 3).map(signal => <BriefSignalRow signal={signal} key={signal.key} />) : <span className="brief-empty">No major risk signal</span>}
                  </BriefPanel>
                  <BriefPanel title="Supports">
                    {valuationBrief.top_positives?.length ? valuationBrief.top_positives.slice(0, 3).map(signal => <BriefSignalRow signal={signal} key={signal.key} />) : <span className="brief-empty">No strong positive signal</span>}
                  </BriefPanel>
                </div>
                {valuationBrief.key_renovations?.length ? (
                  <div className="brief-renovations">
                    {valuationBrief.key_renovations.slice(0, 9).map(item => (
                      <div className={`brief-renovation brief-renovation--${item.status}`} key={item.category}>
                        <span>{renovationStatusLabel(item.category)}</span>
                        <strong>{briefRenovationWhen(item)}</strong>
                      </div>
                    ))}
                  </div>
                ) : null}
                {valuationBrief.missing_evidence?.length ? (
                  <div className="brief-missing">Missing: {valuationBrief.missing_evidence.slice(0, 6).map(renovationStatusLabel).join(', ')}</div>
                ) : null}
              </div>
            </Section>
          )}
          {offerAssessment && (
            <Section title="Offer Assessment" actions={!valuationBrief ? offerActions : undefined}>
              <div className="offer-assessment">
                <div className={`offer-verdict offer-verdict--${offerAssessment.verdict}`}>
                  <div>
                    <div className="offer-verdict-label">{renovationStatusLabel(offerAssessment.verdict)}</div>
                    <div className="offer-verdict-text">{offerAssessment.explanation}</div>
                  </div>
                  <span>{offerAssessment.confidence} confidence</span>
                </div>
                <div className="offer-range-grid">
                  <OfferRange label="Market value" range={offerAssessment.market_value_range} />
                  <OfferRange label="Risk-adjusted" range={offerAssessment.risk_adjusted_value_range} />
                  <OfferRange label="Offer range" range={offerAssessment.recommended_offer_range} />
                  <OfferRange label="Renovation reserve" range={offerAssessment.renovation_risk_reserve} />
                </div>
                {offerAssessment.main_reasons?.length ? (
                  <div className="offer-reasons">
                    {offerAssessment.main_reasons.map(reason => (
                      <div className={`offer-reason offer-reason--${reason.direction}`} key={reason.key}>
                        <span>{reason.severity}</span>
                        <p>{reason.explanation}</p>
                      </div>
                    ))}
                  </div>
                ) : null}
                {offerAssessment.missing?.length ? (
                  <div className="offer-missing">Missing: {offerAssessment.missing.map(renovationStatusLabel).join(', ')}</div>
                ) : null}
              </div>
            </Section>
          )}
          {saleDetail && (
            <Section title="Apartment Profile" actions={!valuationBrief && !offerAssessment ? offerActions : undefined}>
              {apartmentProfileGroups.length ? (
                <div className="apartment-profile">
                  {apartmentProfileGroups.map(group => (
                    <div className="apartment-profile-group" key={group.key}>
                      <div className="apartment-profile-group-title">{group.title}</div>
                      <div className="apartment-profile-grid">
                        {group.facts.map(fact => (
                          <div className={`apartment-profile-fact${fact.tone ? ` apartment-profile-fact--${fact.tone}` : ''}`} key={fact.key}>
                            <span>{fact.label}</span>
                            <strong>{fact.value}</strong>
                          </div>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="listing-empty-state">Build the typed profile from provider fields and extracted inputs</div>
              )}
            </Section>
          )}
          {saleDetail && (
            <Section title="Valuation Inputs" actions={!offerAssessment ? offerActions : undefined}>
              {valuationDisplayFacts.length ? (
                <div className="valuation-inputs">
                  {groupValuationFacts(valuationDisplayFacts).map(group => (
                    <div className="valuation-input-group" key={group.section}>
                      <div className="valuation-input-section">{renovationStatusLabel(group.section)}</div>
                      <div className="valuation-input-grid">
                        {group.facts.map(fact => (
                          <div className="valuation-input-fact" key={`${fact.source || 'source'}-${fact.section}-${fact.key}`}>
                            <span>{renovationStatusLabel(fact.key)}</span>
                            <strong>{formatValuationFactValue(fact)}</strong>
                            {(fact.evidence || fact.confidence != null) && (
                              <p>{[fact.evidence, fact.confidence != null ? `${Math.round(fact.confidence * 100)}%` : undefined].filter(Boolean).join(' · ')}</p>
                            )}
                          </div>
                        ))}
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="listing-empty-state">Run extraction to parse canonical valuation inputs</div>
              )}
            </Section>
          )}
          {(commercial.rent != null || commercial.asking_price != null || commercial.debt_free_price != null ||
            commercial.previous_asking_price != null || commercial.previous_debt_free_price != null ||
            commercial.debt_share_amount != null || matchedTransaction?.price != null || commercial.security_deposit ||
            charges?.maintenance_monthly != null || charges?.total_monthly != null || charges?.water != null ||
            charges?.parking != null || charges?.sauna != null || charges?.electricity || charges?.heating ||
            charges?.notes || commercial.fees_info || commercial.financing_fee_interest_only_period ||
            commercial.financing_fee_interest_only_start_date || commercial.financing_fee_interest_only_end_date ||
            commercial.open_bidding_in_use != null || commercial.open_bidding_starting_selling_price != null ||
            commercial.open_bidding_starting_debt_free_price != null || commercial.open_bidding_latest_offer != null ||
            commercial.minimum_term_months != null || commercial.pets_allowed != null || commercial.furnished != null ||
            commercial.fixed_term != null || commercial.ownership_type || commercial.development_phase ||
            commercial.new_development != null || commercial.other_terms || texts?.charges) && (
            <Section title="Pricing & Charges">
              <div className="listing-table">
                {commercial.rent != null && <Row label="Rent" value={`${fmtPrice(commercial.rent)}${commercial.rent_period ? ` / ${commercial.rent_period}` : ''}`} highlight />}
                {commercial.asking_price != null && <Row label="Asking price" value={fmtPrice(commercial.asking_price)} highlight={!isRental} />}
                {commercial.debt_free_price != null && <Row label="Debt-free price" value={fmtPrice(commercial.debt_free_price)} />}
                {commercial.previous_asking_price != null && <Row label="Previous asking price" value={fmtPrice(commercial.previous_asking_price)} />}
                {commercial.previous_debt_free_price != null && <Row label="Previous debt-free price" value={fmtPrice(commercial.previous_debt_free_price)} />}
                {matchedTransaction?.price != null && <Row label="Matched sale price" value={fmtPrice(matchedTransaction.price)} highlight />}
                {matchedTransaction?.price_per_m2 != null && <Row label="Matched sale / m²" value={fmtEur(matchedTransaction.price_per_m2)} />}
                {matchedTransaction?.condition && <Row label="Transaction condition" value={matchedTransaction.condition} />}
                {matchedTransaction?.floor && <Row label="Transaction floor" value={matchedTransaction.floor} />}
                {matchedTransaction?.plot && <Row label="Transaction plot" value={matchedTransaction.plot} />}
                {matchedTransaction?.energy_class && <Row label="Transaction energy" value={matchedTransaction.energy_class} />}
                {matchedTransaction?.period_identifier && <Row label="Transaction period" value={matchedTransaction.period_identifier} />}
                {matchedTransaction?.description && <Row label="Transaction layout" value={matchedTransaction.description} />}
                {matchedTransaction?.match_score != null && <Row label="Match confidence" value={[matchedTransaction.match_confidence, String(matchedTransaction.match_score)].filter(Boolean).join(' · ')} />}
                {commercial.debt_share_amount != null && <Row label="Debt share" value={fmtPrice(commercial.debt_share_amount)} />}
                {commercial.debt_share_additional_info && <Row label="Debt share info" value={commercial.debt_share_additional_info} />}
                {charges?.maintenance_monthly != null && <Row label="Maintenance charge" value={`${fmtEur(charges.maintenance_monthly)} / mo`} />}
                {charges?.total_monthly != null && charges.total_monthly !== charges.maintenance_monthly && <Row label="Total monthly charge" value={`${fmtEur(charges.total_monthly)} / mo`} />}
                {charges?.water != null && <Row label="Water charge" value={`${fmtEur(charges.water)} / mo`} />}
                {charges?.parking != null && <Row label="Parking charge" value={`${fmtEur(charges.parking)} / mo`} />}
                {charges?.sauna != null && <Row label="Sauna charge" value={`${fmtEur(charges.sauna)} / mo`} />}
                {charges?.electricity && <Row label="Electricity" value={charges.electricity} />}
                {charges?.heating && <Row label="Heating charge" value={charges.heating} />}
                {charges?.notes && <Row label="Charge notes" value={charges.notes} />}
                {commercial.fees_info && <Row label="Fees info" value={commercial.fees_info} />}
                {commercial.financing_fee_interest_only_period && <Row label="Interest-only period" value={commercial.financing_fee_interest_only_period} />}
                {commercial.financing_fee_interest_only_start_date && <Row label="Interest-only starts" value={commercial.financing_fee_interest_only_start_date} />}
                {commercial.financing_fee_interest_only_end_date && <Row label="Interest-only ends" value={commercial.financing_fee_interest_only_end_date} />}
                {commercial.open_bidding_in_use != null && <Row label="Open bidding" value={fmtBool(commercial.open_bidding_in_use)} />}
                {commercial.open_bidding_starting_selling_price != null && <Row label="Bidding start price" value={fmtPrice(commercial.open_bidding_starting_selling_price)} />}
                {commercial.open_bidding_starting_debt_free_price != null && <Row label="Bidding start debt-free" value={fmtPrice(commercial.open_bidding_starting_debt_free_price)} />}
                {commercial.open_bidding_latest_offer != null && <Row label="Latest bid" value={fmtPrice(commercial.open_bidding_latest_offer)} />}
                {commercial.security_deposit && <Row label="Security deposit" value={commercial.security_deposit} />}
                {commercial.minimum_term_months != null && <Row label="Minimum term" value={`${commercial.minimum_term_months} months`} />}
                {commercial.pets_allowed != null && <Row label="Pets allowed" value={commercial.pets_allowed ? 'Yes' : 'No'} />}
                {commercial.furnished != null && <Row label="Furnished" value={fmtBool(commercial.furnished)} />}
                {commercial.fixed_term != null && <Row label="Fixed term" value={fmtBool(commercial.fixed_term)} />}
                {commercial.ownership_type && <Row label="Ownership type" value={commercial.ownership_type} />}
                {commercial.development_phase && <Row label="Development phase" value={commercial.development_phase} />}
                {commercial.new_development != null && <Row label="New development" value={fmtBool(commercial.new_development)} />}
                {commercial.other_terms && <Row label="Other terms" value={commercial.other_terms} />}
              </div>
              {texts?.charges && <TextBlock text={texts.charges} muted />}
            </Section>
          )}
          <Section title="Unit">
            <div className="listing-table">
              {unit.room_layout && <Row label="Room layout" value={unit.room_layout} />}
              {unit.rooms_count != null && <Row label="Rooms" value={String(unit.rooms_count)} />}
              {unit.bedrooms_count != null && <Row label="Bedrooms" value={String(unit.bedrooms_count)} />}
              {unit.area_m2 != null && <Row label="Area" value={`${unit.area_m2.toFixed(1)} m²`} />}
              {unit.living_area_m2 != null && <Row label="Living area" value={`${unit.living_area_m2.toFixed(1)} m²`} />}
              {unit.total_area_m2 != null && <Row label="Total area" value={`${unit.total_area_m2.toFixed(1)} m²`} />}
              {unit.other_area_m2 != null && <Row label="Other area" value={`${unit.other_area_m2.toFixed(1)} m²`} />}
              {commercial.price_per_m2 != null && <Row label="Price / m²" value={fmtEur(commercial.price_per_m2)} />}
              {unit.floor_level != null && <Row label="Floor" value={building.floor_count != null ? `${unit.floor_level} / ${building.floor_count}` : String(unit.floor_level)} />}
              {unit.property_type && <Row label="Property type" value={unit.property_type} />}
              {unit.property_subtype && <Row label="Property subtype" value={unit.property_subtype} />}
              {unit.condition && <Row label="Condition" value={unit.condition} />}
              {unit.balcony != null && <Row label="Balcony" value={fmtBool(unit.balcony)} />}
              {unit.balcony_description && <Row label="Balcony details" value={unit.balcony_description} />}
              {unit.sauna != null && <Row label="Sauna" value={fmtBool(unit.sauna)} />}
              {unit.sauna_description && <Row label="Sauna details" value={unit.sauna_description} />}
              {unit.parking && <Row label="Parking" value={unit.parking} />}
              {unit.availability && <Row label="Availability" value={unit.availability} />}
              {unit.kitchen_description && <Row label="Kitchen" value={unit.kitchen_description} />}
              {unit.bathroom_description && <Row label="Bathroom" value={unit.bathroom_description} />}
              {unit.storage_description && <Row label="Storage" value={unit.storage_description} />}
              {unit.views_description && <Row label="Views" value={unit.views_description} />}
              {unit.floor_materials_description && <Row label="Floor materials" value={unit.floor_materials_description} />}
              {unit.wall_materials_description && <Row label="Wall materials" value={unit.wall_materials_description} />}
              {unit.appliances?.length ? <Row label="Appliances" value={unit.appliances.join(', ')} /> : null}
              {unit.features?.length ? <Row label="Features" value={unit.features.join(', ')} /> : null}
            </div>
          </Section>
          <Section title="Housing Company Details">
            <div className="listing-table">
              {saleDetail?.canonical?.housing_company_id && <Row label="Housing company" value={<Link to={`/housing-company/${encodeURIComponent(saleDetail.canonical.housing_company_id)}`}>Open housing company page</Link>} highlight />}
              {building.housing_company && <Row label="Housing company" value={building.housing_company} />}
              {building.identity?.key && <Row label="Housing company identity" value={`${building.identity.key} · ${Math.round(building.identity.confidence * 100)}%`} />}
              {building.business_id && <Row label="Business ID" value={building.business_id} />}
              {building.build_year != null && <Row label="Year built" value={String(building.build_year)} />}
              {building.construction_year != null && <Row label="Construction year" value={String(building.construction_year)} />}
              {building.building_type && <Row label="Building type" value={building.building_type} />}
              {building.building_subtype && <Row label="Building subtype" value={building.building_subtype} />}
              {building.energy_class && <Row label="Energy class" value={building.energy_class} />}
              {building.heating && <Row label="Heating" value={building.heating} />}
              {building.heating_description && <Row label="Heating details" value={building.heating_description} />}
              {building.heating_fuel && <Row label="Heating fuel" value={building.heating_fuel} />}
              {building.apartment_count != null && <Row label="Apartments" value={String(building.apartment_count)} />}
              {building.business_premise_count != null && <Row label="Business premises" value={String(building.business_premise_count)} />}
              {building.floor_count != null && <Row label="Floors" value={String(building.floor_count)} />}
              {building.elevator != null && <Row label="Elevator" value={building.elevator ? 'Yes' : 'No'} />}
              {building.sauna != null && <Row label="Building sauna" value={fmtBool(building.sauna)} />}
              {building.car_storage && <Row label="Car storage" value={building.car_storage} />}
              {building.building_material && <Row label="Building material" value={building.building_material} />}
              {building.frame_construction_method && <Row label="Frame construction" value={building.frame_construction_method} />}
              {building.wall_structure && <Row label="Wall structure" value={building.wall_structure} />}
              {building.roof_type && <Row label="Roof type" value={building.roof_type} />}
              {building.roof_material && <Row label="Roof material" value={building.roof_material} />}
              {building.management_method && <Row label="Management" value={building.management_method} />}
              {building.property_manager && <Row label="Property manager" value={building.property_manager} />}
              {building.maintenance_responsibility && <Row label="Maintenance responsibility" value={building.maintenance_responsibility} />}
              {building.connectivity && <Row label="Connectivity" value={building.connectivity} />}
              {building.common_areas && <Row label="Common areas" value={building.common_areas} />}
              {building.other_info && <Row label="Other building info" value={building.other_info} />}
            </div>
          </Section>
          {(renovationRows.length > 0 || saleDetail) && (
            <Section title="Housing Company Renovations" actions={renovationActions}>
              {renovationRows.length > 0 ? (
                <div className="renovation-list">
                  {renovationRows.map((item, index) => (
                    <div className="renovation-row" key={`${item.kind}-${item.year || 'no-year'}-${item.status}-${index}`}>
                      <div className="renovation-year">{item.year || '—'}</div>
                      <div className="renovation-main">
                        <div className="renovation-kind">{item.kind}</div>
                        <div className={`renovation-status renovation-status--${item.status}`}>{item.status}</div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="listing-empty-state">No structured renovations</div>
              )}
            </Section>
          )}
          {saleDetail && (
            <Section title="Next 40 Years">
              {renovationForecastRows.length > 0 ? (
                <div className="renovation-list">
                  {renovationForecastRows.map((item, index) => (
                    <div className="renovation-row renovation-row--forecast" key={`${item.category}-${item.year || item.year_range || 'no-year'}-${item.status}-${index}`}>
                      <div className="renovation-year">{fmtRenovationNeedYear(item)}</div>
                      <div className="renovation-main">
                        <div className="renovation-kind">{item.category}</div>
                        <div className="renovation-forecast-meta">
                          <span className={`renovation-status renovation-status--${item.status}`}>{renovationStatusLabel(item.status)}</span>
                          <span>{item.severity} impact</span>
                          {item.confidence && <span>{item.confidence} confidence</span>}
                          {item.scope && item.scope !== 'unknown' && <span>{item.scope}</span>}
                          {item.stage && item.stage !== 'unknown' && <span>{renovationStatusLabel(item.stage)}</span>}
                          {item.component && <span>{renovationStatusLabel(item.component)}</span>}
                          {item.cycle_years != null && <span>{item.cycle_years} year cycle</span>}
                          {item.basis_year != null && <span>from {item.basis_year}</span>}
                          {item.cost_estimate_eur != null && <span>{fmtPrice(item.cost_estimate_eur)}</span>}
                        </div>
                        {item.price_mechanisms?.length ? (
                          <div className="renovation-forecast-tags">
                            {item.price_mechanisms.slice(0, 4).map(value => <span key={value}>{value}</span>)}
                          </div>
                        ) : null}
                        <div className="renovation-explanation">{item.explanation}</div>
                      </div>
                    </div>
                  ))}
                </div>
              ) : (
                <div className="listing-empty-state">Run extraction to project future renovation needs</div>
              )}
            </Section>
          )}
          {site && hasSiteDetails && (
            <Section title="Site & Area">
              <div className="listing-table">
                {site.plot_type && <Row label="Plot type" value={fmtPlotOwnership(site.plot_type)} />}
                {site.plot_ownership_type && <Row label="Plot ownership" value={fmtPlotOwnership(site.plot_ownership_type)} />}
                {site.plot_area_m2 != null && <Row label="Plot area" value={`${site.plot_area_m2.toFixed(0)} m²`} />}
                {site.lot_redemption_info && <Row label="Lot redemption" value={site.lot_redemption_info} />}
                {site.lot_rental_agreement && <Row label="Lot rental agreement" value={site.lot_rental_agreement} />}
                {site.yard && <Row label="Yard" value={site.yard} />}
                {site.shore && <Row label="Shore" value={site.shore} />}
                {site.zoning && <Row label="Zoning" value={site.zoning} />}
                {site.road_access && <Row label="Road access" value={site.road_access} />}
                {site.water_supply && <Row label="Water supply" value={site.water_supply} />}
                {site.water_supply_types?.length ? <Row label="Water supply types" value={site.water_supply_types.join(', ')} /> : null}
                {site.sewer && <Row label="Sewer" value={site.sewer} />}
                {site.services && <Row label="Services" value={site.services} />}
                {site.transport && <Row label="Transport" value={site.transport} />}
                {site.driving_directions && <Row label="Driving directions" value={site.driving_directions} />}
              </div>
            </Section>
          )}
          {texts?.availability && (
            <Section title="Availability">
              <TextBlock text={texts.availability} />
            </Section>
          )}
          {texts?.additional_info && (
            <Section title="Additional Information">
              <TextBlock text={texts.additional_info} />
            </Section>
          )}
          {(texts?.kitchen || texts?.bathroom || texts?.storage || texts?.materials || texts?.amenities || texts?.area || texts?.transport || texts?.building) && (
            <Section title="Source Texts">
              {texts.kitchen && <TextBlock text={texts.kitchen} />}
              {texts.bathroom && <TextBlock text={texts.bathroom} />}
              {texts.storage && <TextBlock text={texts.storage} />}
              {texts.materials && <TextBlock text={texts.materials} />}
              {texts.amenities && <TextBlock text={texts.amenities} />}
              {texts.area && <TextBlock text={texts.area} />}
              {texts.transport && <TextBlock text={texts.transport} />}
              {texts.building && <TextBlock text={texts.building} />}
            </Section>
          )}
          {d.showings?.length ? (
            <Section title="Showings">
              <div className="listing-table">
                {d.showings.map((showing, index) => (
                  <Row key={`${showing.start_at || index}`} label={showing.start_at ? fmtDateTime(showing.start_at) : `Showing ${index + 1}`} value={[showing.end_at && `ends ${fmtDateTime(showing.end_at)}`, showing.info].filter(Boolean).join(' · ') || 'Open'} />
                ))}
              </div>
            </Section>
          ) : null}
          {d.contacts?.length ? (
            <Section title="Contacts">
              <div className="listing-table">
                {d.contacts.map((contact, index) => (
                  <Row key={`${contact.name || contact.email || index}`} label={contact.name || `Contact ${index + 1}`} value={[contact.title, contact.office_name, contact.phone, contact.email].filter(Boolean).join(' · ')} />
                ))}
              </div>
            </Section>
          ) : null}
          {d.links?.length ? (
            <Section title="Links">
              <div className="listing-table">
                {d.links.map(link => (
                  <Row key={link.url} label={link.type || 'Link'} value={<a href={link.url} target="_blank" rel="noopener noreferrer">{link.title || link.url}</a>} />
                ))}
              </div>
            </Section>
          ) : null}
          {d.insights?.items?.length ? (
            <Section title="Insights">
              <div className="listing-table">
                {d.insights.items.map(insight => (
                  <Row key={`${insight.key}-${insight.value}`} label={insight.key} value={[insight.value, insight.source, insight.confidence != null && `${Math.round(insight.confidence * 100)}%`].filter(Boolean).join(' · ')} />
                ))}
              </div>
            </Section>
          ) : null}
          {images.length > 0 && (
            <Section title="Images">
              <div className="listing-image-grid">
                {images.slice(0, 24).map(image => (
                  <a key={image.id || image.url} href={image.variants?.gallery || image.variants?.large || image.url} target="_blank" rel="noopener noreferrer">
                    <img src={image.variants?.thumb || image.variants?.card || image.variants?.large || image.url} alt={image.description || ''} />
                  </a>
                ))}
              </div>
            </Section>
          )}
          <div className="listing-meta-footer">
            <span className="badge badge-default">{d.id}</span>
            {commercial.last_seen_at && (
              <span className="listing-meta-date">
                Last seen {fmtDateTime(commercial.last_seen_at)}
              </span>
            )}
          </div>
        </div>
      </div>
      {matchedTransaction && transactionOpen && (
        <TransactionModal transaction={matchedTransaction} onClose={() => setTransactionOpen(false)} />
      )}
    </div>
  )
}

function BuildingView({ building }: { building: Building }) {
  const details = building.details
  const site = building.site
  const texts = building.texts
  const sourceRecords = building.source_records ?? []
  const related = building.related?.items ?? []
  const renovationRows = renovationItems(details.renovations, texts?.renovations_done, texts?.renovations_planned)
  const mapLatitude = details.location.latitude
  const mapLongitude = details.location.longitude
  const mapLabel = [details.housing_company || details.location.street_address, details.location.postal, details.location.city].filter(Boolean).join(', ')
  const buildingFacts = [
    details.build_year != null && ['Built', String(details.build_year)],
    details.construction_year != null && ['Constructed', String(details.construction_year)],
    details.floor_count != null && ['Floors', String(details.floor_count)],
    details.apartment_count != null && ['Apartments', String(details.apartment_count)],
  ].filter(Boolean) as string[][]
  const hasSiteDetails = !!site && [
    site.plot_type,
    site.plot_ownership_type,
    site.plot_area_m2,
    site.lot_redemption_info,
    site.lot_rental_agreement,
    site.yard,
    site.shore,
    site.zoning,
    site.road_access,
    site.water_supply,
    site.water_supply_types?.length,
    site.sewer,
    site.services,
    site.transport,
    site.driving_directions,
  ].some(value => value != null && value !== '')
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
            <div className="listing-type-pill listing-type-pill--listing">Housing company</div>
            <h1 className="listing-title">{details.housing_company || details.location.street_address || building.id}</h1>
            <div className="listing-location">
              {[details.location.street_address, details.location.postal, details.location.city].filter(Boolean).join(' · ')}
            </div>
          </div>
        </div>
        {buildingFacts.length > 0 && (
          <div className="listing-facts">
            {buildingFacts.map(([label, value]) => <Fact key={label} label={label} value={value} />)}
          </div>
        )}
        {mapLatitude != null && mapLongitude != null && (
          <ListingLocationMap latitude={mapLatitude} longitude={mapLongitude} label={mapLabel || building.id} />
        )}
        <Section title="Housing Company">
          <div className="listing-table">
            {details.location.street_address && <Row label="Address" value={details.location.street_address} />}
            {details.location.postal && <Row label="Postal" value={details.location.postal} />}
            {details.location.city && <Row label="City" value={details.location.city} />}
            {details.housing_company && <Row label="Housing company" value={details.housing_company} />}
            {details.business_id && <Row label="Business ID" value={details.business_id} />}
            {metadataNumber(building.metadata, 'merge_decision_count') ? <Row label="Merged duplicate companies" value={String(metadataNumber(building.metadata, 'merge_decision_count'))} highlight /> : null}
            {metadataStringArray(building.metadata, 'merged_from').length ? <Row label="Merged from" value={metadataStringArray(building.metadata, 'merged_from').join(', ')} /> : null}
            {details.build_year != null && <Row label="Year built" value={String(details.build_year)} />}
            {details.construction_year != null && <Row label="Construction year" value={String(details.construction_year)} />}
            {details.apartment_count != null && <Row label="Apartments" value={String(details.apartment_count)} />}
            {details.business_premise_count != null && <Row label="Business premises" value={String(details.business_premise_count)} />}
            {details.floor_count != null && <Row label="Floors" value={String(details.floor_count)} />}
            {details.elevator != null && <Row label="Elevator" value={fmtBool(details.elevator)} />}
            {details.sauna != null && <Row label="Sauna" value={fmtBool(details.sauna)} />}
            {details.building_type && <Row label="Building type" value={details.building_type} />}
            {details.building_subtype && <Row label="Building subtype" value={details.building_subtype} />}
            {details.heating && <Row label="Heating" value={details.heating} />}
            {details.heating_description && <Row label="Heating details" value={details.heating_description} />}
            {details.heating_fuel && <Row label="Heating fuel" value={details.heating_fuel} />}
            {details.energy_class && <Row label="Energy class" value={details.energy_class} />}
            {details.building_material && <Row label="Material" value={details.building_material} />}
            {details.wall_structure && <Row label="Wall structure" value={details.wall_structure} />}
            {details.frame_construction_method && <Row label="Frame" value={details.frame_construction_method} />}
            {details.roof_type && <Row label="Roof type" value={details.roof_type} />}
            {details.roof_material && <Row label="Roof material" value={details.roof_material} />}
            {details.management_method && <Row label="Management" value={details.management_method} />}
            {details.property_manager && <Row label="Property manager" value={details.property_manager} />}
            {details.maintenance_responsibility && <Row label="Maintenance responsibility" value={details.maintenance_responsibility} />}
            {details.connectivity && <Row label="Connectivity" value={details.connectivity} />}
            {details.common_areas && <Row label="Common areas" value={details.common_areas} />}
            {details.car_storage && <Row label="Car storage" value={details.car_storage} />}
            {details.other_info && <Row label="Other info" value={details.other_info} />}
          </div>
        </Section>
        {renovationRows.length > 0 && (
          <Section title="Renovations">
            <div className="renovation-list">
              {renovationRows.map((item, index) => (
                <div className="renovation-row" key={`${item.kind}-${item.year || 'no-year'}-${item.status}-${index}`}>
                  <div className="renovation-year">{item.year || '—'}</div>
                  <div className="renovation-main">
                    <div className="renovation-kind">{item.kind}</div>
                    <div className={`renovation-status renovation-status--${item.status}`}>{item.status}</div>
                  </div>
                </div>
              ))}
            </div>
          </Section>
        )}
        {site && hasSiteDetails && (
          <Section title="Site & Area">
            <div className="listing-table">
              {site.plot_type && <Row label="Plot type" value={fmtPlotOwnership(site.plot_type)} />}
              {site.plot_ownership_type && <Row label="Plot ownership" value={fmtPlotOwnership(site.plot_ownership_type)} />}
              {site.plot_area_m2 != null && <Row label="Plot area" value={`${site.plot_area_m2.toFixed(0)} m²`} />}
              {site.lot_redemption_info && <Row label="Lot redemption" value={site.lot_redemption_info} />}
              {site.lot_rental_agreement && <Row label="Lot rental agreement" value={site.lot_rental_agreement} />}
              {site.yard && <Row label="Yard" value={site.yard} />}
              {site.shore && <Row label="Shore" value={site.shore} />}
              {site.zoning && <Row label="Zoning" value={site.zoning} />}
              {site.road_access && <Row label="Road access" value={site.road_access} />}
              {site.water_supply && <Row label="Water supply" value={site.water_supply} />}
              {site.water_supply_types?.length ? <Row label="Water supply types" value={site.water_supply_types.join(', ')} /> : null}
              {site.sewer && <Row label="Sewer" value={site.sewer} />}
              {site.services && <Row label="Services" value={site.services} />}
              {site.transport && <Row label="Transport" value={site.transport} />}
              {site.driving_directions && <Row label="Driving directions" value={site.driving_directions} />}
            </div>
          </Section>
        )}
        {sourceRecords.length > 0 && (
          <Section title="Sources">
            <div className="listing-table">
              {sourceRecords.map(source => (
                <Row
                  key={`${source.provider}-${source.kind}-${source.native_id}`}
                  label={`${providerLabel(source.provider)} ${source.kind}`}
                  value={source.url ? <a className="listing-source-link" href={source.url} target="_blank" rel="noopener noreferrer">{source.native_id || source.external_id || source.url}</a> : source.native_id || source.external_id || source.kind}
                />
              ))}
            </div>
          </Section>
        )}
        {related.length > 0 && (
          <Section title="Listings In Housing Company">
            <div className="building-related-grid">
              {related.map(item => (
                <Link key={item.id} className="building-related-card" to={`/listing/${encodeURIComponent(item.id)}`}>
                  <div className="building-related-title">{item.friendly_id || item.room_layout || item.address || item.id}</div>
                  <div className="building-related-meta">
                    {[item.room_layout, item.area_m2 != null && `${item.area_m2.toFixed(1)} m²`, item.build_year != null && `Built ${item.build_year}`].filter(Boolean).join(' · ')}
                  </div>
                  <div className="building-related-meta">
                    {[item.price_per_m2 != null && `${fmtEur(item.price_per_m2)} / m²`, item.last_seen_at && `Seen ${fmtDate(item.last_seen_at)}`].filter(Boolean).join(' · ')}
                  </div>
                  <div className="building-related-footer">
                    {item.price != null && <span>{fmtPrice(item.price)}</span>}
                    {item.sold_price != null && <span className="building-related-sale">Sold {fmtPrice(item.sold_price)}{item.sold_at ? ` · ${fmtDate(item.sold_at)}` : ''}</span>}
                    <span>{[...(item.providers ?? []), ...(item.kinds ?? [])].join(' · ')}</span>
                  </div>
                </Link>
              ))}
            </div>
          </Section>
        )}
      </div>
    </div>
  )
}

function Section({ title, actions, children }: { title: string; actions?: React.ReactNode; children: React.ReactNode }) {
  return (
    <section className="listing-section">
      <div className="listing-section-header">
        <h2 className="listing-section-title">{title}</h2>
        {actions}
      </div>
      {children}
    </section>
  )
}

function extractSaleListingRenovations(id: string): Promise<RenovationExtractionResponse> {
  return customInstance<RenovationExtractionResponse>(
    `/api/v1/sale-listings/${encodeURIComponent(id)}/renovations/extract?model=${encodeURIComponent(RENOVATION_EXTRACTION_MODEL)}`,
    { method: 'POST' },
  )
}

function extractSaleListingDescription(id: string): Promise<DescriptionExtractionResponse> {
  return customInstance<DescriptionExtractionResponse>(
    `/api/v1/sale-listings/${encodeURIComponent(id)}/description/extract?model=${encodeURIComponent(RENOVATION_EXTRACTION_MODEL)}`,
    { method: 'POST' },
  )
}

function extractSaleListingValuationInputs(id: string): Promise<ValuationInputExtractionResponse> {
  return customInstance<ValuationInputExtractionResponse>(
    `/api/v1/sale-listings/${encodeURIComponent(id)}/valuation-inputs/extract?model=${encodeURIComponent(RENOVATION_EXTRACTION_MODEL)}`,
    { method: 'POST' },
  )
}

function projectSaleListingApartmentProfile(id: string): Promise<ApartmentProfileProjectionResponse> {
  return customInstance<ApartmentProfileProjectionResponse>(
    `/api/v1/sale-listings/${encodeURIComponent(id)}/apartment-profile/project`,
    { method: 'POST' },
  )
}

function generateSaleListingHouseOverview(id: string): Promise<HouseOverviewGenerationResponse> {
  return customInstance<HouseOverviewGenerationResponse>(
    `/api/v1/sale-listings/${encodeURIComponent(id)}/house-overview/generate?model=${encodeURIComponent(RENOVATION_EXTRACTION_MODEL)}`,
    { method: 'POST' },
  )
}

function groupValuationFacts(facts: ValuationFact[]): Array<{ section: string; facts: ValuationFact[] }> {
  const grouped = new Map<string, ValuationFact[]>()
  for (const fact of facts) {
    if (!grouped.has(fact.section)) grouped.set(fact.section, [])
    grouped.get(fact.section)!.push(fact)
  }
  return Array.from(grouped.entries()).map(([section, values]) => ({ section, facts: values }))
}

function valuationInputDisplayFacts(input: ValuationInput | undefined, evidenceFacts: ValuationFact[]): ValuationFact[] {
  if (!input) return evidenceFacts
  const sections: Array<keyof ValuationInput> = ['unit', 'layout', 'floor', 'building', 'site', 'charges', 'market', 'documents']
  const facts: ValuationFact[] = []
  for (const section of sections) {
    const values = input[section]
    if (!values || Array.isArray(values) || typeof values !== 'object') continue
    for (const [key, value] of Object.entries(values)) {
      if (value == null || value === '' || (Array.isArray(value) && value.length === 0)) continue
      if (typeof value === 'boolean') {
        facts.push({ section, key, value_kind: 'bool', value_bool: value })
      } else if (typeof value === 'number') {
        facts.push({ section, key, value_kind: 'number', value_number: value })
      } else if (typeof value === 'string') {
        facts.push({ section, key, value_kind: 'text', value_text: value })
      }
    }
  }
  return facts.length ? facts : evidenceFacts
}

function formatValuationFactValue(fact: ValuationFact): string {
  if (fact.value_kind === 'bool' && fact.value_bool != null) return fact.value_bool ? 'yes' : 'no'
  if (fact.value_kind === 'number' && fact.value_number != null) return String(fact.value_number)
  return fact.value_text || '—'
}

const PROFILE_COMPLETENESS_FIELDS: Array<keyof ApartmentProfile> = [
  'area_m2',
  'room_layout',
  'room_count',
  'bedroom_count',
  'floor_level',
  'total_floors',
  'kitchen_type',
  'layout_quality',
  'condition',
  'kitchen_condition',
  'bathroom_condition',
  'surface_renovation_need',
  'modernization_need',
  'sauna',
  'balcony',
  'balcony_glazing',
  'parking_type',
  'storage_quality',
  'view_quality',
  'noise_risk',
  'accessibility',
]

function profileCompleteness(profile: ApartmentProfile): { completed: number; total: number } {
  return {
    completed: PROFILE_COMPLETENESS_FIELDS.filter(key => hasProfileValue(profile[key])).length,
    total: PROFILE_COMPLETENESS_FIELDS.length,
  }
}

function groupApartmentProfile(profile: ApartmentProfile): ApartmentProfileGroup[] {
  return [
    profileGroup('basics', 'Apartment', [
      profileFact('area_m2', 'Area', profile.area_m2 != null ? `${profile.area_m2.toFixed(1)} m²` : undefined),
      profileFact('living_area_m2', 'Living area', profile.living_area_m2 != null ? `${profile.living_area_m2.toFixed(1)} m²` : undefined),
      profileFact('room_layout', 'Layout', profile.room_layout),
      profileFact('room_count', 'Rooms', profile.room_count),
      profileFact('bedroom_count', 'Bedrooms', profile.bedroom_count),
      profileFact('floor_level', 'Floor', floorLabel(profile.floor_level, profile.total_floors)),
    ]),
    profileGroup('layout', 'Layout Quality', [
      profileFact('kitchen_type', 'Kitchen', profile.kitchen_type),
      profileFact('layout_quality', 'Layout quality', profile.layout_quality, positiveFor(profile.layout_quality, ['good', 'excellent', 'efficient'])),
      profileFact('awkward_layout', 'Awkward layout', profile.awkward_layout, profile.awkward_layout ? 'negative' : 'positive'),
    ]),
    profileGroup('condition', 'Condition', [
      profileFact('condition', 'Overall condition', profile.condition, positiveFor(profile.condition, ['good', 'excellent', 'new'])),
      profileFact('kitchen_condition', 'Kitchen', profile.kitchen_condition, positiveFor(profile.kitchen_condition, ['good', 'excellent', 'renovated', 'new'])),
      profileFact('bathroom_condition', 'Bathroom', profile.bathroom_condition, positiveFor(profile.bathroom_condition, ['good', 'excellent', 'renovated', 'new'])),
      profileFact('surface_renovation_need', 'Surface need', profile.surface_renovation_need, profile.surface_renovation_need ? 'negative' : 'positive'),
      profileFact('modernization_need', 'Modernization need', profile.modernization_need, profile.modernization_need ? 'negative' : 'positive'),
    ]),
    profileGroup('features', 'Features', [
      profileFact('sauna', 'Sauna', profile.sauna, profile.sauna ? 'positive' : undefined),
      profileFact('balcony', 'Balcony', profile.balcony, profile.balcony ? 'positive' : undefined),
      profileFact('balcony_glazing', 'Balcony glazing', profile.balcony_glazing, profile.balcony_glazing ? 'positive' : undefined),
      profileFact('parking_type', 'Parking', profile.parking_type),
      profileFact('storage_quality', 'Storage', profile.storage_quality, positiveFor(profile.storage_quality, ['good', 'excellent'])),
      profileFact('view_quality', 'Views', profile.view_quality, positiveFor(profile.view_quality, ['good', 'excellent', 'open', 'sea'])),
      profileFact('noise_risk', 'Noise risk', profile.noise_risk, profile.noise_risk ? 'negative' : 'positive'),
      profileFact('accessibility', 'Accessibility', profile.accessibility, positiveFor(profile.accessibility, ['good', 'excellent', 'accessible'])),
    ]),
    profileGroup('links', 'Canonical Links', [
      profileFact('housing_company_id', 'Housing company ID', profile.housing_company_id),
      profileFact('property_unit_id', 'Property unit ID', profile.property_unit_id),
      profileFact('confidence', 'Confidence', profile.confidence),
      profileFact('updated_at', 'Updated', profile.updated_at ? fmtDateTime(profile.updated_at) : undefined),
    ]),
  ].filter(group => group.facts.length > 0)
}

function profileGroup(key: string, title: string, facts: Array<ApartmentProfileFact | null>): ApartmentProfileGroup {
  return { key, title, facts: facts.filter((fact): fact is ApartmentProfileFact => fact != null) }
}

function profileFact(key: string, label: string, value: unknown, tone?: ApartmentProfileFact['tone']): ApartmentProfileFact | null {
  if (!hasProfileValue(value)) return null
  return { key, label, value: typeof value === 'boolean' ? (value ? 'yes' : 'no') : String(value), tone }
}

function floorLabel(floor?: number, total?: number): string | undefined {
  if (floor == null) return undefined
  return total != null ? `${floor} / ${total}` : String(floor)
}

function hasProfileValue(value: unknown): boolean {
  return value != null && value !== ''
}

function positiveFor(value: string | undefined, positiveTerms: string[]): ApartmentProfileFact['tone'] | undefined {
  if (!value) return undefined
  const normalized = value.toLowerCase()
  if (positiveTerms.some(term => normalized.includes(term))) return 'positive'
  if (['poor', 'bad', 'weak', 'renovation', 'needs'].some(term => normalized.includes(term))) return 'negative'
  return undefined
}

const KEY_RENOVATION_CATEGORIES = ['pipe', 'sewer', 'water_supply', 'roof', 'facade', 'window', 'balcony', 'elevator', 'heating', 'ventilation', 'electricity', 'drainage']

function keyRenovationGridItems(rows: RenovationItem[], forecasts: RenovationForecastItem[]): KeyRenovationGridItem[] {
  const categories = new Set(KEY_RENOVATION_CATEGORIES)
  for (const row of rows) categories.add(normalizeRenovationCategoryKey(row.kind))
  for (const forecast of forecasts) categories.add(normalizeRenovationCategoryKey(forecast.category))
  return Array.from(categories).map(category => {
    const matchingRows = rows.filter(row => normalizeRenovationCategoryKey(row.kind) === category)
    const done = matchingRows.filter(row => row.status === 'done').sort((a, b) => (b.year ?? 0) - (a.year ?? 0))[0]
    const planned = matchingRows.filter(row => row.status === 'planned').sort((a, b) => (a.year ?? 9999) - (b.year ?? 9999))[0]
    const forecast = forecasts.filter(item => normalizeRenovationCategoryKey(item.category) === category).sort((a, b) => renovationForecastSortYear(a) - renovationForecastSortYear(b))[0]
    return { category, done, planned, forecast }
  }).filter(item => item.done || item.planned || item.forecast || KEY_RENOVATION_CATEGORIES.includes(item.category))
}

function normalizeRenovationCategoryKey(value: string): string {
  const key = value.toLowerCase().replace(/[^a-z0-9åäö]+/g, '_').replace(/^_+|_+$/g, '')
  if (['pipes', 'pipe_renovation', 'plumbing', 'putkiremontti', 'linjasaneeraus'].includes(key)) return 'pipe'
  if (['windows', 'ikkunat'].includes(key)) return 'window'
  if (['balconies', 'parvekkeet'].includes(key)) return 'balcony'
  if (['electric', 'electrical', 'sahko', 'sähkö'].includes(key)) return 'electricity'
  if (['water', 'water_supply', 'vesijohto'].includes(key)) return 'water_supply'
  return key
}

function renovationForecastSortYear(item: RenovationForecastItem): number {
  return item.year ?? item.window_start_year ?? 9999
}

function renovationForecastDate(item?: RenovationForecastItem): string | number | undefined {
  if (!item) return undefined
  if (item.year != null) return item.year
  if (item.year_range) return item.year_range
  if (item.window_start_year != null && item.window_end_year != null) return `${item.window_start_year}-${item.window_end_year}`
  return undefined
}

function renovationGridTone(item: KeyRenovationGridItem): string {
  if (item.planned || ['planned', 'expected', 'follow_up', 'verify_status'].includes(item.forecast?.status || '')) return 'risk'
  if (item.done && !item.forecast) return 'done'
  return 'unknown'
}

function renovationGridStatus(item: KeyRenovationGridItem): string {
  if (item.planned) return 'planned'
  if (item.forecast?.status) return renovationStatusLabel(item.forecast.status)
  if (item.done) return 'done'
  return 'missing'
}

function renovationGridExplanation(item: KeyRenovationGridItem): string {
  if (item.forecast?.explanation) return item.forecast.explanation
  if (item.planned?.kind) return item.planned.kind
  if (item.done?.kind) return item.done.kind
  return ''
}

function RenovationDate({ label, value, detail }: { label: string; value?: string | number; detail?: string }) {
  return (
    <div className="renovation-date-cell">
      <span>{label}</span>
      <strong>{value ?? '—'}</strong>
      {detail && <small>{detail}</small>}
    </div>
  )
}

function HouseOverviewList({ title, items, empty }: { title: string; items?: string[] | null; empty: string }) {
  return (
    <div className="house-overview-list">
      <span>{title}</span>
      {items?.length ? (
        <ul>
          {items.map(item => <li key={item}>{item}</li>)}
        </ul>
      ) : (
        <p>{empty}</p>
      )}
    </div>
  )
}

function ProfileStep({ label, done, detail }: { label: string; done: boolean; detail?: string }) {
  return (
    <div className={`profile-step${done ? ' profile-step--done' : ' profile-step--pending'}`}>
      <span>{done ? 'Ready' : 'Missing'}</span>
      <strong>{label}</strong>
      {detail && <p>{detail}</p>}
    </div>
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

function Row({ label, value, highlight }: { label: string; value: React.ReactNode; highlight?: boolean }) {
  return (
    <div className="listing-row">
      <span className="listing-row-label">{label}</span>
      <span className={`listing-row-value${highlight ? ' listing-row-value--highlight' : ''}`}>
        {value}
      </span>
    </div>
  )
}

function OfferRange({ label, range }: { label: string; range?: ValueRange }) {
  return (
    <div className="offer-range">
      <span>{label}</span>
      <strong>{fmtRange(range)}</strong>
    </div>
  )
}

function BriefPanel({ title, children }: { title: string; children: React.ReactNode }) {
  return (
    <div className="brief-panel">
      <div className="brief-panel-title">{title}</div>
      <div className="brief-panel-body">{children}</div>
    </div>
  )
}

function BriefSignalRow({ signal }: { signal: BriefSignal }) {
  return (
    <div className={`brief-signal brief-signal--${signal.direction}`}>
      <span>{signal.severity}</span>
      <strong>{signal.label}</strong>
      {signal.explanation && <p>{signal.explanation}</p>}
    </div>
  )
}

function TransactionModal({ transaction, onClose }: { transaction: NonNullable<SaleListing['commercial']['matched_transaction']>; onClose: () => void }) {
  return (
    <div className="modal-overlay" onClick={onClose}>
      <div className="transaction-modal" onClick={e => e.stopPropagation()}>
        <div className="transaction-modal-header">
          <div>
            <div className="listing-type-pill listing-type-pill--listing">Transaction</div>
            <h2 className="transaction-modal-title">{transaction.description || transaction.id}</h2>
          </div>
          <button type="button" className="transaction-modal-close" onClick={onClose} aria-label="Close transaction details">×</button>
        </div>
        <div className="listing-table">
          <Row label="ID" value={transaction.id} />
          {transaction.first_seen_at && <Row label="First seen" value={fmtDateTime(transaction.first_seen_at)} highlight />}
          {transaction.updated_at && <Row label="Updated" value={fmtDateTime(transaction.updated_at)} />}
          {transaction.period_identifier && <Row label="Period" value={transaction.period_identifier} />}
          {transaction.price != null && <Row label="Price" value={fmtPrice(transaction.price)} highlight />}
          {transaction.price_per_m2 != null && <Row label="Price / m²" value={fmtEur(transaction.price_per_m2)} />}
          {transaction.area_m2 != null && <Row label="Area" value={`${transaction.area_m2.toFixed(1)} m²`} />}
          {transaction.type && <Row label="Type" value={transaction.type} />}
          {transaction.category && <Row label="Category" value={transaction.category} />}
          {transaction.build_year != null && <Row label="Build year" value={String(transaction.build_year)} />}
          {transaction.floor && <Row label="Floor" value={transaction.floor} />}
          {transaction.elevator != null && <Row label="Elevator" value={fmtBool(transaction.elevator)} />}
          {transaction.condition && <Row label="Condition" value={transaction.condition} />}
          {transaction.plot && <Row label="Plot" value={transaction.plot} />}
          {transaction.plot_owned != null && <Row label="Plot owned" value={fmtBool(transaction.plot_owned)} />}
          {transaction.energy_class && <Row label="Energy" value={transaction.energy_class} />}
          {transaction.city && <Row label="City" value={transaction.city} />}
          {transaction.neighborhood && <Row label="Neighborhood" value={transaction.neighborhood} />}
          {transaction.postal_code && <Row label="Postal code" value={transaction.postal_code} />}
          {transaction.match_status && <Row label="Match status" value={transaction.match_status} />}
          {transaction.match_score != null && <Row label="Match score" value={String(transaction.match_score)} />}
          {transaction.match_confidence && <Row label="Match confidence" value={transaction.match_confidence} />}
        </div>
      </div>
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

function fmtRange(range?: ValueRange): string {
  if (!range?.low && !range?.high) return '—'
  if (range.low != null && range.high != null) return `${fmtPrice(range.low)} – ${fmtPrice(range.high)}`
  if (range.low != null) return `from ${fmtPrice(range.low)}`
  return `to ${fmtPrice(range.high as number)}`
}

function fmtEur(n: number): string {
  return new Intl.NumberFormat('fi-FI', { maximumFractionDigits: 0 }).format(n) + ' €'
}

function fmtBool(value: boolean): string {
  return value ? 'Yes' : 'No'
}

function metadataNumber(metadata: Record<string, unknown> | undefined, key: string): number | null {
  const value = metadata?.[key]
  return typeof value === 'number' ? value : null
}

function metadataStringArray(metadata: Record<string, unknown> | undefined, key: string): string[] {
  const value = metadata?.[key]
  return Array.isArray(value) ? value.filter((item): item is string => typeof item === 'string') : []
}

function fmtDateTime(value: string): string {
  return new Date(value).toLocaleString('fi-FI', { dateStyle: 'medium', timeStyle: 'short' })
}

function fmtDate(value: string): string {
  return new Date(value).toLocaleDateString('fi-FI', { dateStyle: 'medium' })
}

function renovationItems(records: ListingDetail['building']['renovations'], doneText?: string, plannedText?: string): RenovationItem[] {
  const items = [
    ...(records ?? []).map(item => ({ kind: item.kind, status: item.done === false ? 'planned' as const : 'done' as const, year: item.year })),
    ...renovationTextItems(doneText, 'done'),
    ...renovationTextItems(plannedText, 'planned'),
  ]
  const seen = new Set<string>()
  return items.filter(item => {
    const key = `${item.status}:${item.year ?? ''}:${item.kind}`
    if (seen.has(key)) return false
    seen.add(key)
    return item.kind.trim() !== ''
  }).sort((a, b) => (b.year ?? 0) - (a.year ?? 0) || a.status.localeCompare(b.status) || a.kind.localeCompare(b.kind))
}

function renovationTextItems(text: string | undefined, status: RenovationStatus): RenovationItem[] {
  return (text ?? '').split(/\r?\n/).map(line => {
    const value = line.trim()
    const year = value.match(/\b(19|20)\d{2}\b/)?.[0]
    return { kind: year ? value.replace(year, '').trim() || value : value, status, year: year ? Number(year) : undefined }
  }).filter(item => item.kind !== '')
}

function fmtRenovationNeedYear(item: RenovationForecastItem): string {
  if (item.year_range) return item.year_range
  return item.year != null ? String(item.year) : 'TBD'
}

function briefRenovationWhen(item: KeyRenovationStatus): string {
  if (item.year != null) return `${renovationStatusLabel(item.status)} ${item.year}`
  if (item.window_start_year != null && item.window_end_year != null) return `${item.window_start_year}-${item.window_end_year}`
  return renovationStatusLabel(item.status)
}

function renovationStatusLabel(status: string): string {
  return status.replace(/_/g, ' ')
}

function fmtPlotOwnership(value: string): string {
  const key = value.trim().toLowerCase().replace(/[^a-z0-9åäö]+/g, '_').replace(/^_+|_+$/g, '')
  if (['1', 'oma', 'own', 'owned', 'omistus', 'omistettu'].includes(key)) return 'owned'
  if (['2', '3', 'vuokra', 'rent', 'rented', 'rental', 'lease', 'leased', 'vuokralla', 'vuokratontti'].includes(key)) return 'rented'
  return value
}

function plotSummary(site: NonNullable<ListingDetail['site']>): string {
  return Array.from(new Set([site.plot_ownership_type, site.plot_type].filter(Boolean).map(value => fmtPlotOwnership(value as string)))).join(' · ')
}
