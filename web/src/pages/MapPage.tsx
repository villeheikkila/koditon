import { useEffect, useMemo, useRef, useState, type WheelEvent } from 'react'
import { useQuery } from '@tanstack/react-query'
import { Link } from 'react-router-dom'
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

const TILE_SIZE = 256

function fmtPrice(value: number) {
  return new Intl.NumberFormat('fi-FI').format(value) + ' €'
}

function fmtDate(value?: string) {
  if (!value) return ''
  return new Intl.DateTimeFormat('fi-FI', { dateStyle: 'medium' }).format(new Date(value))
}

function markerSize(count: number) {
  return Math.max(28, Math.min(58, 24 + Math.log2(count + 1) * 8))
}

function project(lat: number, lng: number, zoom: number) {
  const scale = 2 ** zoom
  const sin = Math.sin((lat * Math.PI) / 180)
  return {
    x: ((lng + 180) / 360) * scale,
    y: (0.5 - Math.log((1 + sin) / (1 - sin)) / (4 * Math.PI)) * scale,
  }
}

function unproject(x: number, y: number, zoom: number) {
  const scale = 2 ** zoom
  const lng = (x / scale) * 360 - 180
  const lat = (Math.atan(Math.sinh(Math.PI * (1 - (2 * y) / scale))) * 180) / Math.PI
  return { lat, lng }
}

function viewportProjection(center: { lat: number; lng: number }, zoom: number, viewport: { width: number; height: number }) {
  const centerPoint = project(center.lat, center.lng, zoom)
  const halfWidth = viewport.width / TILE_SIZE / 2
  const halfHeight = viewport.height / TILE_SIZE / 2
  return {
    zoom,
    topLeft: { x: centerPoint.x - halfWidth, y: centerPoint.y - halfHeight },
    bottomRight: { x: centerPoint.x + halfWidth, y: centerPoint.y + halfHeight },
  }
}

function boundsFor(center: { lat: number; lng: number }, zoom: number, viewport: { width: number; height: number }) {
  const projection = viewportProjection(center, zoom, viewport)
  const topLeft = unproject(projection.topLeft.x, projection.topLeft.y, zoom)
  const bottomRight = unproject(projection.bottomRight.x, projection.bottomRight.y, zoom)
  return {
    min_lat: bottomRight.lat,
    max_lat: topLeft.lat,
    min_lng: topLeft.lng,
    max_lng: bottomRight.lng,
  }
}

function mapTiles(projection: ReturnType<typeof viewportProjection>) {
  const tileCount = 2 ** projection.zoom
  const tiles = []
  for (let x = Math.floor(projection.topLeft.x); x <= Math.floor(projection.bottomRight.x); x += 1) {
    for (let y = Math.floor(projection.topLeft.y); y <= Math.floor(projection.bottomRight.y); y += 1) {
      if (y < 0 || y >= tileCount) continue
      const wrappedX = ((x % tileCount) + tileCount) % tileCount
      tiles.push({
        x,
        urlX: wrappedX,
        y,
        zoom: projection.zoom,
        left: (x - projection.topLeft.x) * TILE_SIZE,
        top: (y - projection.topLeft.y) * TILE_SIZE,
      })
    }
  }
  return tiles
}

export default function MapPage() {
  const mapRef = useRef<HTMLDivElement | null>(null)
  const [center, setCenter] = useState({ lat: 60.1699, lng: 24.9384 })
  const [zoomLevel, setZoomLevel] = useState(14)
  const [viewport, setViewport] = useState({ width: 1200, height: 720 })
  const [source, setSource] = useState('')
  const [kind, setKind] = useState('ad')
  const [selected, setSelected] = useState<MapMarker | null>(null)
  const projection = useMemo(() => viewportProjection(center, zoomLevel, viewport), [center, zoomLevel, viewport])
  const bounds = useMemo(() => boundsFor(center, zoomLevel, viewport), [center, zoomLevel, viewport])
  const tiles = useMemo(() => mapTiles(projection), [projection])
  useEffect(() => {
    if (!mapRef.current) return
    const element = mapRef.current
    const update = () => setViewport({ width: element.clientWidth, height: element.clientHeight })
    update()
    const observer = new ResizeObserver(update)
    observer.observe(element)
    return () => observer.disconnect()
  }, [])
  const query = useQuery({
    queryKey: ['sale-listing-map', bounds, source, kind],
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
      return customInstance<MapResponse>(`/api/v1/sale-listings/map?${params.toString()}`)
    },
    staleTime: 30_000,
  })
  const markers = query.data?.data.markers ?? []
  function move(deltaLat: number, deltaLng: number) {
    setCenter(value => {
      const point = project(value.lat, value.lng, zoomLevel)
      return unproject(point.x + (deltaLng * viewport.width) / TILE_SIZE, point.y - (deltaLat * viewport.height) / TILE_SIZE, zoomLevel)
    })
    setSelected(null)
  }
  function zoom(multiplier: number) {
    setZoomLevel(value => Math.max(8, Math.min(18, value + (multiplier < 1 ? 1 : -1))))
    setSelected(null)
  }
  function handleWheel(event: WheelEvent<HTMLDivElement>) {
    event.preventDefault()
    zoom(event.deltaY > 0 ? 1.18 : 0.84)
  }
  function useCurrentLocation() {
    navigator.geolocation?.getCurrentPosition(position => {
      setCenter({ lat: position.coords.latitude, lng: position.coords.longitude })
      setZoomLevel(15)
      setSelected(null)
    })
  }
  return (
    <div className="map-layout">
      <Nav actions={<span className="search-total">{markers.length.toLocaleString('fi-FI')} locations</span>} />
      <div className="map-toolbar">
        <select className="search-select" value={kind} onChange={event => setKind(event.target.value)}>
          {KIND_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
        </select>
        <select className="search-select" value={source} onChange={event => setSource(event.target.value)}>
          {SOURCE_OPTIONS.map(option => <option key={option.value} value={option.value}>{option.label}</option>)}
        </select>
        <button className="search-clear-btn" type="button" onClick={useCurrentLocation}>Current location</button>
      </div>
      <div className="map-shell">
        <div ref={mapRef} className="map-canvas" onWheel={handleWheel} onDoubleClick={() => zoom(0.58)}>
          <div className="map-tiles" aria-hidden="true">
            {tiles.map(tile => (
              <img
                key={`${tile.zoom}:${tile.x}:${tile.y}`}
                className="map-tile"
                src={`https://tile.openstreetmap.org/${tile.zoom}/${tile.urlX}/${tile.y}.png`}
                style={{ left: tile.left, top: tile.top, width: TILE_SIZE, height: TILE_SIZE }}
                alt=""
              />
            ))}
          </div>
          <div className="map-grid" />
          <button className="map-pan map-pan--up" type="button" onClick={() => move(0.35, 0)}>↑</button>
          <button className="map-pan map-pan--down" type="button" onClick={() => move(-0.35, 0)}>↓</button>
          <button className="map-pan map-pan--left" type="button" onClick={() => move(0, -0.35)}>←</button>
          <button className="map-pan map-pan--right" type="button" onClick={() => move(0, 0.35)}>→</button>
          <div className="map-zoom-controls">
            <button type="button" onClick={() => zoom(0.58)} aria-label="Zoom in">+</button>
            <button type="button" onClick={() => zoom(1.72)} aria-label="Zoom out">-</button>
          </div>
          {query.isPending && <div className="map-loading">Loading locations…</div>}
          {markers.map(marker => {
            const point = project(marker.lat, marker.lng, projection.zoom)
            const left = (point.x - projection.topLeft.x) * TILE_SIZE
            const top = (point.y - projection.topLeft.y) * TILE_SIZE
            const size = markerSize(marker.count)
            return (
              <button
                key={`${marker.lat}:${marker.lng}:${marker.count}`}
                className="map-marker"
                style={{ left, top, width: size, height: size }}
                type="button"
                onClick={() => setSelected(marker)}
                title={marker.address || `${marker.count} listings`}
              >
                {marker.count}
              </button>
            )
          })}
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
