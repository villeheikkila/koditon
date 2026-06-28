import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import Nav from '../components/Nav'
import { usePropertyTargetsMap, type PropertyTargetMapMarkersItem, type PropertyTargetsMapParams } from '../api/koditon'
import { buildAddressLookupPath } from '../lib/address-lookup'

type TargetTypeFilter = 'all' | PropertyTargetMapMarkersItem['target_type']

type MapBounds = {
  min_lat: number
  min_lng: number
  max_lat: number
  max_lng: number
}

type MapMarker = PropertyTargetMapMarkersItem & {
  title?: string
  building_count?: number
  source_count?: number
  document_count?: number
}

const MAP_STYLE: maplibregl.StyleSpecification = {
  version: 8,
  sources: {
    osm: {
      type: 'raster',
      tiles: ['https://tile.openstreetmap.org/{z}/{x}/{y}.png'],
      tileSize: 256,
      attribution: 'OpenStreetMap contributors',
    },
  },
  layers: [
    {
      id: 'osm',
      type: 'raster',
      source: 'osm',
      paint: {
        'raster-saturation': -0.35,
        'raster-brightness-min': 0.18,
        'raster-brightness-max': 0.9,
      },
    },
  ],
}

function useDebouncedValue<T>(value: T, delayMs: number) {
  const [debouncedValue, setDebouncedValue] = useState(value)
  useEffect(() => {
    const timeout = window.setTimeout(() => setDebouncedValue(value), delayMs)
    return () => window.clearTimeout(timeout)
  }, [value, delayMs])
  return debouncedValue
}

export default function MapPage() {
  const navigate = useNavigate()
  const mapRef = useRef<HTMLDivElement | null>(null)
  const mapInstanceRef = useRef<maplibregl.Map | null>(null)
  const renderedMarkersRef = useRef<maplibregl.Marker[]>([])
  const [bounds, setBounds] = useState<MapBounds>()
  const [mapLoaded, setMapLoaded] = useState(false)
  const [clusterRevision, setClusterRevision] = useState(0)
  const [selectedTargetKey, setSelectedTargetKey] = useState('')
  const [query, setQuery] = useState('')
  const [targetFilter, setTargetFilter] = useState<TargetTypeFilter>('all')
  const normalizedQuery = query.trim()
  const debouncedQuery = useDebouncedValue(normalizedQuery, 300)
  const hasSearch = debouncedQuery.length > 0
  const viewportParams = useMemo<PropertyTargetsMapParams>(() => {
    return { ...(bounds ?? {}), limit: 500 }
  }, [bounds])
  const searchParams = useMemo<PropertyTargetsMapParams>(() => ({ q: debouncedQuery, limit: 200 }), [debouncedQuery])
  const viewportQuery = usePropertyTargetsMap(viewportParams, { query: { enabled: bounds !== undefined, placeholderData: previous => previous, staleTime: 30_000 } })
  const searchQuery = usePropertyTargetsMap(searchParams, { query: { enabled: hasSearch, placeholderData: previous => previous, staleTime: 30_000 } })
  const viewportBody = viewportQuery.data?.data as { markers?: MapMarker[] | null } | undefined
  const searchBody = searchQuery.data?.data as { markers?: MapMarker[] | null } | undefined
  const viewportMarkers = useMemo(() => viewportBody?.markers ?? [], [viewportBody?.markers])
  const searchMarkers = useMemo(() => searchBody?.markers ?? [], [searchBody?.markers])
  const markers = targetFilter === 'all' ? viewportMarkers : viewportMarkers.filter(marker => marker.target_type === targetFilter)
  const searchResults = targetFilter === 'all' ? searchMarkers : searchMarkers.filter(marker => marker.target_type === targetFilter)
  const stats = useMemo(() => markerStats(viewportMarkers), [viewportMarkers])
  const selected = [...markers, ...searchResults].find(marker => markerKey(marker) === selectedTargetKey) ?? markers[0] ?? searchResults[0]
  useEffect(() => {
    if (!selected) {
      if (selectedTargetKey) setSelectedTargetKey('')
      return
    }
    const key = markerKey(selected)
    if (key !== selectedTargetKey) setSelectedTargetKey(key)
  }, [selected, selectedTargetKey])
  useEffect(() => {
    if (!mapRef.current || mapInstanceRef.current) return
    const map = new maplibregl.Map({
      container: mapRef.current,
      style: MAP_STYLE,
      center: [24.94, 60.17],
      zoom: 11,
      attributionControl: false,
    })
    mapInstanceRef.current = map
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-left')
    map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-left')
    const updateBounds = () => {
      const next = map.getBounds()
      setBounds({ min_lat: next.getSouth(), min_lng: next.getWest(), max_lat: next.getNorth(), max_lng: next.getEast() })
    }
    map.on('load', () => {
      setMapLoaded(true)
      updateBounds()
    })
    const updateClusters = () => setClusterRevision(revision => revision + 1)
    map.on('moveend', updateBounds)
    map.on('moveend', updateClusters)
    map.on('zoomend', updateClusters)
    const resizeObserver = new ResizeObserver(() => map.resize())
    resizeObserver.observe(mapRef.current)
    return () => {
      resizeObserver.disconnect()
      renderedMarkersRef.current.forEach(marker => marker.remove())
      renderedMarkersRef.current = []
      setMapLoaded(false)
      map.remove()
      mapInstanceRef.current = null
    }
  }, [])
  useEffect(() => {
    const map = mapInstanceRef.current
    if (!map || !mapLoaded) return
    renderedMarkersRef.current.forEach(marker => marker.remove())
    renderedMarkersRef.current = clusterMapMarkers(map, markers).map(cluster => {
      const element = document.createElement('button')
      element.type = 'button'
      if (cluster.markers.length > 1) {
        element.className = 'canonical-marker canonical-marker--cluster'
        element.textContent = String(cluster.markers.length)
        element.setAttribute('aria-label', `${cluster.markers.length} property targets`)
        element.addEventListener('click', () => zoomToCluster(map, cluster.markers))
      } else {
        const marker = cluster.markers[0]
        element.className = `canonical-marker canonical-marker--${marker.target_type}${markerKey(marker) === selectedTargetKey ? ' canonical-marker--selected' : ''}`
        element.textContent = markerLabel(marker)
        element.setAttribute('aria-label', `${labelTarget(marker.target_type)}: ${markerTitle(marker)}`)
        element.addEventListener('click', () => setSelectedTargetKey(markerKey(marker)))
      }
      const popup = cluster.markers.length === 1 ? new maplibregl.Popup({ offset: 16 }).setHTML(markerPopup(cluster.markers[0])) : undefined
      const mapMarker = new maplibregl.Marker({ element }).setLngLat(cluster.center)
      if (popup) mapMarker.setPopup(popup)
      return mapMarker.addTo(map)
    })
    return () => {
      renderedMarkersRef.current.forEach(marker => marker.remove())
      renderedMarkersRef.current = []
    }
  }, [markers, selectedTargetKey, mapLoaded, clusterRevision])
  return (
    <main className="model-page">
      <Nav />
      <div className="canonical-map-shell">
        <header className="model-header canonical-map-header">
          <div>
            <h1>Property targets</h1>
            <p>Standalone houses, physical buildings, and housing companies on the same map.</p>
          </div>
          <Link className="model-upload" to="/search?view=grouped&grouping=grouped">Grouped listings</Link>
        </header>
        <section className="canonical-map-overview" aria-label="Map target summary">
          <button type="button" className={targetTabClass(targetFilter, 'all')} onClick={() => setTargetFilter('all')}>
            <span>All targets</span>
            <strong>{stats.total}</strong>
          </button>
          <button type="button" className={targetTabClass(targetFilter, 'house')} onClick={() => setTargetFilter('house')}>
            <span>Houses</span>
            <strong>{stats.house}</strong>
          </button>
          <button type="button" className={targetTabClass(targetFilter, 'building')} onClick={() => setTargetFilter('building')}>
            <span>Buildings</span>
            <strong>{stats.building}</strong>
          </button>
          <button type="button" className={targetTabClass(targetFilter, 'housing_company')} onClick={() => setTargetFilter('housing_company')}>
            <span>Companies</span>
            <strong>{stats.housing_company}</strong>
          </button>
        </section>
        <div className="canonical-map-layout">
          <section className="canonical-map-main">
            <div ref={mapRef} className="canonical-map-canvas" />
            <div className="canonical-map-search-panel">
              <div className="canonical-map-search">
                <input
                  type="search"
                  value={query}
                  onChange={event => setQuery(event.target.value)}
                  placeholder="Search company, building, house, address"
                />
                {query && <button type="button" onClick={() => setQuery('')}>Clear</button>}
              </div>
              {normalizedQuery && (
                <div className="canonical-map-search-results">
                  <div className="canonical-map-search-results-head">
                    <span>{searchQuery.isFetching ? 'Searching' : 'Search results'}</span>
                    <strong>{searchResults.length}</strong>
                  </div>
                  <div className="canonical-map-search-results-list">
                    {searchResults.slice(0, 8).map(marker => (
                      <button key={markerKey(marker)} type="button" onClick={() => focusMarker(marker)}>
                        <span className={`canonical-map-row-type canonical-map-row-type--${marker.target_type}`}>{targetShortLabel(marker.target_type)}</span>
                        <span>{markerTitle(marker)}</span>
                        <small>{formatLocation(marker)}</small>
                      </button>
                    ))}
                    {!searchQuery.isFetching && searchResults.length === 0 && <p>No matches.</p>}
                  </div>
                </div>
              )}
            </div>
            {(viewportQuery.isFetching || searchQuery.isFetching) && <div className="map-loading">Loading targets</div>}
          </section>
          <aside className="canonical-map-sidebar">
            <MapSelection marker={selected} count={markers.length} />
            <div className="canonical-map-list-head">
              <span>{listHeading(targetFilter, '')}</span>
              <strong>{markers.length}</strong>
            </div>
            <div className="canonical-map-list">
              {markers.map(marker => (
                <button key={markerKey(marker)} type="button" className={`canonical-map-row${markerKey(marker) === (selected ? markerKey(selected) : '') ? ' canonical-map-row--selected' : ''}`} onClick={() => focusMarker(marker)} onDoubleClick={() => openMarker(marker)}>
                  <span className={`canonical-map-row-type canonical-map-row-type--${marker.target_type}`}>{targetShortLabel(marker.target_type)}</span>
                  <span>{markerTitle(marker)}</span>
                  <strong>{marker.offering_count}</strong>
                  <small>{formatLocation(marker)}</small>
                </button>
              ))}
              {markers.length === 0 && <p className="model-empty">No targets for this view.</p>}
            </div>
          </aside>
        </div>
      </div>
    </main>
  )

  function selectMarker(marker: PropertyTargetMapMarkersItem) {
    setSelectedTargetKey(markerKey(marker))
  }

  function focusMarker(marker: PropertyTargetMapMarkersItem) {
    selectMarker(marker)
    mapInstanceRef.current?.flyTo({ center: [marker.lng, marker.lat], zoom: targetZoom(marker), essential: true })
  }

  function openMarker(marker: PropertyTargetMapMarkersItem) {
    navigate(targetPath(marker))
  }
}

function MapSelection({ marker, count }: { marker?: MapMarker; count: number }) {
  if (!marker) {
    return (
      <section className="canonical-map-detail">
        <h2>Property targets</h2>
        <p className="model-empty">No canonical targets in this map area.</p>
      </section>
    )
  }
  const lookupPath = buildAddressLookupPath(marker)
  return (
    <section className="canonical-map-detail">
      <div className="canonical-map-detail-head">
        <div>
          <span className={`canonical-map-target-kind canonical-map-target-kind--${marker.target_type}`}>{labelTarget(marker.target_type)}</span>
          <h2>{markerTitle(marker)}</h2>
          <p>{formatLocation(marker)}</p>
        </div>
        <div className="canonical-map-detail-actions">
          <Link to={targetPath(marker)}>Open</Link>
          {lookupPath && <Link to={lookupPath}>Address lookup</Link>}
        </div>
      </div>
      <div className="canonical-map-role">
        <span>Map role</span>
        <strong>{targetRole(marker)}</strong>
        <p>{targetRoleDetail(marker)}</p>
      </div>
      <dl className="canonical-map-facts">
        <div><dt>Shown</dt><dd>{count}</dd></div>
        <div><dt>{marker.target_type === 'house' ? 'Sources' : 'Buildings'}</dt><dd>{marker.target_type === 'house' ? marker.source_count : marker.building_count}</dd></div>
        <div><dt>{marker.target_type === 'house' ? 'Docs' : 'Units'}</dt><dd>{marker.target_type === 'house' ? marker.document_count : marker.unit_count}</dd></div>
        <div><dt>Offerings</dt><dd>{marker.offering_count}</dd></div>
      </dl>
      <div className="canonical-map-offerings">
        {(marker.offerings ?? []).map(offering => (
          <Link key={`${offering.target.type}:${offering.target.id}`} to={`/target/${offering.target.type}/${offering.target.id}`}>
            <span>{offering.headline || offering.room_layout || 'Offering'}</span>
            <small>{formatOffering(offering.room_layout, offering.area_m2, offering.price_eur)}</small>
          </Link>
        ))}
        {marker.offerings?.length === 0 && <p className="model-empty">No linked offerings yet.</p>}
      </div>
    </section>
  )
}

function markerPopup(marker: MapMarker) {
  const title = escapeHTML(markerTitle(marker))
  const location = escapeHTML(formatLocation(marker))
  const type = escapeHTML(labelTarget(marker.target_type))
  const url = targetPath(marker)
  const lookupURL = buildAddressLookupPath(marker)
  const lookupLink = lookupURL ? `<a href="${lookupURL}">Address lookup</a>` : ''
  return `<div class="canonical-popup"><small>${type}</small><strong>${title}</strong><span>${location}</span><a href="${url}">Open target</a>${lookupLink}</div>`
}

function clusterMapMarkers(map: maplibregl.Map, markers: MapMarker[]) {
  const radius = 52
  const clusters: Array<{ x: number; y: number; markers: MapMarker[] }> = []
  for (const marker of markers) {
    const point = map.project([marker.lng, marker.lat])
    let closest: { x: number; y: number; markers: MapMarker[] } | undefined
    let closestDistance = Number.POSITIVE_INFINITY
    for (const cluster of clusters) {
      const centerX = cluster.x / cluster.markers.length
      const centerY = cluster.y / cluster.markers.length
      const distance = Math.hypot(point.x - centerX, point.y - centerY)
      if (distance < closestDistance) {
        closest = cluster
        closestDistance = distance
      }
    }
    if (closest && closestDistance <= radius) {
      closest.x += point.x
      closest.y += point.y
      closest.markers.push(marker)
      continue
    }
    clusters.push({ x: point.x, y: point.y, markers: [marker] })
  }
  return clusters.map(cell => {
    const center = map.unproject([cell.x / cell.markers.length, cell.y / cell.markers.length])
    return { center: [center.lng, center.lat] as [number, number], markers: cell.markers }
  })
}

function zoomToCluster(map: maplibregl.Map, markers: MapMarker[]) {
  if (markers.length === 0) return
  if (markers.length === 1) {
    map.easeTo({ center: [markers[0].lng, markers[0].lat], zoom: Math.max(map.getZoom(), 15) })
    return
  }
  const bounds = new maplibregl.LngLatBounds()
  markers.forEach(marker => bounds.extend([marker.lng, marker.lat]))
  map.fitBounds(bounds, { padding: 90, maxZoom: Math.max(map.getZoom() + 2, 15) })
}

function targetZoom(marker: PropertyTargetMapMarkersItem) {
  if (marker.target_type === 'housing_company') return 15
  return 16
}

function markerKey(marker: MapMarker) {
  return `${marker.target.type}:${marker.target.id}`
}

function markerTitle(marker: MapMarker) {
  return marker.title || marker.name || marker.address || labelTarget(marker.target.type)
}

function markerLabel(marker: MapMarker) {
  if (marker.target_type === 'house') return 'H'
  if (marker.target_type === 'building') return 'B'
  return 'C'
}

function targetPath(marker: MapMarker) {
  return `/target/${encodeURIComponent(marker.target.type)}/${encodeURIComponent(marker.target.id)}`
}

function labelTarget(type: string) {
  if (type === 'house') return 'House'
  if (type === 'building') return 'Building'
  return 'Housing company'
}

function targetShortLabel(type: string) {
  if (type === 'house') return 'H'
  if (type === 'building') return 'B'
  return 'C'
}

function targetRole(marker: MapMarker) {
  if (marker.target_type === 'house') return 'Standalone house'
  if (marker.target_type === 'building') return 'Physical building'
  return 'Housing company group'
}

function targetRoleDetail(marker: MapMarker) {
  if (marker.target_type === 'house') return 'Listings are shown directly at the house instead of being grouped into a company.'
  if (marker.target_type === 'building') return 'Listings resolve to this building when the company grouping is not the best map surface.'
  return `${marker.building_count || 1} building${marker.building_count === 1 ? '' : 's'} grouped under one company target.`
}

function markerStats(markers: MapMarker[]) {
  return markers.reduce((acc, marker) => {
    acc.total += 1
    acc[marker.target_type] += 1
    return acc
  }, { total: 0, house: 0, building: 0, housing_company: 0 })
}

function targetTabClass(active: TargetTypeFilter, value: TargetTypeFilter) {
  return `canonical-map-tab${active === value ? ' canonical-map-tab--active' : ''}`
}

function listHeading(filter: TargetTypeFilter, hasQuery: string) {
  if (hasQuery) return 'Search results'
  if (filter === 'house') return 'Houses'
  if (filter === 'building') return 'Buildings'
  if (filter === 'housing_company') return 'Housing companies'
  return 'Visible targets'
}

function formatLocation(marker: MapMarker) {
  return [marker.address, marker.postal, marker.city].filter(Boolean).join(' ')
}

function formatOffering(layout?: string, area?: number, price?: number) {
  return [layout, area ? `${area.toLocaleString('fi-FI')} m2` : '', price ? `${price.toLocaleString('fi-FI')} EUR` : ''].filter(Boolean).join(' | ')
}

function escapeHTML(value: string) {
  return value.replaceAll('&', '&amp;').replaceAll('<', '&lt;').replaceAll('>', '&gt;').replaceAll('"', '&quot;').replaceAll("'", '&#39;')
}
