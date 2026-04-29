import {
  authAppleWeb,
  authEmailConfirm,
  authEmailRequest,
  authPasskeyAuthenticate,
  authPasskeyAuthenticateOptions,
  authPasskeyRegisterFinish,
  authPasskeyRegisterOptions,
} from '../api/koditon'

const TOKEN_KEY = 'koditon_access_token'
const DEVICE_ID_KEY = 'koditon_device_id'

export function getToken(): string | null {
  return localStorage.getItem(TOKEN_KEY)
}

export function setToken(token: string): void {
  localStorage.setItem(TOKEN_KEY, token)
}

export function clearToken(): void {
  localStorage.removeItem(TOKEN_KEY)
}

export function getOrCreateDeviceId(): string {
  let id = localStorage.getItem(DEVICE_ID_KEY)
  if (!id) {
    id = crypto.randomUUID()
    localStorage.setItem(DEVICE_ID_KEY, id)
  }
  return id
}

export async function passkeySignIn(): Promise<void> {
  const optionsRes = await authPasskeyAuthenticateOptions()
  if (optionsRes.status !== 200) throw new Error('Failed to get passkey options')
  const { challenge_id, options } = optionsRes.data
  const credential = await navigator.credentials.get({
    publicKey: decodePasskeyOptions(options as Record<string, unknown>),
  }) as PublicKeyCredential | null
  if (!credential) throw new Error('No credential returned')
  const tokenRes = await authPasskeyAuthenticate({
    challenge_id,
    credential_json: JSON.stringify(encodeCredential(credential)),
  }, {
    headers: {
      'X-Device-ID': getOrCreateDeviceId(),
    },
  })
  if (tokenRes.status !== 200) throw new Error('Passkey authentication failed')
  setToken(tokenRes.data.access_token)
}

export async function passkeyRegisterBegin(): Promise<{ challenge_id: string; options: PublicKeyCredentialCreationOptions }> {
  const token = getToken()
  const res = await authPasskeyRegisterOptions({
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  })
  if (res.status !== 200) throw new Error('Failed to get registration options')
  const { challenge_id, options } = res.data
  return { challenge_id, options: decodePasskeyCreationOptions(options as Record<string, unknown>) }
}

export async function passkeyRegisterFinish(challengeId: string, credential: PublicKeyCredential): Promise<string> {
  const token = getToken()
  const res = await authPasskeyRegisterFinish({
    challenge_id: challengeId,
    credential_json: JSON.stringify(encodeCredential(credential)),
  }, {
    headers: {
      ...(token ? { Authorization: `Bearer ${token}` } : {}),
    },
  })
  if (res.status !== 200) throw new Error('Passkey registration failed')
  return res.data.credential_id
}

// Apple JS SDK type declaration
declare const AppleID: {
  auth: {
    init(config: {
      clientId: string
      scope: string
      redirectURI: string
      state?: string
      usePopup?: boolean
    }): void
    signIn(): Promise<{
      authorization: { code: string; id_token: string; state?: string }
      user?: { email?: string; name?: { firstName?: string; lastName?: string } }
    }>
  }
}

export async function appleSignIn(): Promise<void> {
  const serviceId = import.meta.env.VITE_APPLE_SERVICE_ID
  const redirectURI = import.meta.env.VITE_APPLE_REDIRECT_URI
  if (!serviceId || !redirectURI) throw new Error('Apple Sign In is not configured')

  AppleID.auth.init({
    clientId: serviceId,
    scope: 'name email',
    redirectURI,
    usePopup: true,
  })

  const response = await AppleID.auth.signIn()
  const code = response.authorization.code
  const tokenRes = await authAppleWeb({ code }, {
    headers: {
      'X-Device-ID': getOrCreateDeviceId(),
    },
  })
  if (tokenRes.status !== 200) throw new Error('Apple Sign In failed')
  setToken(tokenRes.data.access_token)
}

export async function requestEmailSignIn(email: string): Promise<void> {
  await authEmailRequest({ email })
}

export async function confirmEmailSignIn(token: string): Promise<void> {
  const tokenRes = await authEmailConfirm({ token }, {
    headers: {
      'X-Device-ID': getOrCreateDeviceId(),
    },
  })
  if (tokenRes.status !== 200) throw new Error('Email sign in failed')
  setToken(tokenRes.data.access_token)
}

export function isAppleSignInConfigured(): boolean {
  return !!(import.meta.env.VITE_APPLE_SERVICE_ID && import.meta.env.VITE_APPLE_REDIRECT_URI)
}

// WebAuthn helpers — decode base64url strings from server to ArrayBuffer
function decodePasskeyOptions(options: Record<string, unknown>): PublicKeyCredentialRequestOptions {
  return {
    ...options,
    challenge: base64urlToBuffer(options.challenge as string),
    allowCredentials: (options.allowCredentials as Array<{ id: string; type: string; transports?: string[] }> | undefined)?.map(c => ({
      ...c,
      id: base64urlToBuffer(c.id),
    })),
  } as PublicKeyCredentialRequestOptions
}

function decodePasskeyCreationOptions(options: Record<string, unknown>): PublicKeyCredentialCreationOptions {
  const user = options.user as { id: string; name: string; displayName: string }
  return {
    ...options,
    challenge: base64urlToBuffer(options.challenge as string),
    user: { ...user, id: base64urlToBuffer(user.id) },
    excludeCredentials: (options.excludeCredentials as Array<{ id: string; type: string }> | undefined)?.map(c => ({
      ...c,
      id: base64urlToBuffer(c.id),
    })),
  } as unknown as PublicKeyCredentialCreationOptions
}

function encodeCredential(credential: PublicKeyCredential): Record<string, unknown> {
  const response = credential.response as AuthenticatorAssertionResponse | AuthenticatorAttestationResponse

  if ('authenticatorData' in response) {
    // assertion
    const assertion = response as AuthenticatorAssertionResponse
    return {
      id: credential.id,
      rawId: bufferToBase64url(credential.rawId),
      type: credential.type,
      response: {
        authenticatorData: bufferToBase64url(assertion.authenticatorData),
        clientDataJSON: bufferToBase64url(assertion.clientDataJSON),
        signature: bufferToBase64url(assertion.signature),
        userHandle: assertion.userHandle ? bufferToBase64url(assertion.userHandle) : null,
      },
    }
  } else {
    // attestation
    const attestation = response as AuthenticatorAttestationResponse
    return {
      id: credential.id,
      rawId: bufferToBase64url(credential.rawId),
      type: credential.type,
      response: {
        attestationObject: bufferToBase64url(attestation.attestationObject),
        clientDataJSON: bufferToBase64url(attestation.clientDataJSON),
        transports: attestation.getTransports?.() ?? [],
      },
    }
  }
}

function base64urlToBuffer(value: string): ArrayBuffer {
  const base64 = value.replace(/-/g, '+').replace(/_/g, '/')
  const padded = base64.padEnd(base64.length + (4 - (base64.length % 4)) % 4, '=')
  const binary = atob(padded)
  const buffer = new Uint8Array(binary.length)
  for (let i = 0; i < binary.length; i++) buffer[i] = binary.charCodeAt(i)
  return buffer.buffer
}

function bufferToBase64url(buffer: ArrayBuffer): string {
  const bytes = new Uint8Array(buffer)
  let binary = ''
  for (const b of bytes) binary += String.fromCharCode(b)
  return btoa(binary).replace(/\+/g, '-').replace(/\//g, '_').replace(/=/g, '')
}
