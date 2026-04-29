import { useState } from 'react'
import { passkeySignIn, appleSignIn, isAppleSignInConfigured, requestEmailSignIn } from '../lib/auth'

interface Props {
  onSignIn: () => void
}

export default function SignInPage({ onSignIn }: Props) {
  const [state, setState] = useState<'idle' | 'loading' | 'error'>('idle')
  const [appleState, setAppleState] = useState<'idle' | 'loading' | 'error'>('idle')
  const [emailState, setEmailState] = useState<'idle' | 'loading' | 'sent' | 'error'>('idle')
  const [email, setEmail] = useState('')
  const [error, setError] = useState<string | null>(null)
  const appleConfigured = isAppleSignInConfigured()

  async function handleAppleSignIn() {
    setAppleState('loading')
    setError(null)
    try {
      await appleSignIn()
      onSignIn()
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Apple sign in failed'
      setError(msg)
      setAppleState('error')
    }
  }

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

  async function handleEmailSignIn(event: React.FormEvent) {
    event.preventDefault()
    setEmailState('loading')
    setError(null)
    try {
      await requestEmailSignIn(email.trim())
      setEmailState('sent')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Email sign in failed'
      setError(msg)
      setEmailState('error')
    }
  }

  const isLoading = state === 'loading'
  const isAppleLoading = appleState === 'loading'
  const isEmailLoading = emailState === 'loading'

  return (
    <div className="signin-layout">
      <div className="signin-card">
        <div className="signin-logo">
          <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
          Koditon
        </div>
        <p className="signin-desc">Finnish real estate price data</p>

        <form className="signin-email-form" onSubmit={handleEmailSignIn}>
          <input
            className="signin-email-input"
            type="email"
            placeholder="you@example.com"
            value={email}
            onChange={e => setEmail(e.target.value)}
            autoComplete="email"
            disabled={isEmailLoading}
          />
          <button className="passkey-btn" type="submit" disabled={isEmailLoading || !email.trim()}>
            {isEmailLoading ? (
              <>
                <div className="spinner" style={{ width: 14, height: 14 }} />
                Sending…
              </>
            ) : emailState === 'sent' ? (
              'Check your email'
            ) : (
              'Continue with email'
            )}
          </button>
        </form>

        <div className="signin-divider">or</div>

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

        {appleConfigured && (
          <button
            className="apple-btn"
            onClick={handleAppleSignIn}
            disabled={isAppleLoading || isLoading}
          >
            {isAppleLoading ? (
              <>
                <div className="spinner" style={{ width: 14, height: 14 }} />
                Signing in…
              </>
            ) : (
              <>
                <AppleIcon />
                Sign in with Apple
              </>
            )}
          </button>
        )}

        {error && (
          <p className="signin-error">{error}</p>
        )}

        <p className="signin-hint">
          Use email first. Add a passkey later from account settings.
        </p>
      </div>
    </div>
  )
}

function AppleIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 814 1000" fill="currentColor" aria-hidden="true">
      <path d="M788.1 340.9c-5.8 4.5-108.2 62.2-108.2 190.5 0 148.4 130.3 200.9 134.2 202.2-.6 3.2-20.7 71.9-68.7 141.9-42.8 61.6-87.5 123.1-155.5 123.1s-85.5-39.5-164-39.5c-76 0-103.7 40.8-165.9 40.8s-105-42.3-150.3-109.2-89.6-185.1-89.6-279.1c0-186.3 121.4-284.8 240.8-284.8 108.2 0 159.9 72.2 168.5 74.1 4.5.6 4.5 0 0 0C576.5 275 696.5 340.9 788.1 340.9z M544.4 84.3C576.5 45.7 599 10.1 599 10.1c2.6-5.8 2.6-11.7 0-14.9-3.2-3.2-9-3.2-14.9 0-25.1 8.3-103.7 44.1-152.3 113.8-51.1 73.4-64.1 157.3-64.1 157.3s1.9 1.3 6.4 1.3c23.2 0 107.8-32.1 170.3-183.3z"/>
    </svg>
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
