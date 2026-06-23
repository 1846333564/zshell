<template>
  <main class="app-shell" @contextmenu="handleAppContextMenu">
    <header class="app-topbar">
      <div class="topbar-drag-region">
        <div class="brand-lockup">
          <span class="brand-glyph">z</span>
          <strong>zShell</strong>
        </div>

        <nav class="app-menu-strip" aria-label="应用菜单">
          <div class="app-menu-item">
            <button class="app-menu-button" type="button">zShell</button>
            <div class="app-menu-dropdown">
              <button type="button" @click="showConnectHome">连接首页</button>
              <button type="button" disabled>关于 zShell</button>
              <button type="button" @click="closeWindow">退出</button>
            </div>
          </div>

          <div class="app-menu-item">
            <button class="app-menu-button" type="button">配置管理</button>
            <div class="app-menu-dropdown">
              <button type="button" @click="showConnectHome">连接配置</button>
              <button type="button" disabled>导入配置</button>
              <button type="button" disabled>导出配置</button>
            </div>
          </div>

          <div class="app-menu-item">
            <button class="app-menu-button" type="button">UI管理</button>
            <div class="app-menu-dropdown">
              <button type="button" @click="resetUiScale">重置缩放</button>
              <button type="button" disabled>主题设置</button>
              <button type="button" disabled>布局设置</button>
            </div>
          </div>
        </nav>
      </div>

      <div class="window-controls" aria-label="窗口控制">
        <button type="button" title="最小化" @click="minimizeWindow">-</button>
        <button type="button" title="最大化/还原" @click="toggleMaximizeWindow">□</button>
        <button type="button" class="close" title="关闭" @click="closeWindow">×</button>
      </div>
    </header>

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
  getUIPreferences,
  listConnectionConfigs,
  saveConnectionConfig,
  saveUIPreferences,
  testConnection,
  updateConnectionConfig,
} from './services/apiClient';

const busy = ref(false);
const configLoading = ref(false);
const configError = ref('');
const connectError = ref('');
const consoleHeightPercent = ref(58);
const isDragging = ref(false);
const uiScale = ref(1);
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
  applyUiScale();
  loadUIPreferences();
  window.addEventListener('keydown', handleGlobalKeydown, true);
});

let moveHandler = null;
let upHandler = null;
let saveUiScaleTimer = null;

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

function handleGlobalKeydown(event) {
  if (!event.ctrlKey || isTerminalEvent(event)) {
    return;
  }

  const key = event.key.toLowerCase();
  if (key === '+' || key === '=') {
    adjustUiScale(0.05);
    event.preventDefault();
    return;
  }
  if (key === '-' || key === '_') {
    adjustUiScale(-0.05);
    event.preventDefault();
    return;
  }
  if (key === '0') {
    setUiScale(1, true);
    event.preventDefault();
  }
}

function isTerminalEvent(event) {
  const target = event.target;
  if (!(target instanceof Element)) {
    return false;
  }
  return Boolean(target.closest('.terminal-wrap'));
}

function adjustUiScale(delta) {
  setUiScale(uiScale.value + delta, true);
}

function resetUiScale() {
  setUiScale(1, true);
}

function setUiScale(value, persist = false) {
  uiScale.value = clampUiScale(value);
  applyUiScale();
  if (persist) {
    scheduleSaveUiScale();
  }
}

function clampUiScale(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 1;
  }
  return Math.min(1.35, Math.max(0.82, Number(parsed.toFixed(2))));
}

function applyUiScale() {
  document.documentElement.style.setProperty('--ui-scale', uiScale.value.toFixed(2));
}

async function loadUIPreferences() {
  try {
    const result = await getUIPreferences();
    setUiScale(result?.preferences?.uiScale || 1, false);
  } catch (error) {
    console.warn('load ui preferences failed', error);
  }
}

function scheduleSaveUiScale() {
  if (saveUiScaleTimer) {
    window.clearTimeout(saveUiScaleTimer);
  }
  saveUiScaleTimer = window.setTimeout(async () => {
    saveUiScaleTimer = null;
    try {
      await saveUIPreferences({ uiScale: uiScale.value });
    } catch (error) {
      console.warn('save ui preferences failed', error);
    }
  }, 260);
}

function handleAppContextMenu(event) {
  const target = event.target;
  if (target instanceof Element && target.closest('.file-manager-shell')) {
    return;
  }
  event.preventDefault();
}

function minimizeWindow() {
  callWindowRuntime('WindowMinimise');
}

function toggleMaximizeWindow() {
  callWindowRuntime('WindowToggleMaximise');
}

function closeWindow() {
  if (!callWindowRuntime('Quit')) {
    window.close();
  }
}

function callWindowRuntime(method) {
  const runtime = window.runtime;
  if (runtime && typeof runtime[method] === 'function') {
    runtime[method]();
    return true;
  }
  return false;
}

onBeforeUnmount(() => {
  document.body.style.userSelect = '';
  window.removeEventListener('keydown', handleGlobalKeydown, true);
  if (saveUiScaleTimer) {
    window.clearTimeout(saveUiScaleTimer);
  }
  if (moveHandler) {
    window.removeEventListener('mousemove', moveHandler);
  }
  if (upHandler) {
    window.removeEventListener('mouseup', upHandler);
  }
});
</script>
