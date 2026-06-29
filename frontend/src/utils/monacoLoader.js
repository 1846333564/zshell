import { buildMonacoThemeFromDocument } from '../theme';

let monacoPromise = null;
let monacoConfigured = false;

export const MONACO_THEME = 'zshell-dark';

const EXTENSION_LANGUAGE = {
  bat: 'bat',
  c: 'cpp',
  cc: 'cpp',
  conf: 'ini',
  cpp: 'cpp',
  cs: 'csharp',
  css: 'css',
  dockerfile: 'dockerfile',
  go: 'go',
  h: 'cpp',
  hpp: 'cpp',
  htm: 'html',
  html: 'html',
  ini: 'ini',
  java: 'java',
  js: 'javascript',
  json: 'json',
  jsx: 'javascript',
  less: 'less',
  log: 'plaintext',
  lua: 'lua',
  md: 'markdown',
  mjs: 'javascript',
  mysql: 'mysql',
  php: 'php',
  properties: 'ini',
  ps1: 'powershell',
  py: 'python',
  rb: 'ruby',
  rs: 'rust',
  scss: 'scss',
  sh: 'shell',
  sql: 'sql',
  ts: 'typescript',
  tsx: 'typescript',
  txt: 'plaintext',
  vue: 'html',
  xml: 'xml',
  yaml: 'yaml',
  yml: 'yaml',
};

const FILENAME_LANGUAGE = {
  dockerfile: 'dockerfile',
  makefile: 'shell',
};

export function preloadMonaco() {
  return loadMonaco().then(() => undefined);
}

export function loadMonaco() {
  if (!monacoPromise) {
    monacoPromise = initializeMonaco().catch((error) => {
      monacoPromise = null;
      throw error;
    });
  }
  return monacoPromise;
}

export function detectMonacoLanguage(monaco, filePath = '') {
  const rawName = String(filePath || '').split('/').filter(Boolean).at(-1) || '';
  const lowerName = rawName.toLowerCase();
  const ext = lowerName.includes('.') ? lowerName.split('.').at(-1) : '';
  const candidate = FILENAME_LANGUAGE[lowerName] || EXTENSION_LANGUAGE[ext] || 'plaintext';
  const exists = monaco.languages.getLanguages().some((language) => language.id === candidate);
  return exists ? candidate : 'plaintext';
}

async function initializeMonaco() {
  const workerConstructors = await loadWorkers();
  self.MonacoEnvironment = {
    getWorker(_workerId, label) {
      if (label === 'json') {
        return new workerConstructors.JsonWorker();
      }
      if (label === 'css' || label === 'scss' || label === 'less') {
        return new workerConstructors.CssWorker();
      }
      if (label === 'html' || label === 'handlebars' || label === 'razor') {
        return new workerConstructors.HtmlWorker();
      }
      if (label === 'typescript' || label === 'javascript') {
        return new workerConstructors.TsWorker();
      }
      return new workerConstructors.EditorWorker();
    },
  };

  const monaco = await import('monaco-editor');
  configureMonaco(monaco);
  return monaco;
}

async function loadWorkers() {
  const [editor, json, css, html, ts] = await Promise.all([
    import('monaco-editor/esm/vs/editor/editor.worker?worker'),
    import('monaco-editor/esm/vs/language/json/json.worker?worker'),
    import('monaco-editor/esm/vs/language/css/css.worker?worker'),
    import('monaco-editor/esm/vs/language/html/html.worker?worker'),
    import('monaco-editor/esm/vs/language/typescript/ts.worker?worker'),
  ]);

  return {
    EditorWorker: editor.default,
    JsonWorker: json.default,
    CssWorker: css.default,
    HtmlWorker: html.default,
    TsWorker: ts.default,
  };
}

function configureMonaco(monaco) {
  applyMonacoTheme(monaco);
  if (monacoConfigured) {
    return;
  }
  monacoConfigured = true;

  const diagnostics = {
    noSemanticValidation: true,
    noSyntaxValidation: false,
    noSuggestionDiagnostics: true,
  };
  monaco.languages.typescript?.javascriptDefaults?.setDiagnosticsOptions(diagnostics);
  monaco.languages.typescript?.typescriptDefaults?.setDiagnosticsOptions(diagnostics);
  monaco.languages.typescript?.javascriptDefaults?.setEagerModelSync(false);
  monaco.languages.typescript?.typescriptDefaults?.setEagerModelSync(false);
}

export function applyMonacoTheme(monaco) {
  monaco.editor.defineTheme(MONACO_THEME, buildMonacoThemeFromDocument());
  monaco.editor.setTheme(MONACO_THEME);
}
