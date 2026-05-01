import { NavLink } from 'react-router-dom'

interface NavProps {
  actions?: React.ReactNode
}

export default function Nav({ actions }: NavProps) {
  return (
    <header className="nav">
      <div className="nav-logo">
        <span className="header-logo-dot" />
        Koditon
      </div>
      <nav className="nav-links">
        <NavLink
          to="/search"
          className={({ isActive }) => `nav-link${isActive ? ' nav-link--active' : ''}`}
        >
          Search
        </NavLink>
        <NavLink
          to="/"
          end
          className={({ isActive }) => `nav-link${isActive ? ' nav-link--active' : ''}`}
        >
          Prices
        </NavLink>
        <NavLink
          to="/matches"
          className={({ isActive }) => `nav-link${isActive ? ' nav-link--active' : ''}`}
        >
          Matches
        </NavLink>
      </nav>
      {actions && <div className="nav-actions">{actions}</div>}
    </header>
  )
}
