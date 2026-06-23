<template>
  <main class="app-shell">
    <section class="desktop-layout">
      <aside class="monitor-sidebar panel">
        <MonitorPanel :session="activeSession" />
      </aside>

      <section class="main-workspace">
        <header class="connection-tabbar panel">
          <button
            v-for="item in sessions"
            :key="item.connectionId"
            class="connection-tab"
            :class="{ active: item.connectionId === activeSessionId }"
            @click="activateSession(item.connectionId)"
          >
            <span class="tab-title">{{ item.connectionName }}</span>
            <button class="tab-close" @click.stop="closeSession(item.connectionId)">x</button>
          </button>
          <button class="connection-tab add" title="新建连接" @click="showConnectHome">+</button>
        </header>

        <section v-if="!activeSession" class="connect-workspace panel">
          <div class="connect-columns">
            <section class="history-panel flat">
              <div class="history-head">
                <h3>已保存连接</h3>
                <span>{{ configLoading ? '加载中' : `${savedConnections.length} 条` }}</span>
              </div>

              <div v-if="savedConnections.length === 0" class="empty-tip">
                暂无保存的连接。
              </div>

              <div v-else class="history-list">
                <article
                  v-for="item in savedConnections"
                  :key="item.id"
                  class="history-item"
                  :class="{ editing: item.id === editingConnectionId }"
                >
                  <div class="history-meta">
                    <strong>{{ item.name }}</strong>
                    <span>{{ item.host }}:{{ item.port }} · {{ item.username }} · {{ authLabel(item.authMethod) }}</span>
                  </div>
                  <div class="history-actions">
                    <button class="mini-btn" :disabled="busy" @click="connectFromSaved(item)">连接</button>
                    <button class="mini-btn" :disabled="busy" @click="editSavedConnection(item)">编辑</button>
                    <button class="mini-btn danger" :disabled="busy" @click="removeSavedConnection(item)">删除</button>
                  </div>
                </article>
              </div>
            </section>

            <section class="connect-form-pane">
              <ConnectionForm
                :busy="busy"
                :error="connectError || configError"
                :initial-value="draftConnection"
                :mode="editingConnectionId ? 'edit' : 'create'"
                :submit-label="editingConnectionId ? '保存并连接' : '保存并连接'"
                :title="editingConnectionId ? '编辑连接' : '连接配置'"
                @connect="handleConnect"
              />
            </section>
          </div>
        </section>

        <section v-else class="active-workspace">
          <div class="terminal-band panel" :style="{ height: `calc(${consoleHeightPercent}% - 5px)` }">
            <TerminalTabs
              :key="activeSession.connectionId"
              :connection-id="activeSession.connectionId"
              :connection-name="activeSession.connectionName"
            />
          </div>

          <div class="splitter" title="拖拽调整控制台高度" @mousedown.prevent="startDrag"></div>

          <div class="file-band panel">
            <FileManager :key="activeSession.connectionId" :connection-id="activeSession.connectionId" />
          </div>
        </section>
      </section>
    </section>
  </main>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import ConnectionForm from './components/ConnectionForm.vue';
import FileManager from './components/FileManager.vue';
import MonitorPanel from './components/MonitorPanel.vue';
import TerminalTabs from './components/TerminalTabs.vue';
import {
  deleteConnectionConfig,
  listConnectionConfigs,
  saveConnectionConfig,
  testConnection,
  updateConnectionConfig,
} from './services/apiClient';

const busy = ref(false);
const configLoading = ref(false);
const configError = ref('');
const connectError = ref('');
const consoleHeightPercent = ref(58);
const isDragging = ref(false);
const savedConnections = ref([]);
const draftConnection = ref(defaultConnectionDraft());
const editingConnectionId = ref('');
const sessions = ref([]);
const activeSessionId = ref('');

const activeSession = computed(() => sessions.value.find((item) => item.connectionId === activeSessionId.value) || null);

async function handleConnect(payload) {
  busy.value = true;
  connectError.value = '';
  configError.value = '';

  try {
    const targetId = editingConnectionId.value || payload.id || '';
    const request = normalizeConnectionPayload({ ...payload, id: targetId });
    const result = targetId ? await updateConnectionConfig(request) : await saveConnectionConfig(request);
    const connection = normalizeConnection(result.connection);

    await loadSavedConnections();
    await testConnection(connection.id);
    openSession(connection);
    startNewConnection();
  } catch (error) {
    connectError.value = error instanceof Error ? error.message : '连接失败';
  } finally {
    busy.value = false;
  }
}

async function connectFromSaved(item) {
  busy.value = true;
  connectError.value = '';
  configError.value = '';
  activeSessionId.value = '';

  try {
    await testConnection(item.id);
    openSession(item);
  } catch (error) {
    connectError.value = error instanceof Error ? error.message : '连接失败';
  } finally {
    busy.value = false;
  }
}

function openSession(connection) {
  const session = {
    connectionId: connection.id,
    connectionName: connection.name,
    host: connection.host,
    port: Number(connection.port) || 22,
    username: connection.username,
    authMethod: connection.authMethod || 'password',
  };

  sessions.value = [...sessions.value.filter((item) => item.connectionId !== session.connectionId), session];
  activeSessionId.value = session.connectionId;
  consoleHeightPercent.value = 58;
}

function activateSession(connectionId) {
  activeSessionId.value = connectionId;
}

function closeSession(connectionId) {
  const index = sessions.value.findIndex((item) => item.connectionId === connectionId);
  sessions.value = sessions.value.filter((item) => item.connectionId !== connectionId);
  if (activeSessionId.value !== connectionId) {
    return;
  }
  const next = sessions.value[Math.max(0, index - 1)];
  activeSessionId.value = next?.connectionId || '';
}

function showConnectHome() {
  activeSessionId.value = '';
  startNewConnection();
}

function startNewConnection() {
  draftConnection.value = defaultConnectionDraft();
  editingConnectionId.value = '';
  connectError.value = '';
}

function editSavedConnection(item) {
  activeSessionId.value = '';
  editingConnectionId.value = item.id;
  connectError.value = '';
  draftConnection.value = {
    id: item.id,
    name: item.name,
    host: item.host,
    port: Number(item.port) || 22,
    username: item.username,
    password: '',
    authMethod: item.authMethod || 'password',
  };
}

async function removeSavedConnection(item) {
  if (!window.confirm(`删除连接 "${item.name}"？`)) {
    return;
  }

  busy.value = true;
  connectError.value = '';
  configError.value = '';

  try {
    await deleteConnectionConfig(item.id);
    savedConnections.value = savedConnections.value.filter((connection) => connection.id !== item.id);
    sessions.value = sessions.value.filter((session) => session.connectionId !== item.id);
    if (activeSessionId.value === item.id) {
      activeSessionId.value = '';
    }
    if (editingConnectionId.value === item.id) {
      startNewConnection();
    }
  } catch (error) {
    configError.value = error instanceof Error ? error.message : '删除失败';
  } finally {
    busy.value = false;
  }
}

async function loadSavedConnections() {
  configLoading.value = true;
  configError.value = '';

  try {
    const result = await listConnectionConfigs();
    const connections = Array.isArray(result.connections) ? result.connections : [];
    savedConnections.value = connections.map(normalizeConnection).filter((item) => item.id);
  } catch (error) {
    configError.value = error instanceof Error ? error.message : '读取保存连接失败';
    savedConnections.value = [];
  } finally {
    configLoading.value = false;
  }
}

function normalizeConnection(connection) {
  return {
    id: String(connection?.id || ''),
    name: String(connection?.name || '未命名连接'),
    host: String(connection?.host || ''),
    port: Number(connection?.port) || 22,
    username: String(connection?.username || ''),
    authMethod: String(connection?.authMethod || 'password'),
  };
}

function normalizeConnectionPayload(payload) {
  return {
    id: String(payload.id || ''),
    name: String(payload.name || '').trim(),
    host: String(payload.host || '').trim(),
    port: Number(payload.port) || 22,
    username: String(payload.username || '').trim(),
    password: payload.authMethod === 'password' ? String(payload.password || '') : '',
    authMethod: payload.authMethod || 'password',
  };
}

function defaultConnectionDraft() {
  return {
    id: '',
    name: '默认服务器',
    host: '127.0.0.1',
    port: 22,
    username: 'root',
    password: '',
    authMethod: 'password',
  };
}

function authLabel(authMethod) {
  return authMethod === 'id_rsa' ? '~/.ssh/id_rsa' : '密码';
}

onMounted(() => {
  loadSavedConnections();
});

let moveHandler = null;
let upHandler = null;

function startDrag(event) {
  event.preventDefault();
  isDragging.value = true;
  document.body.style.userSelect = 'none';

  const onMove = (moveEvent) => {
    const pane = document.querySelector('.active-workspace');
    if (!pane) {
      return;
    }
    const rect = pane.getBoundingClientRect();
    const offsetY = moveEvent.clientY - rect.top;
    const percent = (offsetY / rect.height) * 100;
    consoleHeightPercent.value = Math.min(82, Math.max(28, percent));
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
  document.body.style.userSelect = '';
  if (moveHandler) {
    window.removeEventListener('mousemove', moveHandler);
  }
  if (upHandler) {
    window.removeEventListener('mouseup', upHandler);
  }
});
</script>
