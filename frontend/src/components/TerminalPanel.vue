<template>
  <div class="terminal-instance">
    <div class="terminal-meta">
      <span>{{ tabTitle }} · {{ connectionName }}</span>
      <span>{{ online ? '在线' : '离线' }}</span>
    </div>

    <div class="terminal-wrap">
      <div ref="terminalMount" style="width: 100%; height: 100%"></div>
    </div>
  </div>
</template>

<script setup>
import { nextTick, onBeforeUnmount, onMounted, ref, watch } from 'vue';
import { Terminal } from 'xterm';
import { FitAddon } from 'xterm-addon-fit';
import 'xterm/css/xterm.css';
import { createTerminalClient } from '../services/wsClient';

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
});
const terminalMount = ref(null);

let term;
let fitAddon;
let wsClient;
let resizeObserver;

const online = ref(false);

onMounted(async () => {
  term = new Terminal({
    cursorBlink: true,
    convertEol: true,
    fontSize: 14,
    lineHeight: 1.3,
    theme: {
      background: '#030a14',
      foreground: '#e9f5ff',
      cursor: '#64e9ba',
      selectionBackground: '#17415b',
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
    },
  });

  fitAddon = new FitAddon();
  term.loadAddon(fitAddon);
  term.open(terminalMount.value);

  await nextTick();
  fitAddon.fit();

  term.writeln(`${props.tabTitle} 就绪，正在连接 ${props.connectionName} ...`);
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
});

onBeforeUnmount(() => {
  disconnect();
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

async function connect() {
  disconnect();
  term.writeln(`\r\n[connecting] ${props.connectionName}`);

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
</script>
