import { useState } from 'react'
import { BrowserRouter, Routes, Route, Navigate } from 'react-router-dom'
import { getToken, clearToken } from './lib/auth'
import SignInPage from './pages/SignInPage'
import DashboardPage from './pages/DashboardPage'
import DetailPage from './pages/DetailPage'
import SearchPage from './pages/SearchPage'
import OAuthAuthorizePage from './pages/OAuthAuthorizePage'

export default function App() {
  const [authenticated, setAuthenticated] = useState(() => !!getToken())

  function handleSignIn() {
    setAuthenticated(true)
  }

  function handleSignOut() {
    clearToken()
    setAuthenticated(false)
  }

  return (
    <BrowserRouter>
      <Routes>
        {/* Public routes */}
        <Route path="/detail" element={<DetailPage />} />
        <Route path="/search" element={<SearchPage />} />
        <Route path="/oauth/authorize" element={<OAuthAuthorizePage />} />

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
