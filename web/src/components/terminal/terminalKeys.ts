// 虚拟按键与修饰键处理：把按键名转换为终端输入序列。

export type KeyName = 'esc' | 'tab' | 'up' | 'down' | 'left' | 'right' | 'home' | 'end' | 'pgup' | 'pgdn'

// ctrlChar 把单个字符转换为 Ctrl 组合产生的控制字符，无法转换时原样返回。
export function ctrlChar(ch: string): string {
  if (ch.length !== 1) {
    return ch
  }
  const c = ch.toUpperCase().charCodeAt(0)
  if (c >= 64 && c <= 95) {
    return String.fromCharCode(c - 64)
  }
  if (ch === '?') {
    return '\x7f'
  }
  if (ch === ' ') {
    return '\x00'
  }
  return ch
}

const cursorFinal: Record<string, string> = { up: 'A', down: 'B', right: 'C', left: 'D', home: 'H', end: 'F' }
const tildeNum: Record<string, string> = { pgup: '5', pgdn: '6' }

// keySequence 返回按键对应的输入序列；appCursor 为终端当前是否处于应用光标模式。
export function keySequence(name: KeyName, opts: { appCursor: boolean; ctrl: boolean; alt: boolean }): string {
  // xterm 修饰键参数：1 + Alt(2) + Ctrl(4)
  const mod = 1 + (opts.alt ? 2 : 0) + (opts.ctrl ? 4 : 0)
  if (name === 'esc') {
    return '\x1b'
  }
  if (name === 'tab') {
    return opts.alt ? '\x1b\t' : '\t'
  }
  const fin = cursorFinal[name]
  if (fin) {
    if (mod > 1) {
      return `\x1b[1;${mod}${fin}`
    }
    return opts.appCursor ? `\x1bO${fin}` : `\x1b[${fin}`
  }
  const num = tildeNum[name]
  if (mod > 1) {
    return `\x1b[${num};${mod}~`
  }
  return `\x1b[${num}~`
}
