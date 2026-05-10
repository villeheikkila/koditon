import Nav from '../components/Nav'

export default function OAuthAuthorizePage() {
  return (
    <main className="app-shell">
      <Nav />
      <section className="page-section">
        <div className="section-inner">
          <h1>Authorization</h1>
          <p className="muted">OAuth browser handoff is not part of the property model API client.</p>
        </div>
      </section>
    </main>
  )
}
