<template>
  <main class="app-shell">
    <section v-if="!session.connected" class="connect-page">
      <div class="connect-card panel">
        <h1>zShell Lite</h1>
        <p>轻量级本地 SSH / SFTP 工具</p>

        <div class="connect-actions">
          <button class="small-btn" @click="startNewConnection">新建连接</button>
        </div>

        <section class="history-panel">
          <div class="history-head">
            <h3>历史连接</h3>
            <span>{{ historyConnections.length }} 条</span>
          </div>

          <div v-if="historyConnections.length === 0" class="empty-tip">
            暂无历史连接，点击“新建连接”开始。
          </div>

          <div v-else class="history-list">
            <article
              v-for="item in historyConnections"
              :key="item.id"
              class="history-item"
            >
              <div class="history-meta">
                <strong>{{ item.name }}</strong>
                <span>{{ item.host }}:{{ item.port }} · {{ item.username }}</span>
              </div>
              <div class="history-actions">
                <button class="mini-btn" @click="fillFromHistory(item)">填充</button>
                <button class="mini-btn danger" @click="removeHistory(item.id)">删除</button>
              </div>
            </article>
          </div>
        </section>

        <ConnectionForm
          :busy="busy"
          :error="connectError"
          :initial-value="draftConnection"
          @connect="handleConnect"
        />
      </div>
    </section>

    <section v-else class="workspace-page">
      <header class="workspace-header panel">
        <div>
          <h2>{{ session.connectionName }}</h2>
          <p>{{ session.host }}:{{ session.port }} · {{ session.username }}</p>
        </div>
        <div class="header-actions">
          <button class="small-btn" @click="disconnectToNewConnection">新建连接</button>
          <button class="small-btn danger" @click="disconnectWorkspace">断开连接</button>
        </div>
      </header>

      <section class="workspace-body">
        <aside class="workspace-left panel">
          <h3>监控 / 信息区（预留）</h3>
          <p>后续可放置 CPU、内存、网络、告警等信息。</p>
          <ul>
            <li>连接名: {{ session.connectionName }}</li>
            <li>服务器: {{ session.host }}</li>
            <li>端口: {{ session.port }}</li>
            <li>用户: {{ session.username }}</li>
          </ul>
        </aside>

        <section class="workspace-right" ref="rightPaneRef" :class="{ dragging: isDragging }">
          <div class="panel console-panel" :style="{ height: `calc(${consoleHeightPercent}% - 5px)` }">
            <TerminalTabs
              :connection-id="session.connectionId"
              :connection-name="session.connectionName"
            />
          </div>

          <div
            class="splitter"
            title="拖拽调整控制台高度"
            @mousedown.prevent="startDrag"
          ></div>

          <div class="panel file-panel workspace-files">
            <FileManager :connection-id="session.connectionId" />
          </div>
        </section>
      </section>
    </section>
  </main>
</template>

<script setup>
import { onBeforeUnmount, onMounted, reactive, ref } from 'vue';
import ConnectionForm from './components/ConnectionForm.vue';
import FileManager from './components/FileManager.vue';
import TerminalTabs from './components/TerminalTabs.vue';
import { createConnection, testConnection } from './services/apiClient';

const CONNECTION_HISTORY_KEY = 'zshell.connection.history.v1';

const busy = ref(false);
const connectError = ref('');
const rightPaneRef = ref(null);
const consoleHeightPercent = ref(62);
const isDragging = ref(false);
const historyConnections = ref([]);
const draftConnection = ref(defaultConnectionDraft());

const session = reactive({
  connected: false,
  connectionId: '',
  connectionName: '',
  host: '',
  port: 22,
  username: '',
});

async function handleConnect(payload) {
  busy.value = true;
  connectError.value = '';

  saveHistoryEntry(payload);

  try {
    const created = await createConnection(payload);
    const connectionId = created.connection.id;

    await testConnection(connectionId);

    session.connected = true;
    session.connectionId = connectionId;
    session.connectionName = payload.name;
    session.host = payload.host;
    session.port = payload.port;
    session.username = payload.username;

    consoleHeightPercent.value = 62;
  } catch (error) {
    connectError.value = error instanceof Error ? error.message : '连接失败';
    disconnectWorkspace();
  } finally {
    busy.value = false;
  }
}

function disconnectWorkspace() {
  session.connected = false;
  session.connectionId = '';
  session.connectionName = '';
  session.host = '';
  session.port = 22;
  session.username = '';
}

function disconnectToNewConnection() {
  disconnectWorkspace();
  startNewConnection();
}

function startNewConnection() {
  draftConnection.value = defaultConnectionDraft();
  connectError.value = '';
}

function fillFromHistory(item) {
  draftConnection.value = {
    name: item.name,
    host: item.host,
    port: item.port,
    username: item.username,
    password: item.password,
  };
  connectError.value = '';
}

function removeHistory(id) {
  historyConnections.value = historyConnections.value.filter((item) => item.id !== id);
  persistHistory();
}

function saveHistoryEntry(payload) {
  const id = `${payload.host}:${payload.port}:${payload.username}`;
  const entry = {
    id,
    name: payload.name,
    host: payload.host,
    port: Number(payload.port) || 22,
    username: payload.username,
    password: payload.password || '',
    lastUsedAt: Date.now(),
  };

  const next = historyConnections.value.filter((item) => item.id !== id);
  next.unshift(entry);
  historyConnections.value = next.slice(0, 50);
  persistHistory();
}

function persistHistory() {
  localStorage.setItem(CONNECTION_HISTORY_KEY, JSON.stringify(historyConnections.value));
}

function loadHistory() {
  try {
    const raw = localStorage.getItem(CONNECTION_HISTORY_KEY);
    if (!raw) {
      historyConnections.value = [];
      return;
    }

    const parsed = JSON.parse(raw);
    if (!Array.isArray(parsed)) {
      historyConnections.value = [];
      return;
    }

    historyConnections.value = parsed
      .filter((item) => item && item.id && item.host && item.username)
      .map((item) => ({
        id: String(item.id),
        name: String(item.name || '未命名连接'),
        host: String(item.host),
        port: Number(item.port) || 22,
        username: String(item.username),
        password: String(item.password || ''),
        lastUsedAt: Number(item.lastUsedAt) || 0,
      }))
      .sort((a, b) => b.lastUsedAt - a.lastUsedAt)
      .slice(0, 50);
  } catch {
    historyConnections.value = [];
  }
}

function defaultConnectionDraft() {
  return {
    name: '默认服务器',
    host: '127.0.0.1',
    port: 22,
    username: 'root',
    password: '',
  };
}

onMounted(() => {
  loadHistory();
});

let moveHandler = null;
let upHandler = null;

function startDrag(event) {
  event.preventDefault();

  const pane = rightPaneRef.value;
  if (!pane) {
    return;
  }

  isDragging.value = true;
  document.body.style.userSelect = 'none';

  const onMove = (event) => {
    const rect = pane.getBoundingClientRect();
    const offsetY = event.clientY - rect.top;
    const percent = (offsetY / rect.height) * 100;
    consoleHeightPercent.value = Math.min(85, Math.max(30, percent));
  };

  const onUp = () => {
    isDragging.value = false;
    document.body.style.userSelect = '';

    if (moveHandler) {
      window.removeEventListener('mousemove', moveHandler);
    }
    if (upHandler) {
      window.removeEventListener('mouseup', upHandler);
    }
    moveHandler = null;
    upHandler = null;
  };

  moveHandler = onMove;
  upHandler = onUp;
  window.addEventListener('mousemove', onMove);
  window.addEventListener('mouseup', onUp);
}

onBeforeUnmount(() => {
  isDragging.value = false;
  document.body.style.userSelect = '';

  if (moveHandler) {
    window.removeEventListener('mousemove', moveHandler);
  }
  if (upHandler) {
    window.removeEventListener('mouseup', upHandler);
  }
});
</script>
