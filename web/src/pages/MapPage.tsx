import { useEffect, useMemo, useRef, useState } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
import maplibregl, { type GeoJSONSource, type MapLayerMouseEvent } from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import Nav from '../components/Nav'
import { customInstance } from '../lib/axios-instance'

type MapMarker = {
  lat: number
  lng: number
  count: number
  address?: string
  city?: string
  postal?: string
  min_price?: number
  max_price?: number
  min_area_m2?: number
  max_area_m2?: number
  providers?: string[]
  kinds?: string[]
  listing_ids?: string[]
  listings?: MapListing[]
}

type MapListing = {
  id: string
  headline?: string
  address?: string
  city?: string
  postal?: string
  layout?: string
  area_m2?: number
  price?: number
  price_per_m2?: number
  build_year?: number
  last_seen_at?: string
  providers?: string[]
  kinds?: string[]
}

type MapResponse = {
  data: { markers: MapMarker[] }
}

type MapBounds = {
  min_lat: number
  min_lng: number
  max_lat: number
  max_lng: number
}

type MapFilters = {
  q: string
  city: string
  postal: string
  min_price: string
  max_price: string
  min_area: string
  max_area: string
  min_price_m2: string
  max_price_m2: string
  rooms: string
  min_build_year: string
  max_build_year: string
  property_type: string
  condition: string
  energy_class: string
  elevator: string
  sauna: string
  balcony: string
  plot_owned: string
  new_development: string
  has_transaction: string
}

const KIND_OPTIONS = [
  { value: 'ad', label: 'Full ads' },
  { value: '', label: 'All kinds' },
  { value: 'announcement', label: 'Announcements' },
]

const SOURCE_OPTIONS = [
  { value: '', label: 'All sources' },
  { value: 'frontdoor', label: 'Frontdoor' },
  { value: 'shortcut', label: 'Shortcut' },
]

const EMPTY_FILTERS: MapFilters = {
  q: '',
  city: '',
  postal: '',
  min_price: '',
  max_price: '',
  min_area: '',
  max_area: '',
  min_price_m2: '',
  max_price_m2: '',
  rooms: '',
  min_build_year: '',
  max_build_year: '',
  property_type: '',
  condition: '',
  energy_class: '',
  elevator: '',
  sauna: '',
  balcony: '',
  plot_owned: '',
  new_development: '',
  has_transaction: '',
}

const PROPERTY_TYPE_OPTIONS = [
  { value: '', label: 'Any type' },
  { value: 'apartment_house', label: 'Apartment house' },
  { value: 'row_house', label: 'Row house' },
  { value: 'detached_house', label: 'Detached house' },
]

const CONDITION_OPTIONS = [
  { value: '', label: 'Any condition' },
  { value: 'good', label: 'Good' },
  { value: 'satisfactory', label: 'Satisfactory' },
  { value: 'poor', label: 'Poor' },
]

const BOOLEAN_OPTIONS = [
  { value: '', label: 'Any' },
  { value: 'true', label: 'Yes' },
  { value: 'false', label: 'No' },
]

const INITIAL_BOUNDS = {
  min_lat: 60.12,
  min_lng: 24.84,
  max_lat: 60.22,
  max_lng: 25.04,
}

const MAP_STYLE: maplibregl.StyleSpecification = {
  version: 8,
  glyphs: 'https://demotiles.maplibre.org/font/{fontstack}/{range}.pbf',
  sources: {
    osm: {
      type: 'raster',
      tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
      tileSize: 256,
      attribution: '© OpenStreetMap contributors',
    },
  },
  layers: [
    {
      id: 'osm',
      type: 'raster',
      source: 'osm',
      paint: {
        'raster-saturation': -0.25,
        'raster-brightness-min': 0.1,
        'raster-brightness-max': 0.82,
      },
    },
  ],
}

function fmtPrice(value: number) {
  return new Intl.NumberFormat('fi-FI').format(value) + ' €'
}

function fmtDate(value?: string) {
  if (!value) return ''
  return new Intl.DateTimeFormat('fi-FI', { dateStyle: 'medium' }).format(new Date(value))
}

function boundsFromMap(map: maplibregl.Map): MapBounds {
  const bounds = map.getBounds()
  return {
    min_lat: bounds.getSouth(),
    min_lng: bounds.getWest(),
    max_lat: bounds.getNorth(),
    max_lng: bounds.getEast(),
  }
}

function markerKey(marker: MapMarker) {
  return `${marker.lat}:${marker.lng}:${marker.count}`
}

function markerFeatureCollection(markers: MapMarker[]) {
  return {
    type: 'FeatureCollection' as const,
    features: markers.map(marker => ({
      type: 'Feature' as const,
      geometry: {
        type: 'Point' as const,
        coordinates: [marker.lng, marker.lat],
      },
      properties: {
        key: markerKey(marker),
        count: marker.count,
        label: String(marker.count),
      },
    })),
  }
}

function addMarkerLayers(map: maplibregl.Map) {
  if (map.getSource('listing-markers')) return
  map.addSource('listing-markers', {
    type: 'geojson',
    data: markerFeatureCollection([]),
    cluster: true,
    clusterRadius: 54,
    clusterMaxZoom: 16,
  })
  map.addLayer({
    id: 'listing-cluster-halos',
    type: 'circle',
    source: 'listing-markers',
    filter: ['has', 'point_count'],
    paint: {
      'circle-radius': ['step', ['get', 'point_count'], 28, 8, 36, 24, 46, 80, 58],
      'circle-color': '#d5793f',
      'circle-opacity': 0.18,
      'circle-blur': 0.55,
    },
  })
  map.addLayer({
    id: 'listing-cluster-circles',
    type: 'circle',
    source: 'listing-markers',
    filter: ['has', 'point_count'],
    paint: {
      'circle-radius': ['step', ['get', 'point_count'], 18, 8, 24, 24, 31, 80, 38],
      'circle-color': ['step', ['get', 'point_count'], '#6d3a1f', 8, '#7f4927', 24, '#955b31', 80, '#b76a38'],
      'circle-opacity': 0.96,
      'circle-stroke-color': '#d9915a',
      'circle-stroke-width': 2,
    },
  })
  map.addLayer({
    id: 'listing-cluster-labels',
    type: 'symbol',
    source: 'listing-markers',
    filter: ['has', 'point_count'],
    layout: {
      'text-field': ['get', 'point_count_abbreviated'],
      'text-font': ['Open Sans Regular'],
      'text-size': 14,
      'text-allow-overlap': true,
    },
    paint: {
      'text-color': '#f4efe9',
      'text-halo-color': '#422312',
      'text-halo-width': 1,
    },
  })
  map.addLayer({
    id: 'listing-marker-halos',
    type: 'circle',
    source: 'listing-markers',
    filter: ['!', ['has', 'point_count']],
    paint: {
      'circle-radius': ['interpolate', ['linear'], ['get', 'count'], 1, 18, 10, 24, 50, 34],
      'circle-color': '#d5793f',
      'circle-opacity': 0.14,
      'circle-blur': 0.45,
    },
  })
  map.addLayer({
    id: 'listing-marker-circles',
    type: 'circle',
    source: 'listing-markers',
    filter: ['!', ['has', 'point_count']],
    paint: {
      'circle-radius': ['interpolate', ['linear'], ['get', 'count'], 1, 12, 10, 17, 50, 24],
      'circle-color': '#422312',
      'circle-opacity': 0.96,
      'circle-stroke-color': '#b96f3c',
      'circle-stroke-width': 2,
    },
  })
  map.addLayer({
    id: 'listing-marker-labels',
    type: 'symbol',
    source: 'listing-markers',
    filter: ['!', ['has', 'point_count']],
    layout: {
      'text-field': ['get', 'label'],
      'text-font': ['Open Sans Regular'],
      'text-size': 13,
      'text-allow-overlap': true,
    },
    paint: {
      'text-color': '#f4efe9',
      'text-halo-color': '#422312',
      'text-halo-width': 1,
    },
  })
}

function RangeFields({ label, min, max, minPlaceholder, maxPlaceholder, onMin, onMax }: { label: string; min: string; max: string; minPlaceholder: string; maxPlaceholder: string; onMin: (value: string) => void; onMax: (value: string) => void }) {
  return (
    <div className="map-filter-range">
      <div className="map-filter-label">{label}</div>
      <div className="map-filter-range-inputs">
        <input value={min} onChange={event => onMin(event.target.value)} placeholder={minPlaceholder} inputMode="numeric" />
        <span>-</span>
        <input value={max} onChange={event => onMax(event.target.value)} placeholder={maxPlaceholder} inputMode="numeric" />
      </div>
    </div>
  )
}

function BooleanField({ label, value, onChange }: { label: string; value: string; onChange: (value: string) => void }) {
  return (
    <label className="map-filter-field">
      <span>{label}</span>
      <select value={value} onChange={event => onChange(event.target.value)}>
        {BOOLEAN_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
      </select>
    </label>
  )
}

export default function MapPage() {
  const mapRef = useRef<HTMLDivElement | null>(null)
  const mapInstanceRef = useRef<maplibregl.Map | null>(null)
  const markersRef = useRef<MapMarker[]>([])
  const [bounds, setBounds] = useState<MapBounds>(INITIAL_BOUNDS)
  const [source, setSource] = useState('')
  const [kind, setKind] = useState('ad')
  const [filtersOpen, setFiltersOpen] = useState(false)
  const [filters, setFilters] = useState<MapFilters>(EMPTY_FILTERS)
  const [selected, setSelected] = useState<MapMarker | null>(null)
  const activeFilterCount = Object.values(filters).filter(Boolean).length
  const query = useQuery({
    queryKey: ['sale-listing-map', bounds, source, kind, filters],
    queryFn: () => {
      const params = new URLSearchParams({
        min_lat: String(bounds.min_lat),
        min_lng: String(bounds.min_lng),
        max_lat: String(bounds.max_lat),
        max_lng: String(bounds.max_lng),
        limit: '700',
      })
      if (source) params.set('source', source)
      if (kind) params.set('kind', kind)
      Object.entries(filters).forEach(([key, value]) => {
        if (value) params.set(key, value)
      })
      return customInstance<MapResponse>(`/api/v1/sale-listings/map?${params.toString()}`)
    },
    staleTime: 30_000,
  })
  const markers = query.data?.data.markers ?? []
  const markerGeoJSON = useMemo(() => markerFeatureCollection(markers), [markers])
  useEffect(() => {
    markersRef.current = markers
  }, [markers])
  useEffect(() => {
    if (!mapRef.current || mapInstanceRef.current) return
    const map = new maplibregl.Map({
      container: mapRef.current,
      style: MAP_STYLE,
      center: [24.9384, 60.1699],
      zoom: 13,
      attributionControl: false,
    })
    mapInstanceRef.current = map
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-left')
    map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-left')
    const updateBounds = () => setBounds(boundsFromMap(map))
    const selectMarker = (event: MapLayerMouseEvent) => {
      const key = event.features?.[0]?.properties?.key
      const marker = markersRef.current.find(value => markerKey(value) === key)
      if (marker) setSelected(marker)
    }
    const zoomToCluster = (event: MapLayerMouseEvent) => {
      const feature = event.features?.[0]
      const clusterID = feature?.properties?.cluster_id
      const coordinates = feature?.geometry.type === 'Point' ? feature.geometry.coordinates : null
      const markerSource = map.getSource('listing-markers') as GeoJSONSource | undefined
      if (clusterID == null || !coordinates || !markerSource) return
      markerSource.getClusterExpansionZoom(clusterID).then(zoom => {
        map.easeTo({
          center: coordinates as [number, number],
          zoom,
          duration: 520,
          easing: value => 1 - Math.pow(1 - value, 3),
        })
      })
      setSelected(null)
    }
    map.on('load', () => {
      addMarkerLayers(map)
      updateBounds()
      map.on('click', 'listing-cluster-circles', zoomToCluster)
      map.on('click', 'listing-cluster-labels', zoomToCluster)
      map.on('click', 'listing-marker-circles', selectMarker)
      map.on('click', 'listing-marker-labels', selectMarker)
      map.on('mouseenter', 'listing-cluster-circles', () => {
        map.getCanvas().style.cursor = 'pointer'
      })
      map.on('mouseenter', 'listing-cluster-labels', () => {
        map.getCanvas().style.cursor = 'pointer'
      })
      map.on('mouseenter', 'listing-marker-circles', () => {
        map.getCanvas().style.cursor = 'pointer'
      })
      map.on('mouseenter', 'listing-marker-labels', () => {
        map.getCanvas().style.cursor = 'pointer'
      })
      map.on('mouseleave', 'listing-marker-circles', () => {
        map.getCanvas().style.cursor = ''
      })
      map.on('mouseleave', 'listing-marker-labels', () => {
        map.getCanvas().style.cursor = ''
      })
      map.on('mouseleave', 'listing-cluster-circles', () => {
        map.getCanvas().style.cursor = ''
      })
      map.on('mouseleave', 'listing-cluster-labels', () => {
        map.getCanvas().style.cursor = ''
      })
    })
    map.on('moveend', updateBounds)
    return () => {
      map.remove()
      mapInstanceRef.current = null
    }
  }, [])
  useEffect(() => {
    const map = mapInstanceRef.current
    if (!map) return
    const update = () => {
      addMarkerLayers(map)
      const markerSource = map.getSource('listing-markers') as GeoJSONSource | undefined
      markerSource?.setData(markerGeoJSON)
    }
    if (map.isStyleLoaded()) update()
    else map.once('load', update)
  }, [markerGeoJSON])
  function useCurrentLocation() {
    navigator.geolocation?.getCurrentPosition(position => {
      mapInstanceRef.current?.flyTo({ center: [position.coords.longitude, position.coords.latitude], zoom: 15 })
      setSelected(null)
    })
  }
  function setFilter<K extends keyof MapFilters>(key: K, value: MapFilters[K]) {
    setFilters(current => ({ ...current, [key]: value }))
    setSelected(null)
  }
  function clearFilters() {
    setFilters(EMPTY_FILTERS)
    setSelected(null)
  }
  return (
    <div className="map-layout">
      <Nav actions={<span className="search-total">{markers.length.toLocaleString('fi-FI')} locations</span>} />
      <div className="map-toolbar">
        <select className="search-select" value={kind} onChange={event => { setKind(event.target.value); setSelected(null) }}>
          {KIND_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
        </select>
        <select className="search-select" value={source} onChange={event => { setSource(event.target.value); setSelected(null) }}>
          {SOURCE_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
        </select>
        <button className="search-clear-btn" type="button" onClick={() => setFiltersOpen(value => !value)}>
          Filters{activeFilterCount > 0 ? ` (${activeFilterCount})` : ''}
        </button>
        <button className="search-clear-btn" type="button" onClick={useCurrentLocation}>Current location</button>
      </div>
      {filtersOpen && (
        <div className="map-filter-panel">
          <div className="map-filter-head">
            <div>
              <div className="map-filter-title">Map filters</div>
              <div className="map-filter-subtitle">Uses normalized canonical listing facts.</div>
            </div>
            <button className="search-clear-btn" type="button" onClick={clearFilters}>Clear</button>
          </div>
          <div className="map-filter-grid">
            <label className="map-filter-field map-filter-field--wide">
              <span>Search</span>
              <input value={filters.q} onChange={event => setFilter('q', event.target.value)} placeholder="Address, postal, description" />
            </label>
            <label className="map-filter-field">
              <span>City</span>
              <input value={filters.city} onChange={event => setFilter('city', event.target.value)} placeholder="Helsinki" />
            </label>
            <label className="map-filter-field">
              <span>Postal</span>
              <input value={filters.postal} onChange={event => setFilter('postal', event.target.value)} placeholder="00100" />
            </label>
            <RangeFields label="Price" min={filters.min_price} max={filters.max_price} minPlaceholder="Min €" maxPlaceholder="Max €" onMin={value => setFilter('min_price', value)} onMax={value => setFilter('max_price', value)} />
            <RangeFields label="Area" min={filters.min_area} max={filters.max_area} minPlaceholder="Min m²" maxPlaceholder="Max m²" onMin={value => setFilter('min_area', value)} onMax={value => setFilter('max_area', value)} />
            <RangeFields label="€/m²" min={filters.min_price_m2} max={filters.max_price_m2} minPlaceholder="Min" maxPlaceholder="Max" onMin={value => setFilter('min_price_m2', value)} onMax={value => setFilter('max_price_m2', value)} />
            <RangeFields label="Build year" min={filters.min_build_year} max={filters.max_build_year} minPlaceholder="From" maxPlaceholder="To" onMin={value => setFilter('min_build_year', value)} onMax={value => setFilter('max_build_year', value)} />
            <label className="map-filter-field">
              <span>Rooms</span>
              <select value={filters.rooms} onChange={event => setFilter('rooms', event.target.value)}>
                <option value="">Any rooms</option>
                {[1, 2, 3, 4, 5, 6, 7].map(value => <option key={value} value={String(value)}>{value === 7 ? '7+' : value} rooms</option>)}
              </select>
            </label>
            <label className="map-filter-field">
              <span>Property type</span>
              <select value={filters.property_type} onChange={event => setFilter('property_type', event.target.value)}>
                {PROPERTY_TYPE_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
            <label className="map-filter-field">
              <span>Condition</span>
              <select value={filters.condition} onChange={event => setFilter('condition', event.target.value)}>
                {CONDITION_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
              </select>
            </label>
            <label className="map-filter-field">
              <span>Energy</span>
              <input value={filters.energy_class} onChange={event => setFilter('energy_class', event.target.value)} placeholder="A2018, B, E13_G" />
            </label>
            <BooleanField label="Elevator" value={filters.elevator} onChange={value => setFilter('elevator', value)} />
            <BooleanField label="Sauna" value={filters.sauna} onChange={value => setFilter('sauna', value)} />
            <BooleanField label="Balcony" value={filters.balcony} onChange={value => setFilter('balcony', value)} />
            <BooleanField label="Owned plot" value={filters.plot_owned} onChange={value => setFilter('plot_owned', value)} />
            <BooleanField label="New development" value={filters.new_development} onChange={value => setFilter('new_development', value)} />
            <BooleanField label="Linked transaction" value={filters.has_transaction} onChange={value => setFilter('has_transaction', value)} />
          </div>
        </div>
      )}
      <div className="map-shell">
        <div ref={mapRef} className="map-canvas">
          {query.isPending && <div className="map-loading">Loading locations…</div>}
        </div>
        <aside className="map-panel">
          {selected ? (
            <>
              <div className="map-panel-title">{selected.address || 'Location'}</div>
              <div className="map-panel-sub">{[selected.postal, selected.city].filter(Boolean).join(' ')}</div>
              <div className="map-panel-facts">
                <span>{selected.count} listings</span>
                {selected.min_price != null && selected.max_price != null && (
                  <span>{selected.min_price === selected.max_price ? fmtPrice(selected.min_price) : `${fmtPrice(selected.min_price)} - ${fmtPrice(selected.max_price)}`}</span>
                )}
                {selected.min_area_m2 != null && selected.max_area_m2 != null && (
                  <span>{selected.min_area_m2.toFixed(0)}-{selected.max_area_m2.toFixed(0)} m²</span>
                )}
              </div>
              <div className="search-card-badges">
                {selected.providers?.map(provider => <span key={provider} className={`search-badge search-badge--${provider}`}>{provider}</span>)}
                {selected.kinds?.map(value => <span key={value} className="search-badge search-badge--kind">{value}</span>)}
              </div>
              <div className="map-listing-list">
                {selected.listings?.map(listing => (
                  <Link key={listing.id} className="map-listing-card" to={`/listing/${encodeURIComponent(listing.id)}`}>
                    <div className="map-listing-head">
                      <div>
                        <div className="map-listing-title">{listing.address || listing.headline || 'Listing'}</div>
                        <div className="map-listing-sub">{[listing.postal, listing.city].filter(Boolean).join(' ')}</div>
                      </div>
                      {listing.price != null && <div className="map-listing-price">{fmtPrice(listing.price)}</div>}
                    </div>
                    <div className="map-listing-meta">
                      {listing.layout && <span>{listing.layout}</span>}
                      {listing.area_m2 != null && <span>{listing.area_m2.toFixed(1)} m²</span>}
                      {listing.price_per_m2 != null && <span>{Math.round(listing.price_per_m2).toLocaleString('fi-FI')} €/m²</span>}
                      {listing.build_year != null && <span>Built {listing.build_year}</span>}
                      {listing.last_seen_at && <span>Seen {fmtDate(listing.last_seen_at)}</span>}
                    </div>
                    <div className="search-card-badges">
                      {listing.providers?.map(provider => <span key={provider} className={`search-badge search-badge--${provider}`}>{provider}</span>)}
                      {listing.kinds?.map(value => <span key={value} className="search-badge search-badge--kind">{value}</span>)}
                    </div>
                  </Link>
                ))}
                {selected.listings && selected.count > selected.listings.length && (
                  <div className="map-listing-more">Showing {selected.listings.length} of {selected.count} ads at this location</div>
                )}
              </div>
            </>
          ) : (
            <div className="map-empty">
              <div className="map-panel-title">Select a location</div>
              <div className="map-panel-sub">Markers are grouped by exact canonical building point.</div>
            </div>
          )}
        </aside>
      </div>
    </div>
  )
}
