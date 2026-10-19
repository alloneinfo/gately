// Passkey（WebAuthn）浏览器端辅助：二进制与 base64url 互转，以及创建/获取凭据。

export function passkeySupported(): boolean {
  return typeof window !== 'undefined' && !!window.PublicKeyCredential && !!navigator.credentials
}

function toB64(buf: ArrayBuffer): string {
  let s = ''
  for (const b of new Uint8Array(buf)) {
    s += String.fromCharCode(b)
  }
  return btoa(s).replace(/\+/g, '-').replace(/\//g, '_').replace(/=+$/, '')
}

function fromB64(s: string): ArrayBuffer {
  const pad = '='.repeat((4 - (s.length % 4)) % 4)
  const bin = atob(s.replace(/-/g, '+').replace(/_/g, '/') + pad)
  const out = new Uint8Array(bin.length)
  for (let i = 0; i < bin.length; i++) {
    out[i] = bin.charCodeAt(i)
  }
  return out.buffer
}

interface CredDescriptor {
  type: string
  id: string
}

// 服务端下发的注册参数（二进制字段为 base64url）
export interface CreateOptions {
  challenge: string
  rp: { id: string; name: string }
  user: { id: string; name: string; displayName: string }
  pubKeyCredParams: { type: 'public-key'; alg: number }[]
  excludeCredentials: CredDescriptor[]
  authenticatorSelection: AuthenticatorSelectionCriteria
  attestation: AttestationConveyancePreference
  timeout: number
}

export interface GetOptions {
  challenge: string
  rpId: string
  userVerification: UserVerificationRequirement
  timeout: number
  allowCredentials: CredDescriptor[]
}

// createPasskey 调用浏览器创建 Passkey，返回可直接提交给服务端的 JSON。
export async function createPasskey(o: CreateOptions) {
  const cred = (await navigator.credentials.create({
    publicKey: {
      challenge: fromB64(o.challenge),
      rp: o.rp,
      user: { id: fromB64(o.user.id), name: o.user.name, displayName: o.user.displayName },
      pubKeyCredParams: o.pubKeyCredParams,
      excludeCredentials: o.excludeCredentials.map((c) => ({ type: 'public-key' as const, id: fromB64(c.id) })),
      authenticatorSelection: o.authenticatorSelection,
      attestation: o.attestation,
      timeout: o.timeout,
    },
  })) as PublicKeyCredential | null
  if (!cred) {
    throw new Error('cancelled')
  }
  const r = cred.response as AuthenticatorAttestationResponse
  return {
    id: cred.id,
    type: cred.type,
    response: { clientDataJSON: toB64(r.clientDataJSON), attestationObject: toB64(r.attestationObject) },
  }
}

// getPasskey 调用浏览器使用 Passkey 签名，返回可直接提交给服务端的 JSON。
export async function getPasskey(o: GetOptions) {
  const cred = (await navigator.credentials.get({
    publicKey: {
      challenge: fromB64(o.challenge),
      rpId: o.rpId,
      userVerification: o.userVerification,
      timeout: o.timeout,
      allowCredentials: o.allowCredentials.map((c) => ({ type: 'public-key' as const, id: fromB64(c.id) })),
    },
  })) as PublicKeyCredential | null
  if (!cred) {
    throw new Error('cancelled')
  }
  const r = cred.response as AuthenticatorAssertionResponse
  return {
    id: cred.id,
    type: cred.type,
    response: { clientDataJSON: toB64(r.clientDataJSON), authenticatorData: toB64(r.authenticatorData), signature: toB64(r.signature) },
  }
}
