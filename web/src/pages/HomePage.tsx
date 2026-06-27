import { useState } from 'react'
import { useNavigate } from 'react-router-dom'
import { entityDetail } from '../api/koditon'
import { buildAddressLookupPath, looksLikeEntityInput } from '../lib/address-lookup'

export default function HomePage() {
  const [input, setInput] = useState('')
  const [error, setError] = useState('')
  const [loading, setLoading] = useState(false)
  const navigate = useNavigate()

  async function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = input.trim()
    if (!trimmed) return
    setError('')
    if (!looksLikeEntityInput(trimmed)) {
      navigate(buildAddressLookupPath({ address: trimmed }))
      return
    }
    setLoading(true)
    try {
      const response = await entityDetail({ id: trimmed })
      if (response.status === 200 && response.data.street_address) {
        navigate(buildAddressLookupPath({ address: response.data.street_address, city: response.data.city, postal: response.data.postal, source: response.data.source }))
        return
      }
      setError('No address found for that listing.')
    } catch {
      setError('Could not resolve that listing. Try a street address instead.')
    } finally {
      setLoading(false)
    }
  }

  return (
    <div className="home-layout">
      <div className="home-card">
        <div className="home-logo">
          <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
          Koditon
        </div>
        <p className="home-desc">Look up an address, listing URL, or source canonical ID.</p>
        <form className="home-search-form" onSubmit={handleSearch}>
          <input
            className="filter-input home-search-input"
            type="text"
            placeholder="Askvägen 4, 22100 Maarianhamina"
            value={input}
            onChange={e => setInput(e.target.value)}
            autoFocus
          />
          <button className="passkey-btn home-search-btn" type="submit" disabled={!input.trim() || loading}>
            {loading ? 'Resolving' : 'Look up'}
          </button>
        </form>
        {error && <p className="home-error">{error}</p>}
        <p className="home-hint">
          Examples: <code>Askvägen 4</code>, <code>frontdoor:ad:21531967</code>, <code>https://...</code>
        </p>
      </div>
    </div>
  )
}
