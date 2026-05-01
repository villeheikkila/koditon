import { useCallback, useEffect, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { consumeReturnPath, hasAccessToken, restoreSession, signOutSession } from './lib/auth-store'
import SignInPage from './pages/SignInPage'
import DashboardPage from './pages/DashboardPage'
import DetailPage from './pages/DetailPage'
import SearchPage from './pages/SearchPage'
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
    if (returnTo && returnTo !== `${window.location.pathname}${window.location.search}${window.location.hash}`) {
      window.location.assign(returnTo)
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
        <Route path="/listing/:id" element={<DetailPage kind="listing" />} />
        <Route path="/rental/:id" element={<DetailPage kind="rental" />} />
        <Route path="/building/:id" element={<DetailPage kind="building" />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/matches" element={authenticated ? <MatchesPage /> : <SignInPage onSignIn={handleSignIn} />} />
        <Route path="/oauth/authorize" element={<OAuthAuthorizePage />} />
        <Route path="/email/confirm/:token" element={<EmailConfirmPage onSignIn={handleSignIn} />} />

        {/* Auth-gated root */}
        <Route
          path="/"
          element={
            authenticated
              ? <DashboardPage onSignOut={handleSignOut} />
              : <SignInPage onSignIn={handleSignIn} />
          }
        />

        {/* Catch-all: redirect to root */}
        <Route path="*" element={<Navigate to="/" replace />} />
      </Routes>
    </BrowserRouter>
  )
}
