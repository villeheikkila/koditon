import { useCallback, useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { getToken, clearToken } from './lib/auth'
import SignInPage from './pages/SignInPage'
import DashboardPage from './pages/DashboardPage'
import DetailPage from './pages/DetailPage'
import SearchPage from './pages/SearchPage'
import OAuthAuthorizePage from './pages/OAuthAuthorizePage'
import EmailConfirmPage from './pages/EmailConfirmPage'

export default function App() {
  const [authenticated, setAuthenticated] = useState(() => !!getToken())

  const handleSignIn = useCallback(function handleSignIn() {
    setAuthenticated(true)
  }, [])

  const handleSignOut = useCallback(function handleSignOut() {
    clearToken()
    setAuthenticated(false)
  }, [])

  return (
    <BrowserRouter>
      <Routes>
        {/* Public routes */}
        <Route path="/listing/:id" element={<DetailPage kind="listing" />} />
        <Route path="/rental/:id" element={<DetailPage kind="rental" />} />
        <Route path="/building/:id" element={<DetailPage kind="building" />} />
        <Route path="/search" element={<SearchPage />} />
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
