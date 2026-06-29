export const DEFAULT_THEME_KEY = 'zshell';
export const CUSTOM_THEME_KEY = 'custom';

export const THEME_COLOR_FIELDS = [
  { key: 'background', label: '背景' },
  { key: 'backgroundAlt', label: '背景层' },
  { key: 'backgroundElevated', label: '高亮背景' },
  { key: 'panel', label: '面板' },
  { key: 'line', label: '边框' },
  { key: 'primary', label: '主色' },
  { key: 'primaryAlt', label: '辅色' },
  { key: 'danger', label: '危险色' },
  { key: 'text', label: '正文' },
  { key: 'muted', label: '弱文字' },
];

const DEFAULT_COLORS = {
  background: '#06131f',
  backgroundAlt: '#0f2a3f',
  backgroundElevated: '#1a4564',
  panel: '#07141f',
  line: '#97d7ff',
  primary: '#64e9ba',
  primaryAlt: '#28c5ff',
  danger: '#ff6f7d',
  text: '#e6f2ff',
  muted: '#9bb6cd',
  terminalBackground: '#030a14',
  terminalForeground: '#e9f5ff',
  selection: '#17415b',
  black: '#030a14',
  red: '#ff6f7d',
  green: '#64e9ba',
  yellow: '#f2d479',
  blue: '#53c6ff',
  magenta: '#7bb7ff',
  cyan: '#58e5e5',
  white: '#e9f5ff',
  brightBlack: '#2b3f55',
  brightRed: '#ff95a0',
  brightGreen: '#8dffd8',
  brightYellow: '#fff0ab',
  brightBlue: '#8bdcff',
  brightMagenta: '#a2d2ff',
  brightCyan: '#8af2f2',
  brightWhite: '#ffffff',
};

const THEME_PRESETS = [
  {
    id: DEFAULT_THEME_KEY,
    name: 'zShell 默认',
    colors: DEFAULT_COLORS,
  },
  {
    id: 'dracula',
    name: 'Dracula',
    colors: {
      background: '#20212b',
      backgroundAlt: '#282a36',
      backgroundElevated: '#44475a',
      panel: '#282a36',
      line: '#6272a4',
      primary: '#ff79c6',
      primaryAlt: '#bd93f9',
      danger: '#ff5555',
      text: '#f8f8f2',
      muted: '#b8bfdc',
      terminalBackground: '#282a36',
      terminalForeground: '#f8f8f2',
      selection: '#44475a',
      black: '#21222c',
      red: '#ff5555',
      green: '#50fa7b',
      yellow: '#f1fa8c',
      blue: '#bd93f9',
      magenta: '#ff79c6',
      cyan: '#8be9fd',
      white: '#f8f8f2',
      brightBlack: '#6272a4',
      brightRed: '#ff6e6e',
      brightGreen: '#69ff94',
      brightYellow: '#ffffa5',
      brightBlue: '#d6acff',
      brightMagenta: '#ff92df',
      brightCyan: '#a4ffff',
      brightWhite: '#ffffff',
    },
  },
  {
    id: 'nord',
    name: 'Nord',
    colors: {
      background: '#2e3440',
      backgroundAlt: '#3b4252',
      backgroundElevated: '#4c566a',
      panel: '#303744',
      line: '#81a1c1',
      primary: '#88c0d0',
      primaryAlt: '#8fbcbb',
      danger: '#bf616a',
      text: '#eceff4',
      muted: '#d8dee9',
      terminalBackground: '#2e3440',
      terminalForeground: '#eceff4',
      selection: '#434c5e',
      black: '#3b4252',
      red: '#bf616a',
      green: '#a3be8c',
      yellow: '#ebcb8b',
      blue: '#81a1c1',
      magenta: '#b48ead',
      cyan: '#88c0d0',
      white: '#e5e9f0',
      brightBlack: '#4c566a',
      brightRed: '#bf616a',
      brightGreen: '#a3be8c',
      brightYellow: '#ebcb8b',
      brightBlue: '#81a1c1',
      brightMagenta: '#b48ead',
      brightCyan: '#8fbcbb',
      brightWhite: '#eceff4',
    },
  },
  {
    id: 'tokyo-night',
    name: 'Tokyo Night',
    colors: {
      background: '#1a1b26',
      backgroundAlt: '#1f2335',
      backgroundElevated: '#24283b',
      panel: '#1f2335',
      line: '#3b4261',
      primary: '#7aa2f7',
      primaryAlt: '#bb9af7',
      danger: '#f7768e',
      text: '#c0caf5',
      muted: '#a9b1d6',
      terminalBackground: '#1a1b26',
      terminalForeground: '#c0caf5',
      selection: '#33467c',
      black: '#15161e',
      red: '#f7768e',
      green: '#9ece6a',
      yellow: '#e0af68',
      blue: '#7aa2f7',
      magenta: '#bb9af7',
      cyan: '#7dcfff',
      white: '#c0caf5',
      brightBlack: '#414868',
      brightRed: '#ff899d',
      brightGreen: '#9fe044',
      brightYellow: '#faba4a',
      brightBlue: '#8db0ff',
      brightMagenta: '#c7a9ff',
      brightCyan: '#a4daff',
      brightWhite: '#ffffff',
    },
  },
  {
    id: 'catppuccin-mocha',
    name: 'Catppuccin Mocha',
    colors: {
      background: '#1e1e2e',
      backgroundAlt: '#181825',
      backgroundElevated: '#313244',
      panel: '#1e1e2e',
      line: '#45475a',
      primary: '#89b4fa',
      primaryAlt: '#cba6f7',
      danger: '#f38ba8',
      text: '#cdd6f4',
      muted: '#a6adc8',
      terminalBackground: '#1e1e2e',
      terminalForeground: '#cdd6f4',
      selection: '#45475a',
      black: '#45475a',
      red: '#f38ba8',
      green: '#a6e3a1',
      yellow: '#f9e2af',
      blue: '#89b4fa',
      magenta: '#cba6f7',
      cyan: '#94e2d5',
      white: '#bac2de',
      brightBlack: '#585b70',
      brightRed: '#f38ba8',
      brightGreen: '#a6e3a1',
      brightYellow: '#f9e2af',
      brightBlue: '#89b4fa',
      brightMagenta: '#cba6f7',
      brightCyan: '#94e2d5',
      brightWhite: '#cdd6f4',
    },
  },
  {
    id: 'gruvbox-dark',
    name: 'Gruvbox Dark',
    colors: {
      background: '#282828',
      backgroundAlt: '#1d2021',
      backgroundElevated: '#3c3836',
      panel: '#282828',
      line: '#665c54',
      primary: '#b8bb26',
      primaryAlt: '#83a598',
      danger: '#fb4934',
      text: '#ebdbb2',
      muted: '#a89984',
      terminalBackground: '#282828',
      terminalForeground: '#ebdbb2',
      selection: '#504945',
      black: '#282828',
      red: '#cc241d',
      green: '#98971a',
      yellow: '#d79921',
      blue: '#458588',
      magenta: '#b16286',
      cyan: '#689d6a',
      white: '#a89984',
      brightBlack: '#928374',
      brightRed: '#fb4934',
      brightGreen: '#b8bb26',
      brightYellow: '#fabd2f',
      brightBlue: '#83a598',
      brightMagenta: '#d3869b',
      brightCyan: '#8ec07c',
      brightWhite: '#ebdbb2',
    },
  },
  {
    id: 'one-dark',
    name: 'One Dark',
    colors: {
      background: '#282c34',
      backgroundAlt: '#21252b',
      backgroundElevated: '#3e4451',
      panel: '#282c34',
      line: '#5c6370',
      primary: '#61afef',
      primaryAlt: '#c678dd',
      danger: '#e06c75',
      text: '#abb2bf',
      muted: '#828997',
      terminalBackground: '#282c34',
      terminalForeground: '#abb2bf',
      selection: '#3e4451',
      black: '#282c34',
      red: '#e06c75',
      green: '#98c379',
      yellow: '#e5c07b',
      blue: '#61afef',
      magenta: '#c678dd',
      cyan: '#56b6c2',
      white: '#abb2bf',
      brightBlack: '#5c6370',
      brightRed: '#e06c75',
      brightGreen: '#98c379',
      brightYellow: '#e5c07b',
      brightBlue: '#61afef',
      brightMagenta: '#c678dd',
      brightCyan: '#56b6c2',
      brightWhite: '#ffffff',
    },
  },
  {
    id: 'solarized-dark',
    name: 'Solarized Dark',
    colors: {
      background: '#002b36',
      backgroundAlt: '#073642',
      backgroundElevated: '#0b3a46',
      panel: '#073642',
      line: '#586e75',
      primary: '#268bd2',
      primaryAlt: '#2aa198',
      danger: '#dc322f',
      text: '#eee8d5',
      muted: '#93a1a1',
      terminalBackground: '#002b36',
      terminalForeground: '#839496',
      selection: '#073642',
      black: '#073642',
      red: '#dc322f',
      green: '#859900',
      yellow: '#b58900',
      blue: '#268bd2',
      magenta: '#d33682',
      cyan: '#2aa198',
      white: '#eee8d5',
      brightBlack: '#586e75',
      brightRed: '#cb4b16',
      brightGreen: '#586e75',
      brightYellow: '#657b83',
      brightBlue: '#839496',
      brightMagenta: '#6c71c4',
      brightCyan: '#93a1a1',
      brightWhite: '#fdf6e3',
    },
  },
];

const THEME_BY_ID = new Map(THEME_PRESETS.map((theme) => [theme.id, theme]));

export const THEME_OPTIONS = [
  ...THEME_PRESETS.map((theme) => ({
    id: theme.id,
    name: theme.name,
    preview: previewColors(theme.colors),
  })),
  {
    id: CUSTOM_THEME_KEY,
    name: '自定义',
    preview: previewColors(DEFAULT_COLORS),
  },
];

export function createDefaultCustomTheme() {
  return THEME_COLOR_FIELDS.reduce((result, field) => {
    result[field.key] = DEFAULT_COLORS[field.key];
    return result;
  }, {});
}

export function normalizeThemeKey(value) {
  const key = String(value || '').trim().toLowerCase();
  if (key === CUSTOM_THEME_KEY || THEME_BY_ID.has(key)) {
    return key;
  }
  return DEFAULT_THEME_KEY;
}

export function normalizeCustomTheme(value) {
  const source = value && typeof value === 'object' ? value : {};
  return THEME_COLOR_FIELDS.reduce((result, field) => {
    result[field.key] = normalizeHexColor(source[field.key], DEFAULT_COLORS[field.key]);
    return result;
  }, {});
}

export function resolveTheme(themeKey, customTheme) {
  const normalizedKey = normalizeThemeKey(themeKey);
  if (normalizedKey === CUSTOM_THEME_KEY) {
    const colors = deriveCustomColors(normalizeCustomTheme(customTheme));
    return {
      id: CUSTOM_THEME_KEY,
      name: '自定义',
      colors,
      preview: previewColors(colors),
    };
  }

  const preset = THEME_BY_ID.get(normalizedKey) || THEME_BY_ID.get(DEFAULT_THEME_KEY);
  const colors = { ...DEFAULT_COLORS, ...preset.colors };
  return {
    id: preset.id,
    name: preset.name,
    colors,
    preview: previewColors(colors),
  };
}

export function applyThemeToDocument(theme) {
  if (typeof document === 'undefined') {
    return;
  }

  const colors = theme.colors;
  const root = document.documentElement;
  root.dataset.theme = theme.id;
  setHexVariable(root, '--bg-1', colors.background);
  setHexVariable(root, '--bg-2', colors.backgroundAlt);
  setHexVariable(root, '--bg-3', colors.backgroundElevated);
  setHexVariable(root, '--primary', colors.primary);
  setHexVariable(root, '--primary-2', colors.primaryAlt);
  setHexVariable(root, '--danger', colors.danger);
  setHexVariable(root, '--text', colors.text);
  setHexVariable(root, '--muted', colors.muted);
  setRgbVariable(root, '--bg-1-rgb', colors.background);
  setRgbVariable(root, '--bg-2-rgb', colors.backgroundAlt);
  setRgbVariable(root, '--bg-3-rgb', colors.backgroundElevated);
  setRgbVariable(root, '--panel-rgb', colors.panel);
  setRgbVariable(root, '--line-rgb', colors.line);
  setRgbVariable(root, '--primary-rgb', colors.primary);
  setRgbVariable(root, '--primary-2-rgb', colors.primaryAlt);
  setRgbVariable(root, '--danger-rgb', colors.danger);
  setRgbVariable(root, '--text-rgb', colors.text);
  setRgbVariable(root, '--muted-rgb', colors.muted);
  setRgbVariable(root, '--terminal-bg-rgb', colors.terminalBackground);
  setRgbVariable(root, '--terminal-fg-rgb', colors.terminalForeground);
  root.style.setProperty('--card', rgbaFromHex(colors.panel, 0.78));
  root.style.setProperty('--line', rgbaFromHex(colors.line, 0.25));
  root.style.setProperty('--app-bg-final', colors.terminalBackground || colors.background);
  root.style.setProperty('--on-primary', readableTextColor(colors.primary));
  root.style.setProperty('--modal-backdrop', rgbaFromHex(colors.background, 0.72));
  root.style.setProperty('--primary-soft', rgbaFromHex(colors.primary, 0.18));
  root.style.setProperty('--primary-2-soft', rgbaFromHex(colors.primaryAlt, 0.16));
  root.style.setProperty('--danger-soft', rgbaFromHex(colors.danger, 0.24));

  setTerminalVariables(root, colors);

  if (typeof window !== 'undefined') {
    window.dispatchEvent(new CustomEvent('zshell-theme-change', { detail: { theme } }));
  }
}

export function buildTerminalThemeFromDocument() {
  if (typeof document === 'undefined') {
    return createTerminalTheme(resolveTheme(DEFAULT_THEME_KEY, createDefaultCustomTheme()));
  }

  const styles = getComputedStyle(document.documentElement);
  return {
    background: cssValue(styles, '--terminal-bg', DEFAULT_COLORS.terminalBackground),
    foreground: cssValue(styles, '--terminal-fg', DEFAULT_COLORS.terminalForeground),
    cursor: cssValue(styles, '--terminal-cursor', DEFAULT_COLORS.primary),
    selectionBackground: cssValue(styles, '--terminal-selection', DEFAULT_COLORS.selection),
    black: cssValue(styles, '--terminal-black', DEFAULT_COLORS.black),
    red: cssValue(styles, '--terminal-red', DEFAULT_COLORS.red),
    green: cssValue(styles, '--terminal-green', DEFAULT_COLORS.green),
    yellow: cssValue(styles, '--terminal-yellow', DEFAULT_COLORS.yellow),
    blue: cssValue(styles, '--terminal-blue', DEFAULT_COLORS.blue),
    magenta: cssValue(styles, '--terminal-magenta', DEFAULT_COLORS.magenta),
    cyan: cssValue(styles, '--terminal-cyan', DEFAULT_COLORS.cyan),
    white: cssValue(styles, '--terminal-white', DEFAULT_COLORS.white),
    brightBlack: cssValue(styles, '--terminal-bright-black', DEFAULT_COLORS.brightBlack),
    brightRed: cssValue(styles, '--terminal-bright-red', DEFAULT_COLORS.brightRed),
    brightGreen: cssValue(styles, '--terminal-bright-green', DEFAULT_COLORS.brightGreen),
    brightYellow: cssValue(styles, '--terminal-bright-yellow', DEFAULT_COLORS.brightYellow),
    brightBlue: cssValue(styles, '--terminal-bright-blue', DEFAULT_COLORS.brightBlue),
    brightMagenta: cssValue(styles, '--terminal-bright-magenta', DEFAULT_COLORS.brightMagenta),
    brightCyan: cssValue(styles, '--terminal-bright-cyan', DEFAULT_COLORS.brightCyan),
    brightWhite: cssValue(styles, '--terminal-bright-white', DEFAULT_COLORS.brightWhite),
  };
}

export function createTerminalTheme(theme) {
  const colors = theme.colors;
  return {
    background: colors.terminalBackground,
    foreground: colors.terminalForeground,
    cursor: colors.primary,
    selectionBackground: colors.selection,
    black: colors.black,
    red: colors.red,
    green: colors.green,
    yellow: colors.yellow,
    blue: colors.blue,
    magenta: colors.magenta,
    cyan: colors.cyan,
    white: colors.white,
    brightBlack: colors.brightBlack,
    brightRed: colors.brightRed,
    brightGreen: colors.brightGreen,
    brightYellow: colors.brightYellow,
    brightBlue: colors.brightBlue,
    brightMagenta: colors.brightMagenta,
    brightCyan: colors.brightCyan,
    brightWhite: colors.brightWhite,
  };
}

export function buildMonacoThemeFromDocument() {
  if (typeof document === 'undefined') {
    return createMonacoTheme(resolveTheme(DEFAULT_THEME_KEY, createDefaultCustomTheme()));
  }

  const styles = getComputedStyle(document.documentElement);
  const colors = {
    background: cssValue(styles, '--terminal-bg', DEFAULT_COLORS.terminalBackground),
    foreground: cssValue(styles, '--terminal-fg', DEFAULT_COLORS.terminalForeground),
    lineHighlight: cssValue(styles, '--bg-2', DEFAULT_COLORS.backgroundAlt),
    selection: cssValue(styles, '--terminal-selection', DEFAULT_COLORS.selection),
    cursor: cssValue(styles, '--terminal-cursor', DEFAULT_COLORS.primary),
    lineNumber: cssValue(styles, '--muted', DEFAULT_COLORS.muted),
    activeLineNumber: cssValue(styles, '--primary-2', DEFAULT_COLORS.primaryAlt),
    inputBackground: cssValue(styles, '--bg-1', DEFAULT_COLORS.background),
    hoverBackground: cssValue(styles, '--bg-3', DEFAULT_COLORS.backgroundElevated),
    comment: cssValue(styles, '--muted', DEFAULT_COLORS.muted),
    keyword: cssValue(styles, '--primary', DEFAULT_COLORS.primary),
    number: cssValue(styles, '--terminal-yellow', DEFAULT_COLORS.yellow),
    string: cssValue(styles, '--primary-2', DEFAULT_COLORS.primaryAlt),
    type: cssValue(styles, '--terminal-magenta', DEFAULT_COLORS.magenta),
  };
  return createMonacoTheme({ colors });
}

export function isHexColor(value) {
  return /^#[0-9a-f]{6}$/i.test(String(value || '').trim());
}

function deriveCustomColors(customTheme) {
  return {
    ...DEFAULT_COLORS,
    ...customTheme,
    terminalBackground: customTheme.background,
    terminalForeground: customTheme.text,
    selection: customTheme.backgroundElevated,
    black: customTheme.background,
    red: customTheme.danger,
    green: customTheme.primary,
    blue: customTheme.primaryAlt,
    magenta: customTheme.primaryAlt,
    cyan: customTheme.primary,
    white: customTheme.text,
    brightBlack: customTheme.line,
    brightRed: customTheme.danger,
    brightGreen: customTheme.primary,
    brightBlue: customTheme.primaryAlt,
    brightMagenta: customTheme.primaryAlt,
    brightCyan: customTheme.primary,
    brightWhite: customTheme.text,
  };
}

function createMonacoTheme(theme) {
  const colors = theme.colors;
  return {
    base: 'vs-dark',
    inherit: true,
    rules: [
      { token: 'comment', foreground: stripHash(colors.comment || colors.muted) },
      { token: 'keyword', foreground: stripHash(colors.keyword || colors.primary) },
      { token: 'number', foreground: stripHash(colors.number || colors.yellow) },
      { token: 'string', foreground: stripHash(colors.string || colors.primaryAlt) },
      { token: 'type', foreground: stripHash(colors.type || colors.magenta) },
    ],
    colors: {
      'editor.background': colors.background || colors.terminalBackground,
      'editor.foreground': colors.foreground || colors.terminalForeground,
      'editor.lineHighlightBackground': colors.lineHighlight || colors.backgroundAlt,
      'editor.selectionBackground': colors.selection,
      'editorCursor.foreground': colors.cursor || colors.primary,
      'editorLineNumber.foreground': colors.lineNumber || colors.muted,
      'editorLineNumber.activeForeground': colors.activeLineNumber || colors.primaryAlt,
      'input.background': colors.inputBackground || colors.background,
      'input.foreground': colors.foreground || colors.terminalForeground,
      'list.hoverBackground': colors.hoverBackground || colors.backgroundElevated,
      'widget.shadow': '#00000066',
    },
  };
}

function setTerminalVariables(root, colors) {
  const terminalVariables = {
    '--terminal-bg': colors.terminalBackground,
    '--terminal-fg': colors.terminalForeground,
    '--terminal-cursor': colors.primary,
    '--terminal-selection': colors.selection,
    '--terminal-black': colors.black,
    '--terminal-red': colors.red,
    '--terminal-green': colors.green,
    '--terminal-yellow': colors.yellow,
    '--terminal-blue': colors.blue,
    '--terminal-magenta': colors.magenta,
    '--terminal-cyan': colors.cyan,
    '--terminal-white': colors.white,
    '--terminal-bright-black': colors.brightBlack,
    '--terminal-bright-red': colors.brightRed,
    '--terminal-bright-green': colors.brightGreen,
    '--terminal-bright-yellow': colors.brightYellow,
    '--terminal-bright-blue': colors.brightBlue,
    '--terminal-bright-magenta': colors.brightMagenta,
    '--terminal-bright-cyan': colors.brightCyan,
    '--terminal-bright-white': colors.brightWhite,
  };
  for (const [name, value] of Object.entries(terminalVariables)) {
    root.style.setProperty(name, value);
  }
}

function previewColors(colors) {
  return [colors.background, colors.panel, colors.primary, colors.primaryAlt, colors.danger];
}

function normalizeHexColor(value, fallback) {
  const raw = String(value || '').trim();
  if (/^#[0-9a-f]{3}$/i.test(raw)) {
    const [, r, g, b] = raw;
    return `#${r}${r}${g}${g}${b}${b}`.toLowerCase();
  }
  if (isHexColor(raw)) {
    return raw.toLowerCase();
  }
  return fallback;
}

function setHexVariable(root, name, value) {
  root.style.setProperty(name, value);
}

function setRgbVariable(root, name, value) {
  const rgb = hexToRgb(value);
  root.style.setProperty(name, `${rgb.r}, ${rgb.g}, ${rgb.b}`);
}

function rgbaFromHex(value, alpha) {
  const rgb = hexToRgb(value);
  return `rgba(${rgb.r}, ${rgb.g}, ${rgb.b}, ${alpha})`;
}

function hexToRgb(value) {
  const hex = normalizeHexColor(value, DEFAULT_COLORS.background).slice(1);
  return {
    r: Number.parseInt(hex.slice(0, 2), 16),
    g: Number.parseInt(hex.slice(2, 4), 16),
    b: Number.parseInt(hex.slice(4, 6), 16),
  };
}

function readableTextColor(value) {
  const rgb = hexToRgb(value);
  const channels = [rgb.r, rgb.g, rgb.b].map((channel) => {
    const normalized = channel / 255;
    return normalized <= 0.03928 ? normalized / 12.92 : ((normalized + 0.055) / 1.055) ** 2.4;
  });
  const luminance = 0.2126 * channels[0] + 0.7152 * channels[1] + 0.0722 * channels[2];
  return luminance > 0.56 ? '#06131f' : '#f8fbff';
}

function cssValue(styles, name, fallback) {
  const value = styles.getPropertyValue(name).trim();
  return value || fallback;
}

function stripHash(value) {
  return String(value || '').replace(/^#/, '');
}
