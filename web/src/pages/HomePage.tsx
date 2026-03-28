import { useState } from 'react'
import { useNavigate } from 'react-router-dom'

export default function HomePage() {
  const [input, setInput] = useState('')
  const navigate = useNavigate()

  function handleSearch(e: React.FormEvent) {
    e.preventDefault()
    const trimmed = input.trim()
    if (!trimmed) return
    navigate('/detail?' + new URLSearchParams({ id: trimmed }))
  }

  return (
    <div className="home-layout">
      <div className="home-card">
        <div className="home-logo">
          <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
          Koditon
        </div>
        <p className="home-desc">Look up Finnish real estate ad and building details by canonical ID or source URL.</p>
        <form className="home-search-form" onSubmit={handleSearch}>
          <input
            className="filter-input home-search-input"
            type="text"
            placeholder="shortcut:ad:12345  or  https://..."
            value={input}
            onChange={e => setInput(e.target.value)}
            autoFocus
          />
          <button className="passkey-btn home-search-btn" type="submit" disabled={!input.trim()}>
            Look up
          </button>
        </form>
        <p className="home-hint">
          Examples: <code>shortcut:ad:12345</code>, <code>frontdoor:building:uuid</code>
        </p>
      </div>
    </div>
  )
}
