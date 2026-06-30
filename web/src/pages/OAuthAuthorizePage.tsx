import { useCallback, useEffect, useState } from 'react'
import { useSearchParams } from 'react-router-dom'
import { getValidAccessToken } from '../lib/auth-store'
import SignInPage from './SignInPage'

type Props = {
  authenticated: boolean
  onSignIn: () => void
}

type HandoffDetails = {
  handoff_id: string
  client_id: string
  client_display_name: string
  redirect_host: string
  scopes: string[]
  expires_at_unix: number
}

type ApprovalResult = {
  ok: boolean
  redirect_url?: string
}

type ViewState = 'loading' | 'ready' | 'approving' | 'denying' | 'done' | 'error'

export default function OAuthAuthorizePage({ authenticated, onSignIn }: Props) {
  const [searchParams] = useSearchParams()
  const [state, setState] = useState<ViewState>('loading')
  const [details, setDetails] = useState<HandoffDetails | null>(null)
  const [error, setError] = useState<string | null>(null)
  const handoffToken = searchParams.get('handoff_token')?.trim() ?? ''

  const resolveHandoff = useCallback(async function resolveHandoff() {
    if (!authenticated) return
    if (!handoffToken) {
      setState('error')
      setError('Missing OAuth authorization handoff.')
      return
    }
    setState('loading')
    setError(null)
    try {
      const token = await getValidAccessToken()
      if (!token) throw new Error('Sign in is required.')
      const response = await fetch('/oauth/authorize/handoff/resolve', {
        method: 'POST',
        credentials: 'include',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ handoff_token: handoffToken }),
      })
      const payload = await readJSON(response)
      if (!response.ok) throw new Error(oauthErrorMessage(payload, 'Unable to load authorization request.'))
      setDetails(payload as HandoffDetails)
      setState('ready')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to load authorization request.')
      setState('error')
    }
  }, [authenticated, handoffToken])

  useEffect(() => {
    void resolveHandoff()
  }, [resolveHandoff])

  async function approve() {
    if (!details) return
    setState('approving')
    setError(null)
    try {
      const token = await getValidAccessToken()
      if (!token) throw new Error('Sign in is required.')
      const response = await fetch('/oauth/authorize/handoff/approve', {
        method: 'POST',
        credentials: 'include',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ handoff_id: details.handoff_id }),
      })
      const payload = await readJSON(response)
      if (!response.ok) throw new Error(oauthErrorMessage(payload, 'Unable to approve authorization.'))
      const result = payload as ApprovalResult
      if (result.redirect_url) {
        window.location.assign(result.redirect_url)
        return
      }
      setState('done')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to approve authorization.')
      setState('ready')
    }
  }

  async function deny() {
    if (!details) return
    setState('denying')
    setError(null)
    try {
      const token = await getValidAccessToken()
      if (!token) throw new Error('Sign in is required.')
      const response = await fetch('/oauth/authorize/handoff/deny', {
        method: 'POST',
        credentials: 'include',
        headers: {
          Authorization: `Bearer ${token}`,
          'Content-Type': 'application/json',
        },
        body: JSON.stringify({ handoff_id: details.handoff_id }),
      })
      const payload = await readJSON(response)
      if (!response.ok) throw new Error(oauthErrorMessage(payload, 'Unable to deny authorization.'))
      setState('done')
    } catch (err) {
      setError(err instanceof Error ? err.message : 'Unable to deny authorization.')
      setState('ready')
    }
  }

  if (!authenticated) {
    return <SignInPage onSignIn={onSignIn} />
  }

  return (
    <main className="signin-layout">
      <section className="oauth-review-card">
        <div className="signin-logo">
          <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
          Koditon
        </div>
        {state === 'loading' && (
          <div className="loading-state" style={{ padding: 18 }}>
            <div className="spinner" />
          </div>
        )}
        {(state === 'ready' || state === 'approving' || state === 'denying') && details && (
          <>
            <div className="oauth-client-section">
              <div className="oauth-client-kicker">Authorize access</div>
              <div className="oauth-client-name">{details.client_display_name || details.client_id}</div>
              <div className="oauth-client-host">{details.redirect_host}</div>
            </div>
            <div className="oauth-scope-section">
              <div className="oauth-scope-heading">Permissions</div>
              <ul className="oauth-scope-list">
                {details.scopes.map(scope => (
                  <li className="oauth-scope-item" key={scope}>
                    <CheckIcon />
                    <span>{scopeLabel(scope)}</span>
                  </li>
                ))}
              </ul>
            </div>
            {error && <p className="signin-error">{error}</p>}
            <div className="oauth-actions">
              <button className="passkey-btn" onClick={approve} disabled={state === 'approving' || state === 'denying'}>
                {state === 'approving' ? (
                  <>
                    <div className="spinner" style={{ width: 14, height: 14 }} />
                    Authorizing…
                  </>
                ) : (
                  'Authorize'
                )}
              </button>
              <button className="oauth-deny-btn" onClick={deny} disabled={state === 'approving' || state === 'denying'}>
                {state === 'denying' ? 'Denying…' : 'Deny'}
              </button>
            </div>
            <p className="signin-hint">You can revoke connected apps from your account later.</p>
          </>
        )}
        {state === 'done' && (
          <div className="oauth-client-section">
            <div className="oauth-client-name">Request handled</div>
            <div className="signin-desc">You can return to the app that requested access.</div>
          </div>
        )}
        {state === 'error' && (
          <>
            <div className="oauth-client-section">
              <div className="oauth-client-name">Authorization unavailable</div>
              <div className="signin-desc">{error ?? 'Unable to load authorization request.'}</div>
            </div>
            <button className="passkey-btn" onClick={resolveHandoff}>Try again</button>
          </>
        )}
      </section>
    </main>
  )
}

async function readJSON(response: Response): Promise<unknown> {
  const contentType = response.headers.get('content-type') ?? ''
  if (!contentType.includes('application/json')) {
    return response.text()
  }
  return response.json()
}

function oauthErrorMessage(payload: unknown, fallback: string): string {
  if (typeof payload === 'object' && payload != null) {
    if ('error_description' in payload && payload.error_description != null) return String(payload.error_description)
    if ('detail' in payload && payload.detail != null) return String(payload.detail)
    if ('error' in payload && payload.error != null) return String(payload.error)
  }
  if (typeof payload === 'string' && payload.trim()) return payload.trim()
  return fallback
}

function scopeLabel(scope: string): string {
  switch (scope) {
    case 'mcp:core:read':
      return 'Read Koditon property and market data through MCP'
    case 'core:read':
      return 'Read Koditon property and market data'
    case 'profile:read':
      return 'Read your Koditon profile'
    case 'profile:write':
      return 'Update your Koditon profile'
    default:
      return scope
  }
}

function CheckIcon() {
  return (
    <svg width="15" height="15" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2.25" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <path d="M20 6 9 17l-5-5" />
    </svg>
  )
}
