import { useState } from 'react'
import { passkeySignIn } from '../lib/auth'

interface Props {
  onSignIn: () => void
}

export default function SignInPage({ onSignIn }: Props) {
  const [state, setState] = useState<'idle' | 'loading' | 'error'>('idle')
  const [error, setError] = useState<string | null>(null)

  async function handleSignIn() {
    setState('loading')
    setError(null)
    try {
      await passkeySignIn()
      onSignIn()
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Sign in failed'
      setError(msg)
      setState('error')
    }
  }

  const isLoading = state === 'loading'

  return (
    <div className="signin-layout">
      <div className="signin-card">
        <div className="signin-logo">
          <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
          Koditon
        </div>
        <p className="signin-desc">Finnish real estate price data</p>

        <button
          className="passkey-btn"
          onClick={handleSignIn}
          disabled={isLoading}
        >
          {isLoading ? (
            <>
              <div className="spinner" style={{ width: 14, height: 14 }} />
              Authenticating…
            </>
          ) : (
            <>
              <PasskeyIcon />
              Sign in with passkey
            </>
          )}
        </button>

        {error && (
          <p className="signin-error">{error}</p>
        )}

        <p className="signin-hint">
          Use your device biometrics or security key to sign in.
        </p>
      </div>
    </div>
  )
}

function PasskeyIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="4" />
      <path d="M16 8h6" />
      <path d="M19 5v6" />
      <path d="M2 20s1-4 6-4 6 4 6 4" />
    </svg>
  )
}
