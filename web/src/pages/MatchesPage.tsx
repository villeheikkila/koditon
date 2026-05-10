import Nav from '../components/Nav'

export default function MatchesPage() {
  return (
    <main className="app-shell">
      <Nav />
      <section className="page-section">
        <div className="section-inner">
          <h1>Linking workbench</h1>
          <p className="muted">Transaction and source-listing linking will move to canonical target relink APIs.</p>
        </div>
      </section>
    </main>
  )
}
