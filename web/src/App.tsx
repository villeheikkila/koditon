import { useCallback, useEffect, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { consumeReturnPath, hasAccessToken, restoreSession, signOutSession } from './lib/auth-store'
import SignInPage from './pages/SignInPage'
import DashboardPage from './pages/DashboardPage'
import DetailPage from './pages/DetailPage'
import SourceEntityPage from './pages/SourceEntityPage'
import SearchPage from './pages/SearchPage'
import AddressLookupPage from './pages/AddressLookupPage'
import MapPage from './pages/MapPage'
import MatchesPage from './pages/MatchesPage'
import OAuthAuthorizePage from './pages/OAuthAuthorizePage'
import EmailConfirmPage from './pages/EmailConfirmPage'

export default function App() {
  const [authReady, setAuthReady] = useState(false)
  const [authenticated, setAuthenticated] = useState(() => hasAccessToken())

  useEffect(() => {
    let cancelled = false
    restoreSession().then(ok => {
      if (cancelled) return
      setAuthenticated(ok)
      setAuthReady(true)
    })
    return () => {
      cancelled = true
    }
  }, [])

  const handleSignIn = useCallback(function handleSignIn() {
    setAuthenticated(true)
    const returnTo = consumeReturnPath()
    const currentPath = `${window.location.pathname}${window.location.search}${window.location.hash}`
    if (returnTo && returnTo !== currentPath) {
      window.location.assign(returnTo)
    } else if (window.location.pathname.startsWith('/email/confirm/')) {
      window.location.replace('/')
    }
  }, [])

  const handleSignOut = useCallback(async function handleSignOut() {
    await signOutSession()
    setAuthenticated(false)
  }, [])

  if (!authReady) {
    return null
  }

  return (
    <BrowserRouter>
      <Routes>
        {/* Public routes */}
        <Route path="/target/:targetType/:id" element={<DetailPage />} />
        <Route path="/listing/:id" element={<SourceEntityPage kind="listing" />} />
        <Route path="/rental/:id" element={<SourceEntityPage kind="rental" />} />
        <Route path="/housing-company/:id" element={<SourceEntityPage kind="housingCompany" />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/address" element={<AddressLookupPage />} />
        <Route path="/map" element={<MapPage />} />
        <Route path="/prices" element={authenticated ? <DashboardPage onSignOut={handleSignOut} /> : <SignInPage onSignIn={handleSignIn} />} />
        <Route path="/matches" element={authenticated ? <MatchesPage /> : <SignInPage onSignIn={handleSignIn} />} />
        <Route path="/oauth/authorize" element={<OAuthAuthorizePage authenticated={authenticated} onSignIn={handleSignIn} />} />
        <Route path="/email/confirm/:token" element={<EmailConfirmPage onSignIn={handleSignIn} />} />

        <Route path="/" element={<Navigate to="/address" replace />} />

        {/* Catch-all: redirect to root */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
