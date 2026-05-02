import { useEffect, useRef } from 'react'
import maplibregl from 'maplibre-gl'
import 'maplibre-gl/dist/maplibre-gl.css'

type ListingLocationMapProps = {
  latitude: number
  longitude: number
  label: string
}

const MAP_STYLE: maplibregl.StyleSpecification = {
  version: 8,
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

export default function ListingLocationMap({ latitude, longitude, label }: ListingLocationMapProps) {
  const mapRef = useRef<HTMLDivElement | null>(null)
  const mapInstanceRef = useRef<maplibregl.Map | null>(null)
  useEffect(() => {
    if (!mapRef.current || mapInstanceRef.current) return
    const map = new maplibregl.Map({
      container: mapRef.current,
      style: MAP_STYLE,
      center: [longitude, latitude],
      zoom: 15.5,
      attributionControl: false,
    })
    mapInstanceRef.current = map
    map.addControl(new maplibregl.NavigationControl({ showCompass: false }), 'top-left')
    map.addControl(new maplibregl.AttributionControl({ compact: true }), 'bottom-left')
    map.on('load', () => {
      map.addSource('listing-location', {
        type: 'geojson',
        data: {
          type: 'FeatureCollection',
          features: [
            {
              type: 'Feature',
              geometry: { type: 'Point', coordinates: [longitude, latitude] },
              properties: { label },
            },
          ],
        },
      })
      map.addLayer({
        id: 'listing-location-halo',
        type: 'circle',
        source: 'listing-location',
        paint: {
          'circle-radius': 30,
          'circle-color': '#d5793f',
          'circle-opacity': 0.18,
          'circle-blur': 0.5,
        },
      })
      map.addLayer({
        id: 'listing-location-marker',
        type: 'circle',
        source: 'listing-location',
        paint: {
          'circle-radius': 13,
          'circle-color': '#422312',
          'circle-opacity': 0.96,
          'circle-stroke-color': '#d9915a',
          'circle-stroke-width': 2,
        },
      })
    })
    return () => {
      map.remove()
      mapInstanceRef.current = null
    }
  }, [label, latitude, longitude])
  return (
    <div className="listing-map-card">
      <div className="listing-map-header">
        <div>
          <div className="listing-map-title">Location</div>
          <div className="listing-map-subtitle">{label}</div>
        </div>
      </div>
      <div ref={mapRef} className="listing-map-canvas" />
    </div>
  )
}
