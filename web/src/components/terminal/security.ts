// Web 终端的访问门禁：用户协议确认与二次验证（2FA）。
// 后端在协议未接受或 2FA 未验证时返回 403，并带 needAgreement / need2fa 标记；
// 这里统一弹出对应对话框，用户完成后自动重试原请求。
import { reactive } from 'vue'
import { ApiError, api } from '../../api'
import { t } from '../../i18n'
import type { SecurityState } from './types'

// CancelError 表示用户主动取消了协议确认或验证码输入。
export class CancelError extends Error {
  constructor() {
    super(t('terminal.security.cancelled'))
  }
}

export function isCancel(e: unknown): boolean {
  return e instanceof CancelError
}

// gate 控制全局唯一的 SecurityGate 组件显示哪个对话框。
export const gate = reactive({ agreement: false, twofa: false })

interface Waiter {
  promise: Promise<void>
  resolve: () => void
  reject: (e: Error) => void
}

function makeWaiter(): Waiter {
  let resolve!: () => void
  let reject!: (e: Error) => void
  const promise = new Promise<void>((res, rej) => {
    resolve = res
    reject = rej
  })
  return { promise, resolve, reject }
}

let agreementWaiter: Waiter | undefined
let twofaWaiter: Waiter | undefined

function requestAgreement(): Promise<void> {
  if (!agreementWaiter) {
    agreementWaiter = makeWaiter()
    gate.agreement = true
  }
  return agreementWaiter.promise
}

function requestTwoFA(): Promise<void> {
  if (!twofaWaiter) {
    twofaWaiter = makeWaiter()
    gate.twofa = true
  }
  return twofaWaiter.promise
}

// settleAgreement 由 SecurityGate 调用：ok 为 true 表示已接受协议，否则视为取消。
export function settleAgreement(ok: boolean) {
  const w = agreementWaiter
  agreementWaiter = undefined
  gate.agreement = false
  if (ok) {
    w?.resolve()
  } else {
    w?.reject(new CancelError())
  }
}

// settleTwoFA 由 SecurityGate 调用：ok 为 true 表示验证通过，否则视为取消。
export function settleTwoFA(ok: boolean) {
  const w = twofaWaiter
  twofaWaiter = undefined
  gate.twofa = false
  if (ok) {
    w?.resolve()
  } else {
    w?.reject(new CancelError())
  }
}

// withAccess 执行请求；遇到门禁 403 时弹出对话框，通过后重试，最多重试 3 次。
export async function withAccess<T>(fn: () => Promise<T>): Promise<T> {
  for (let i = 0; ; i++) {
    try {
      return await fn()
    } catch (e) {
      if (i >= 3 || !(e instanceof ApiError) || e.status !== 403) {
        throw e
      }
      if (e.data?.needAgreement) {
        await requestAgreement()
      } else if (e.data?.need2fa) {
        await requestTwoFA()
      } else {
        throw e
      }
    }
  }
}

// ensureAccess 在建立 WebSocket 之前预检门禁（WebSocket 握手失败拿不到 403 详情）。
export function ensureAccess(): Promise<void> {
  return withAccess(async () => {
    const s = await api.get<SecurityState>('/api/webterminal/security')
    if (!s.agreementAccepted) {
      throw new ApiError(t('terminal.security.needAgreement'), 403, { needAgreement: true })
    }
    if (s.twoFAEnable && !s.verified) {
      throw new ApiError(t('terminal.security.needTwofa'), 403, { need2fa: true })
    }
  })
}

// agreementText 返回当前语言的用户协议全文（在函数内取值，切换语言后即时生效）。
export function agreementText(): string {
  return t('terminal.agreementText')
}
