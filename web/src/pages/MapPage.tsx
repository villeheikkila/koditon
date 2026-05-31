import { useEffect, useMemo, useRef, useState } from 'react'
import { Link, useNavigate } from 'react-router-dom'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'
import Nav from '../components/Nav'
import { usePropertyTargetsMap, type PropertyTargetMapMarkersItem, type PropertyTargetsMapParams } from '../api/koditon'

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

export default function MapPage() {
  const navigate = useNavigate()
  const mapRef = useRef<HTMLDivElement | null>(null)
  const mapInstanceRef = useRef<maplibregl.Map | null>(null)
  const renderedMarkersRef = useRef<maplibregl.Marker[]>([])
  const [bounds, setBounds] = useState<MapBounds>()
  const [selectedTargetKey, setSelectedTargetKey] = useState('')
  const [query, setQuery] = useState('')
  const normalizedQuery = query.trim()
  const params = useMemo<PropertyTargetsMapParams>(() => {
    if (normalizedQuery) return { q: normalizedQuery, limit: 200 }
    return { ...(bounds ?? {}), limit: 500 }
  }, [bounds, normalizedQuery])
  const mapQuery = usePropertyTargetsMap(params, { query: { staleTime: 30_000 } })
  const body = mapQuery.data?.data as { markers?: MapMarker[] | null } | undefined
  const markers = body?.markers ?? []
  const selected = markers.find(marker => markerKey(marker) === selectedTargetKey) ?? markers[0]
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
    map.on('load', updateBounds)
    map.on('moveend', updateBounds)
    return () => {
      renderedMarkersRef.current.forEach(marker => marker.remove())
      renderedMarkersRef.current = []
      map.remove()
      mapInstanceRef.current = null
    }
  }, [])
  useEffect(() => {
    const map = mapInstanceRef.current
    if (!map) return
    renderedMarkersRef.current.forEach(marker => marker.remove())
    renderedMarkersRef.current = markers.map(marker => {
      const element = document.createElement('button')
      element.type = 'button'
      element.className = `canonical-marker${markerKey(marker) === selectedTargetKey ? ' canonical-marker--selected' : ''}`
      element.textContent = markerLabel(marker)
      element.addEventListener('click', () => setSelectedTargetKey(markerKey(marker)))
      const popup = new maplibregl.Popup({ offset: 16 }).setHTML(markerPopup(marker))
      return new maplibregl.Marker({ element }).setLngLat([marker.lng, marker.lat]).setPopup(popup).addTo(map)
    })
    return () => {
      renderedMarkersRef.current.forEach(marker => marker.remove())
      renderedMarkersRef.current = []
    }
  }, [markers, selectedTargetKey])
  useEffect(() => {
    if (!normalizedQuery || markers.length === 0) return
    const map = mapInstanceRef.current
    if (!map) return
    if (markers.length === 1) {
      map.flyTo({ center: [markers[0].lng, markers[0].lat], zoom: 15 })
      setSelectedTargetKey(markerKey(markers[0]))
      return
    }
    const bounds = new maplibregl.LngLatBounds()
    markers.forEach(marker => bounds.extend([marker.lng, marker.lat]))
    map.fitBounds(bounds, { padding: { top: 80, right: 430, bottom: 80, left: 80 }, maxZoom: 15 })
    setSelectedTargetKey(current => current && markers.some(marker => markerKey(marker) === current) ? current : markerKey(markers[0]))
  }, [markers, normalizedQuery])
  return (
    <main className="model-page">
      <Nav />
      <div className="canonical-map-shell">
        <header className="model-header canonical-map-header">
          <div>
            <h1>Map</h1>
            <p>Canonical houses, buildings, and housing companies with linked offerings.</p>
          </div>
          <Link className="model-upload" to="/search">Targets</Link>
        </header>
        <div className="canonical-map-layout">
          <section className="canonical-map-main">
            <div ref={mapRef} className="canonical-map-canvas" />
            <div className="canonical-map-search">
              <input
                type="search"
                value={query}
                onChange={event => setQuery(event.target.value)}
                placeholder="Search target, address, city, postal"
              />
              {query && <button type="button" onClick={() => setQuery('')}>Clear</button>}
            </div>
            {mapQuery.isFetching && <div className="map-loading">Loading targets</div>}
          </section>
          <aside className="canonical-map-sidebar">
            <MapSelection marker={selected} count={markers.length} />
            <div className="canonical-map-list-head">
              <span>{normalizedQuery ? 'Search results' : 'Visible targets'}</span>
              <strong>{markers.length}</strong>
            </div>
            <div className="canonical-map-list">
              {markers.map(marker => (
                <button key={markerKey(marker)} type="button" className={`canonical-map-row${markerKey(marker) === (selected ? markerKey(selected) : '') ? ' canonical-map-row--selected' : ''}`} onClick={() => selectMarker(marker)} onDoubleClick={() => openMarker(marker)}>
                  <span>{markerTitle(marker)}</span>
                  <strong>{marker.offering_count}</strong>
                  <small>{formatLocation(marker)}</small>
                </button>
              ))}
            </div>
          </aside>
        </div>
      </div>
    </main>
  )

  function selectMarker(marker: PropertyTargetMapMarkersItem) {
    setSelectedTargetKey(markerKey(marker))
    mapInstanceRef.current?.flyTo({ center: [marker.lng, marker.lat], zoom: Math.max(mapInstanceRef.current.getZoom(), 14) })
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
  return (
    <section className="canonical-map-detail">
      <div className="canonical-map-detail-head">
        <div>
          <h2>{markerTitle(marker)}</h2>
          <p>{formatLocation(marker)}</p>
        </div>
        <Link to={targetPath(marker)}>Open</Link>
      </div>
      <dl className="canonical-map-facts">
        <div><dt>Visible</dt><dd>{count}</dd></div>
        <div><dt>Buildings</dt><dd>{marker.building_count}</dd></div>
        <div><dt>Units</dt><dd>{marker.unit_count}</dd></div>
        <div><dt>Offerings</dt><dd>{marker.offering_count}</dd></div>
      </dl>
      <div className="canonical-map-offerings">
        {(marker.offerings ?? []).map(offering => (
          <Link key={offering.target.id} to={`/target/${offering.target.type}/${offering.target.id}`}>
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
  const url = targetPath(marker)
  return `<div class="canonical-popup"><strong>${title}</strong><span>${location}</span><a href="${url}">Open target</a></div>`
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
  return String(Math.max(marker.offering_count, marker.unit_count, 1))
}

function targetPath(marker: MapMarker) {
  return `/target/${encodeURIComponent(marker.target.type)}/${encodeURIComponent(marker.target.id)}`
}

function labelTarget(type: string) {
  if (type === 'house') return 'House'
  if (type === 'building') return 'Building'
  return 'Housing company'
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
