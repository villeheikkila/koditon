import { useState } from 'react'
import { passkeyRegisterBegin, passkeyRegisterFinish } from '../lib/auth'

interface Props {
  onClose: () => void
}

export default function PasskeyManager({ onClose }: Props) {
  const [regState, setRegState] = useState<'idle' | 'loading' | 'success' | 'error'>('idle')
  const [regError, setRegError] = useState<string | null>(null)

  async function handleRegister() {
    setRegState('loading')
    setRegError(null)
    try {
      const { challenge_id, options } = await passkeyRegisterBegin()
      const credential = await navigator.credentials.create({ publicKey: options }) as PublicKeyCredential | null
      if (!credential) throw new Error('No credential created')
      await passkeyRegisterFinish(challenge_id, credential)
      setRegState('success')
    } catch (e) {
      const msg = e instanceof Error ? e.message : 'Registration failed'
      setRegError(msg)
      setRegState('error')
    }
  }

  return (
    <div className="modal-overlay" onClick={e => { if (e.target === e.currentTarget) onClose() }}>
      <div className="modal-card">
        <div className="modal-header">
          <span className="modal-title">Passkeys</span>
          <button className="modal-close" onClick={onClose} aria-label="Close">×</button>
        </div>

        <div className="modal-body">
          <p className="modal-desc">
            Add a passkey to sign in using your device biometrics or security key.
          </p>

          {regState === 'success' ? (
            <div className="reg-success">
              <span>Passkey registered successfully.</span>
            </div>
          ) : (
            <button
              className="passkey-btn"
              onClick={handleRegister}
              disabled={regState === 'loading'}
            >
              {regState === 'loading' ? (
                <>
                  <div className="spinner" style={{ width: 14, height: 14 }} />
                  Registering…
                </>
              ) : (
                <>
                  <AddPasskeyIcon />
                  Add a passkey
                </>
              )}
            </button>
          )}

          {regError && <p className="signin-error">{regError}</p>}
        </div>
      </div>
    </div>
  )
}

function AddPasskeyIcon() {
  return (
    <svg width="16" height="16" viewBox="0 0 24 24" fill="none" stroke="currentColor" strokeWidth="2" strokeLinecap="round" strokeLinejoin="round" aria-hidden="true">
      <circle cx="8" cy="8" r="4" />
      <path d="M16 11h6" />
      <path d="M19 8v6" />
      <path d="M2 20s1-4 6-4 6 4 6 4" />
    </svg>
  )
}
