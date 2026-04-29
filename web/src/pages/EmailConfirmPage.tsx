import { useEffect, useState } from 'react'
import { Link, useParams } from 'react-router-dom'
import { confirmEmailSignIn } from '../lib/auth'

interface Props {
  onSignIn: () => void
}

export default function EmailConfirmPage({ onSignIn }: Props) {
  const { token = '' } = useParams()
  const [state, setState] = useState<'loading' | 'done' | 'error'>(token ? 'loading' : 'error')
  const [error, setError] = useState('')

  useEffect(() => {
    if (!token) {
      return
    }
    confirmEmailSignIn(token)
      .then(() => {
        setState('done')
        onSignIn()
      })
      .catch(e => {
        setError(e instanceof Error ? e.message : 'Email sign in failed')
        setState('error')
      })
  }, [token, onSignIn])

  return (
    <div className="signin-layout">
      <div className="signin-card">
        <div className="signin-logo">
          <span className="header-logo-dot" style={{ width: 10, height: 10 }} />
          Koditon
        </div>
        {state === 'loading' && (
          <>
            <div className="spinner" />
            <p className="signin-desc">Signing you in…</p>
          </>
        )}
        {state === 'done' && <p className="signin-desc">Signed in.</p>}
        {state === 'error' && (
          <>
            <p className="signin-error">{error || 'Email sign-in link is missing.'}</p>
            <Link to="/" className="listing-back">← Back</Link>
          </>
        )}
      </div>
    </div>
  )
}
