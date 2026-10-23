// 终端配色主题。
import type { ITheme } from '@xterm/xterm'
import { t } from '../../i18n'
import { mix, type ThemeDef } from '../../themes'
import { useTheme } from '../../composables/useTheme'

const { current: uiTheme } = useTheme()

export interface TermTheme {
  name: string
  // label 为产品名等无需翻译的名称；需要翻译的主题改用 labelKey
  label?: string
  labelKey?: string
  theme: ITheme
}

export const termThemes: TermTheme[] = [
  // 跟随界面主题：底色、文字、光标取自当前界面配色，ANSI 色按亮暗沿用默认深色或 GitHub 浅色；用 getter 让界面换肤后取到最新值
  {
    name: 'follow',
    labelKey: 'terminal.themes.follow',
    get theme() {
      return followTheme(uiTheme.value)
    },
  },
  {
    name: 'default',
    labelKey: 'terminal.themes.default',
    theme: {
      background: '#1d1f21', foreground: '#c5c8c6', cursor: '#c5c8c6', selectionBackground: '#373b41',
      black: '#282a2e', red: '#cc6666', green: '#b5bd68', yellow: '#f0c674', blue: '#81a2be', magenta: '#b294bb', cyan: '#8abeb7', white: '#c5c8c6',
      brightBlack: '#666666', brightRed: '#d54e53', brightGreen: '#b9ca4a', brightYellow: '#e7c547', brightBlue: '#7aa6da', brightMagenta: '#c397d8', brightCyan: '#70c0b1', brightWhite: '#eaeaea',
    },
  },
  {
    name: 'dracula',
    label: 'Dracula',
    theme: {
      background: '#282a36', foreground: '#f8f8f2', cursor: '#f8f8f2', selectionBackground: '#44475a',
      black: '#21222c', red: '#ff5555', green: '#50fa7b', yellow: '#f1fa8c', blue: '#bd93f9', magenta: '#ff79c6', cyan: '#8be9fd', white: '#f8f8f2',
      brightBlack: '#6272a4', brightRed: '#ff6e6e', brightGreen: '#69ff94', brightYellow: '#ffffa5', brightBlue: '#d6acff', brightMagenta: '#ff92df', brightCyan: '#a4ffff', brightWhite: '#ffffff',
    },
  },
  {
    name: 'monokai',
    label: 'Monokai',
    theme: {
      background: '#272822', foreground: '#f8f8f2', cursor: '#f8f8f0', selectionBackground: '#49483e',
      black: '#272822', red: '#f92672', green: '#a6e22e', yellow: '#f4bf75', blue: '#66d9ef', magenta: '#ae81ff', cyan: '#a1efe4', white: '#f8f8f2',
      brightBlack: '#75715e', brightRed: '#f92672', brightGreen: '#a6e22e', brightYellow: '#f4bf75', brightBlue: '#66d9ef', brightMagenta: '#ae81ff', brightCyan: '#a1efe4', brightWhite: '#f9f8f5',
    },
  },
  {
    name: 'onedark',
    label: 'One Dark',
    theme: {
      background: '#282c34', foreground: '#abb2bf', cursor: '#528bff', selectionBackground: '#3e4451',
      black: '#282c34', red: '#e06c75', green: '#98c379', yellow: '#e5c07b', blue: '#61afef', magenta: '#c678dd', cyan: '#56b6c2', white: '#abb2bf',
      brightBlack: '#5c6370', brightRed: '#e06c75', brightGreen: '#98c379', brightYellow: '#e5c07b', brightBlue: '#61afef', brightMagenta: '#c678dd', brightCyan: '#56b6c2', brightWhite: '#ffffff',
    },
  },
  {
    name: 'nord',
    label: 'Nord',
    theme: {
      background: '#2e3440', foreground: '#d8dee9', cursor: '#d8dee9', selectionBackground: '#434c5e',
      black: '#3b4252', red: '#bf616a', green: '#a3be8c', yellow: '#ebcb8b', blue: '#81a1c1', magenta: '#b48ead', cyan: '#88c0d0', white: '#e5e9f0',
      brightBlack: '#4c566a', brightRed: '#bf616a', brightGreen: '#a3be8c', brightYellow: '#ebcb8b', brightBlue: '#81a1c1', brightMagenta: '#b48ead', brightCyan: '#8fbcbb', brightWhite: '#eceff4',
    },
  },
  {
    name: 'solarized-dark',
    label: 'Solarized Dark',
    theme: {
      background: '#002b36', foreground: '#839496', cursor: '#93a1a1', selectionBackground: '#073642',
      black: '#073642', red: '#dc322f', green: '#859900', yellow: '#b58900', blue: '#268bd2', magenta: '#d33682', cyan: '#2aa198', white: '#eee8d5',
      brightBlack: '#002b36', brightRed: '#cb4b16', brightGreen: '#586e75', brightYellow: '#657b83', brightBlue: '#839496', brightMagenta: '#6c71c4', brightCyan: '#93a1a1', brightWhite: '#fdf6e3',
    },
  },
  {
    name: 'solarized-light',
    label: 'Solarized Light',
    theme: {
      background: '#fdf6e3', foreground: '#586e75', cursor: '#586e75', selectionBackground: '#d9d2bf',
      black: '#073642', red: '#dc322f', green: '#859900', yellow: '#b58900', blue: '#268bd2', magenta: '#d33682', cyan: '#2aa198', white: '#eee8d5',
      brightBlack: '#002b36', brightRed: '#cb4b16', brightGreen: '#586e75', brightYellow: '#657b83', brightBlue: '#839496', brightMagenta: '#6c71c4', brightCyan: '#93a1a1', brightWhite: '#fdf6e3',
    },
  },
  {
    name: 'github-light',
    labelKey: 'terminal.themes.githubLight',
    theme: {
      background: '#ffffff', foreground: '#24292f', cursor: '#0969da', selectionBackground: '#b6d6f7',
      black: '#24292f', red: '#cf222e', green: '#116329', yellow: '#9a6700', blue: '#0550ae', magenta: '#8250df', cyan: '#1b7c83', white: '#6e7781',
      brightBlack: '#57606a', brightRed: '#a40e26', brightGreen: '#1a7f37', brightYellow: '#633c01', brightBlue: '#218bff', brightMagenta: '#a475f9', brightCyan: '#3192aa', brightWhite: '#8c959f',
    },
  },
]

// followTheme 由界面主题生成终端配色。
function followTheme(ui: ThemeDef): ITheme {
  const base = findTheme(ui.dark ? 'default' : 'github-light').theme
  return {
    ...base,
    background: ui.card,
    foreground: ui.text,
    cursor: ui.primary,
    cursorAccent: ui.card,
    selectionBackground: mix(ui.primary, ui.card, ui.dark ? 0.6 : 0.75),
  }
}

export function findTheme(name: string): TermTheme {
  return termThemes.find((x) => x.name === name) ?? termThemes[0]
}

// themeLabel 返回主题的显示名称（按当前语言翻译）。
export function themeLabel(th: TermTheme): string {
  return th.labelKey ? t(th.labelKey) : (th.label ?? th.name)
}
