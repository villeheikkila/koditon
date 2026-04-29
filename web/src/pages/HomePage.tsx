import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

export default function HomePage() {
  const [input, setInput] = useState('')
  const navigate = useNavigate()

  function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = input.trim()
    if (!trimmed) return
    const id = encodeURIComponent(trimmed)
    if (trimmed.startsWith('b_') || trimmed.split(':')[1] === 'building') {
      navigate(`/building/${id}`)
      return
    }
    if (trimmed.startsWith('r_')) {
      navigate(`/rental/${id}`)
      return
    }
    navigate(`/listing/${id}`)
  }

  return (
    <div className="home-layout">
      <div className="home-card">
        <div className="home-logo">
          <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
          Koditon
        </div>
        <p className="home-desc">Look up Finnish real estate listing and building details by ID or source URL.</p>
        <form className="home-search-form" onSubmit={handleSearch}>
          <input
            className="filter-input home-search-input"
            type="text"
            placeholder="l_abc123...  or  https://..."
            value={input}
            onChange={e => setInput(e.target.value)}
            autoFocus
          />
          <button className="passkey-btn home-search-btn" type="submit" disabled={!input.trim()}>
            Look up
          </button>
        </form>
        <p className="home-hint">
          Examples: <code>l_abc123...</code>, <code>b_abc123...</code>
        </p>
      </div>
    </div>
  )
}
