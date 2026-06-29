<template>
  <div class="terminal-instance" @click="hideTerminalMenu" @contextmenu.prevent.stop="openTerminalMenu">
    <div class="terminal-wrap">
      <div ref="terminalMount" style="width: 100%; height: 100%"></div>
    </div>

    <Teleport to="body">
      <div
        v-if="terminalMenu.visible"
        class="context-menu terminal-context-menu"
        :style="{ left: `${terminalMenu.x}px`, top: `${terminalMenu.y}px` }"
        @click.stop
        @contextmenu.prevent.stop
      >
        <button :disabled="!terminalMenu.hasSelection" @click="copyFromTerminalMenu">复制</button>
        <button :disabled="!online" @click="pasteFromTerminalMenu">粘贴</button>
        <div class="context-menu-separator"></div>
        <button @click="clearFromTerminalMenu">清屏</button>
        <button @click="reconnectFromTerminalMenu">重新连接</button>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { nextTick, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { createTerminalClient } from '../services/wsClient';
import { buildTerminalThemeFromDocument } from '../theme';
import { viewportContextMenuPosition } from '../utils/contextMenuPosition';

const props = defineProps({
	connectionId: {
		type: String,
		required: true,
	},
  connectionName: {
    type: String,
    default: '未连接',
  },
	tabTitle: {
		type: String,
		default: '终端',
	},
	active: {
		type: Boolean,
		default: false,
	},
  terminalFontSize: {
    type: Number,
    default: 14,
  },
});
const emit = defineEmits(['terminal-font-size-change']);
const terminalMount = ref(null);
const terminalMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  hasSelection: false,
});

let term;
let fitAddon;
let wsClient;
let resizeObserver;
let themeChangeHandler;
let currentTerminalFontSize = normalizeTerminalFontSize(props.terminalFontSize);

const online = ref(false);

onMounted(async () => {
  term = new Terminal({
    cursorBlink: true,
    convertEol: true,
    fontSize: currentTerminalFontSize,
    lineHeight: 1.3,
    theme: buildTerminalThemeFromDocument(),
  });
  term.attachCustomKeyEventHandler(handleTerminalShortcut);

  fitAddon = new FitAddon();
  term.loadAddon(fitAddon);
  term.open(terminalMount.value);

  await nextTick();
  fitAddon.fit();

  term.writeln('正在连接...');
  await connect();

  term.onData((data) => {
    if (!online.value || !wsClient) {
      return;
    }

    // PTY mode should receive raw keystrokes directly.
    wsClient.sendInput(data);
  });

  resizeObserver = new ResizeObserver(() => {
    if (!fitAddon || !term) {
      return;
    }

    fitAddon.fit();
    if (online.value && wsClient) {
      wsClient.sendResize(term.cols, term.rows);
    }
  });

  resizeObserver.observe(terminalMount.value);
  themeChangeHandler = () => applyTerminalTheme();
  window.addEventListener('zshell-theme-change', themeChangeHandler);
});

onBeforeUnmount(() => {
  disconnect();
  if (themeChangeHandler) {
    window.removeEventListener('zshell-theme-change', themeChangeHandler);
  }
  resizeObserver?.disconnect();
  term?.dispose();
});

watch(
  () => props.connectionId,
  async (value, prev) => {
    if (!value || value === prev || !term) {
      return;
    }
    await connect();
  },
);

watch(
  () => props.active,
  async (isActive) => {
    if (!isActive || !fitAddon || !term) {
      return;
    }

    await nextTick();
    fitAddon.fit();
    if (online.value && wsClient) {
      wsClient.sendResize(term.cols, term.rows);
    }
  },
);

watch(
  () => props.terminalFontSize,
  (value) => {
    const next = normalizeTerminalFontSize(value);
    if (next === currentTerminalFontSize) {
      return;
    }
    currentTerminalFontSize = next;
    applyTerminalFontSize();
  },
);

async function connect() {
  disconnect();
  term.writeln('\r\n[connecting]');

  wsClient = createTerminalClient({
    connectionId: props.connectionId,
    onOpen: () => {
      online.value = true;
      term.writeln('\r\n[connected] 已连接，开始输入命令');
      if (fitAddon) {
        fitAddon.fit();
      }
      wsClient.sendResize(term.cols, term.rows);
    },
    onClose: () => {
      online.value = false;
      term.writeln('\r\n[disconnected] 连接已关闭');
    },
    onOutput: (text) => {
      term.write(text);
    },
    onError: (code, message) => {
      term.writeln(`\r\n[error] ${code}: ${message}`);
    },
  });

  await wsClient.waitUntilOpen();
}

function disconnect() {
  if (wsClient) {
    wsClient.close();
    wsClient = null;
  }
  online.value = false;
}

function handleTerminalShortcut(event) {
  if (event.type !== 'keydown' || !event.ctrlKey) {
    return true;
  }

  const key = event.key.toLowerCase();
  if (event.shiftKey && key === 'c') {
    copyTerminalSelection();
    event.preventDefault();
    return false;
  }

  if (event.shiftKey && key === 'v') {
    pasteClipboardToTerminal();
    event.preventDefault();
    return false;
  }

  if (key === '+' || key === '=') {
    adjustTerminalFontSize(1);
    event.preventDefault();
    return false;
  }

  if (key === '-' || key === '_') {
    adjustTerminalFontSize(-1);
    event.preventDefault();
    return false;
  }

  if (key === '0') {
    setTerminalFontSize(14);
    event.preventDefault();
    return false;
  }

  return true;
}

function adjustTerminalFontSize(delta) {
  setTerminalFontSize(currentTerminalFontSize + delta);
}

function setTerminalFontSize(value) {
  const next = normalizeTerminalFontSize(value);
  if (next === currentTerminalFontSize) {
    return;
  }
  currentTerminalFontSize = next;
  applyTerminalFontSize();
  emit('terminal-font-size-change', currentTerminalFontSize);
}

function applyTerminalFontSize() {
  if (!term || !fitAddon) {
    return;
  }
  term.options.fontSize = currentTerminalFontSize;
  fitAddon.fit();
  if (online.value && wsClient) {
    wsClient.sendResize(term.cols, term.rows);
  }
}

function applyTerminalTheme() {
  if (!term) {
    return;
  }
  term.options.theme = buildTerminalThemeFromDocument();
}

function normalizeTerminalFontSize(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 14;
  }
  return Math.min(28, Math.max(10, Math.round(parsed)));
}

function copyTerminalSelection() {
  const text = term?.getSelection() || '';
  if (!text) {
    return;
  }
  navigator.clipboard?.writeText(text).catch(() => {});
}

function pasteClipboardToTerminal() {
  if (!online.value || !wsClient || !navigator.clipboard?.readText) {
    return;
  }
  navigator.clipboard.readText().then((text) => {
    if (text && online.value && wsClient) {
      wsClient.sendInput(text);
    }
  }).catch(() => {});
}

function openTerminalMenu(event) {
  const position = viewportContextMenuPosition(event, { width: 160, height: 176 });
  terminalMenu.visible = true;
  terminalMenu.x = position.x;
  terminalMenu.y = position.y;
  terminalMenu.hasSelection = Boolean(term?.getSelection());
}

function hideTerminalMenu() {
  terminalMenu.visible = false;
}

function copyFromTerminalMenu() {
  copyTerminalSelection();
  hideTerminalMenu();
}

function pasteFromTerminalMenu() {
  pasteClipboardToTerminal();
  hideTerminalMenu();
}

function clearFromTerminalMenu() {
  term?.clear();
  hideTerminalMenu();
}

async function reconnectFromTerminalMenu() {
  hideTerminalMenu();
  await connect();
}
</script>
