import { useState } from 'react'
import { getToken, clearToken } from './lib/auth'
import SignInPage from './pages/SignInPage'
import DashboardPage from './pages/DashboardPage'

export default function App() {
  const [authenticated, setAuthenticated] = useState(() => !!getToken())

  function handleSignIn() {
    setAuthenticated(true)
  }

  function handleSignOut() {
    clearToken()
    setAuthenticated(false)
  }

  if (!authenticated) {
    return <SignInPage onSignIn={handleSignIn} />
  }

  return <DashboardPage onSignOut={handleSignOut} />
}
