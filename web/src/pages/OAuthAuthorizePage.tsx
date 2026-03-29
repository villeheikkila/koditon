import { useState, useEffect } from 'react'
import { getToken, passkeySignIn, appleSignIn, isAppleSignInConfigured } from '../lib/auth'

type Phase = 'loading' | 'sign-in' | 'code-entry' | 'review' | 'approving' | 'denied' | 'error'

interface HandoffDetails {
  handoff_id: string
  client_display_name: string
  redirect_host: string
  scopes: string[]
  expires_at_unix: number
}

const SCOPE_DESCRIPTIONS: Record<string, string> = {
  'mcp:core:read': 'Read real estate listing and transaction data',
}

function scopeLabel(scope: string): string {
  return SCOPE_DESCRIPTIONS[scope] ?? scope
}

export default function OAuthAuthorizePage() {
  const params = new URLSearchParams(window.location.search)
  const handoffToken = params.get('handoff_token') ?? ''

  const [phase, setPhase] = useState<Phase>('loading')
  const [handoff, setHandoff] = useState<HandoffDetails | null>(null)
  const [errorMessage, setErrorMessage] = useState('')
  const [signInError, setSignInError] = useState('')
  const [signInLoading, setSignInLoading] = useState(false)
  const [userCode, setUserCode] = useState('')

  async function resolveHandoff(accessToken: string, hToken: string, code: string) {
    setPhase('loading')
    const body: Record<string, string> = {}
    if (hToken) {
      body.handoff_token = hToken
    } else if (code.trim()) {
      body.user_code = code.trim().toUpperCase()
    } else {
      setPhase('code-entry')
      return
    }

    try {
      const res = await fetch('/oauth/authorize/handoff/resolve', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${accessToken}`,
        },
        body: JSON.stringify(body),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        setErrorMessage(err.error_description ?? 'Authorization request not found or expired.')
        setPhase('error')
        return
      }
      const data = await res.json()
      setHandoff(data)
      setPhase('review')
    } catch {
      setErrorMessage('Failed to connect to server.')
      setPhase('error')
    }
  }

  useEffect(() => {
    const token = getToken()
    if (!token) {
      setPhase(handoffToken ? 'sign-in' : 'sign-in')
      return
    }
    if (handoffToken) {
      resolveHandoff(token, handoffToken, '')
    } else {
      setPhase('code-entry')
    }
  }, [])

  async function handlePasskeySignIn() {
    setSignInLoading(true)
    setSignInError('')
    try {
      await passkeySignIn()
      const token = getToken()!
      await resolveHandoff(token, handoffToken, userCode)
    } catch (e) {
      setSignInError(e instanceof Error ? e.message : 'Sign in failed')
      setPhase('sign-in')
    } finally {
      setSignInLoading(false)
    }
  }

  async function handleAppleSignIn() {
    setSignInLoading(true)
    setSignInError('')
    try {
      await appleSignIn()
      const token = getToken()!
      await resolveHandoff(token, handoffToken, userCode)
    } catch (e) {
      setSignInError(e instanceof Error ? e.message : 'Apple sign in failed')
      setPhase('sign-in')
    } finally {
      setSignInLoading(false)
    }
  }

  async function handleCodeSubmit(e: React.FormEvent) {
    e.preventDefault()
    const token = getToken()
    if (!token) {
      setPhase('sign-in')
      return
    }
    await resolveHandoff(token, '', userCode)
  }

  async function handleApprove() {
    if (!handoff) return
    setPhase('approving')
    try {
      const res = await fetch('/oauth/authorize/handoff/approve', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${getToken()}`,
        },
        body: JSON.stringify({ handoff_id: handoff.handoff_id }),
      })
      if (!res.ok) {
        const err = await res.json().catch(() => ({}))
        setErrorMessage(err.error_description ?? 'Failed to approve the request.')
        setPhase('error')
        return
      }
      const data = await res.json()
      if (data.redirect_url) {
        window.location.href = data.redirect_url
      } else {
        setErrorMessage('Approved but no redirect URL received.')
        setPhase('error')
      }
    } catch {
      setErrorMessage('Failed to connect to server.')
      setPhase('error')
    }
  }

  async function handleDeny() {
    if (!handoff) return
    try {
      await fetch('/oauth/authorize/handoff/deny', {
        method: 'POST',
        headers: {
          'Content-Type': 'application/json',
          Authorization: `Bearer ${getToken()}`,
        },
        body: JSON.stringify({ handoff_id: handoff.handoff_id }),
      })
    } catch {}
    setPhase('denied')
  }

  if (phase === 'loading' || phase === 'approving') {
    return (
      <div className="signin-layout">
        <div className="signin-card">
          <div className="signin-logo">
            <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
            Koditon
          </div>
          <div className="spinner" style={{ marginTop: 8 }} />
          <p className="signin-desc">
            {phase === 'approving' ? 'Approving…' : 'Loading…'}
          </p>
        </div>
      </div>
    )
  }

  if (phase === 'sign-in') {
    return (
      <div className="signin-layout">
        <div className="signin-card">
          <div className="signin-logo">
            <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
            Koditon
          </div>
          <p className="signin-desc">Sign in to authorize the request</p>

          <button className="passkey-btn" onClick={handlePasskeySignIn} disabled={signInLoading}>
            {signInLoading ? (
              <><div className="spinner" style={{ width: 14, height: 14 }} />Authenticating…</>
            ) : (
              <><PasskeyIcon />Sign in with passkey</>
            )}
          </button>

          {isAppleSignInConfigured() && (
            <>
              <div className="signin-divider">or</div>
              <button className="apple-btn" onClick={handleAppleSignIn} disabled={signInLoading}>
                {signInLoading ? (
                  <><div className="spinner" style={{ width: 14, height: 14 }} />Signing in…</>
                ) : (
                  <><AppleIcon />Sign in with Apple</>
                )}
              </button>
            </>
          )}

          {signInError && <p className="signin-error">{signInError}</p>}
        </div>
      </div>
    )
  }

  if (phase === 'code-entry') {
    return (
      <div className="signin-layout">
        <div className="signin-card">
          <div className="signin-logo">
            <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
            Koditon
          </div>
          <p className="signin-desc">Enter the code shown by your MCP client</p>

          <form onSubmit={handleCodeSubmit} style={{ width: '100%', display: 'flex', flexDirection: 'column', gap: 8 }}>
            <input
              className="oauth-code-input"
              value={userCode}
              onChange={e => setUserCode(e.target.value)}
              placeholder="e.g. ABCD"
              maxLength={12}
              spellCheck={false}
              autoComplete="off"
              autoCapitalize="characters"
              autoFocus
            />
            <button type="submit" className="passkey-btn" disabled={!userCode.trim()}>
              Continue
            </button>
          </form>
        </div>
      </div>
    )
  }

  if (phase === 'denied') {
    return (
      <div className="signin-layout">
        <div className="signin-card">
          <div className="signin-logo">
            <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
            Koditon
          </div>
          <p className="signin-desc">Access denied</p>
          <p className="signin-hint">You denied the authorization request. You can close this tab.</p>
        </div>
      </div>
    )
  }

  if (phase === 'error') {
    return (
      <div className="signin-layout">
        <div className="signin-card">
          <div className="signin-logo">
            <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
            Koditon
          </div>
          <p className="signin-error">{errorMessage}</p>
          <p className="signin-hint">You can close this tab and try again.</p>
        </div>
      </div>
    )
  }

  if (phase === 'review' && handoff) {
    const expiresAt = new Date(handoff.expires_at_unix * 1000)
    const expired = expiresAt < new Date()

    return (
      <div className="signin-layout">
        <div className="oauth-review-card">
          <div className="signin-logo" style={{ marginBottom: 4 }}>
            <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
            Koditon
          </div>

          <div className="oauth-client-section">
            <p className="oauth-client-kicker">Requesting access</p>
            <p className="oauth-client-name">{handoff.client_display_name}</p>
            <p className="oauth-client-host">{handoff.redirect_host}</p>
          </div>

          <div className="oauth-scope-section">
            <p className="oauth-scope-heading">Permissions requested</p>
            <ul className="oauth-scope-list">
              {handoff.scopes.map(scope => (
                <li key={scope} className="oauth-scope-item">
                  <CheckIcon />
                  <span>{scopeLabel(scope)}</span>
                </li>
              ))}
            </ul>
          </div>

          {expired ? (
            <p className="signin-error">This authorization request has expired. Please restart the connection from your MCP client.</p>
          ) : (
            <div className="oauth-actions">
              <button className="passkey-btn" onClick={handleApprove}>
                Authorize
              </button>
              <button className="oauth-deny-btn" onClick={handleDeny}>
                Deny
              </button>
            </div>
          )}
        </div>
      </div>
    )
  }

  return null
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

function AppleIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 814 1000" fill="currentColor" aria-hidden="true">
      <path d="M788.1 340.9c-5.8 4.5-108.2 62.2-108.2 190.5 0 148.4 130.3 200.9 134.2 202.2-.6 3.2-20.7 71.9-68.7 141.9-42.8 61.6-87.5 123.1-155.5 123.1s-85.5-39.5-164-39.5c-76 0-103.7 40.8-165.9 40.8s-105-42.3-150.3-109.2-89.6-185.1-89.6-279.1c0-186.3 121.4-284.8 240.8-284.8 108.2 0 159.9 72.2 168.5 74.1 4.5.6 4.5 0 0 0C576.5 275 696.5 340.9 788.1 340.9z M544.4 84.3C576.5 45.7 599 10.1 599 10.1c2.6-5.8 2.6-11.7 0-14.9-3.2-3.2-9-3.2-14.9 0-25.1 8.3-103.7 44.1-152.3 113.8-51.1 73.4-64.1 157.3-64.1 157.3s1.9 1.3 6.4 1.3c23.2 0 107.8-32.1 170.3-183.3z" />
    </svg>
  )
}

function CheckIcon() {
  return (
    <svg width="14" height="14" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.5" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <polyline points="20 6 9 17 4 12" />
    </svg>
  )
}
