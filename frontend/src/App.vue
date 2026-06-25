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
              <button type="button" @click="showAboutDialog">关于 zShell</button>
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

    <section class="desktop-layout" :class="{ 'home-layout': !activeSession }">
      <aside v-if="activeSession" class="monitor-sidebar panel">
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
                    <span>{{ item.host }}:{{ item.port }} · {{ item.username }} · {{ authLabel(item.authMethod) }} · {{ workModeLabel(item.workMode) }}</span>
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
              :terminal-font-size="terminalFontSize"
              @terminal-font-size-change="handleTerminalFontSizeChange"
            />
          </div>

          <div class="splitter" title="拖拽调整控制台高度" @mousedown.prevent="startDrag"></div>

          <div class="file-band panel">
            <FileManager
              :key="`${activeSession.connectionId}:${activeSession.workMode}`"
              :connection-id="activeSession.connectionId"
              :work-mode="activeSession.workMode"
              :hardware="activeSession.hardware"
            />
          </div>
        </section>
      </section>
    </section>

    <Teleport to="body">
      <div v-if="aboutDialog.visible" class="modal-backdrop" @click.self="hideAboutDialog">
        <section class="app-dialog about-dialog" @click.stop>
          <header class="dialog-head">
            <div>
              <strong>关于 zShell</strong>
              <span>版本 {{ appInfo.version || '0.0.1' }}</span>
            </div>
            <button type="button" class="dialog-close" @click="hideAboutDialog">×</button>
          </header>

          <div class="dialog-body">
            <p>属于{{ appInfo.company || '重庆创翼科技有限公司' }}，开发者{{ appInfo.developer || 'zly' }}，{{ appInfo.channel || '暂时内测版' }}。</p>
          </div>

          <footer class="dialog-actions">
            <button class="small-btn" type="button" :disabled="updateDialog.status === 'checking' || updateDialog.status === 'applying'" @click="checkUpdatesFromAbout">
              检查更新
            </button>
            <button class="small-btn" type="button" @click="hideAboutDialog">关闭</button>
          </footer>
        </section>
      </div>

      <div v-if="updateDialog.visible" class="modal-backdrop" @click.self="closeUpdateDialog">
        <section class="app-dialog update-dialog" @click.stop>
          <header class="dialog-head">
            <div>
              <strong>{{ updateDialog.title }}</strong>
              <span>{{ updateDialog.subtitle }}</span>
            </div>
            <button type="button" class="dialog-close" :disabled="updateDialog.status === 'applying'" @click="closeUpdateDialog">×</button>
          </header>

          <div class="dialog-body">
            <p>{{ updateDialog.message }}</p>
            <div v-if="updateDialog.showProgress" class="update-progress-panel">
              <div class="update-progress-head">
                <strong>{{ updateDialog.progressLabel }}</strong>
                <span>{{ updateDialog.progress }}%</span>
              </div>
              <div class="progress-track update-progress-track">
                <span :style="{ width: `${updateDialog.progress}%` }"></span>
              </div>
              <div class="update-progress-detail">
                <span>{{ updateDialog.detail || '等待下一步...' }}</span>
                <span>{{ updateDialog.transferText }}</span>
              </div>
              <div v-if="updateDialog.logs.length" class="update-progress-log">
                <div v-for="item in updateDialog.logs" :key="item.id" class="update-progress-log-row">
                  <span>{{ item.time }}</span>
                  <strong>{{ item.text }}</strong>
                </div>
              </div>
            </div>
            <pre v-if="updateDialog.notes" class="release-notes">{{ updateDialog.notes }}</pre>
            <p v-if="updateDialog.error" class="dialog-error">{{ updateDialog.error }}</p>
          </div>

          <footer class="dialog-actions">
            <button
              v-if="updateDialog.available"
              class="small-btn"
              type="button"
              :disabled="updateDialog.status === 'applying'"
              @click="confirmApplyUpdate"
            >
              确认更新
            </button>
            <button v-if="updateDialog.releaseUrl" class="small-btn" type="button" @click="openReleasePage">打开下载页</button>
            <button class="small-btn" type="button" :disabled="updateDialog.status === 'applying'" @click="closeUpdateDialog">关闭</button>
          </footer>
        </section>
      </div>
    </Teleport>
  </main>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, ref } from 'vue';
import ConnectionForm from './components/ConnectionForm.vue';
import FileManager from './components/FileManager.vue';
import MonitorPanel from './components/MonitorPanel.vue';
import TerminalTabs from './components/TerminalTabs.vue';
import {
  applyUpdate,
  checkForUpdate,
  deleteConnectionConfig,
  getAppInfo,
  getUIPreferences,
  listConnectionConfigs,
  saveConnectionConfig,
  saveUIPreferences,
  testConnection,
  updateConnectionConfig,
} from './services/apiClient';
import { cancelEditorWarmup, scheduleEditorWarmup } from './services/editorWarmup';

const busy = ref(false);
const configLoading = ref(false);
const configError = ref('');
const connectError = ref('');
const consoleHeightPercent = ref(58);
const isDragging = ref(false);
const uiScale = ref(1);
const terminalFontSize = ref(14);
const savedConnections = ref([]);
const draftConnection = ref(defaultConnectionDraft());
const editingConnectionId = ref('');
const sessions = ref([]);
const activeSessionId = ref('');
const appInfo = ref({});
const aboutDialog = ref({ visible: false });
const updateDialog = ref(defaultUpdateDialog());
const releaseLatestURL = 'https://github.com/1846333564/zshell/releases/latest';

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

    const testResult = await testConnection(connection.id);
    const testedConnection = normalizeConnection(testResult.connection || { ...connection, hardware: testResult.hardware });
    await loadSavedConnections();
    openSession(testedConnection);
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

  try {
    const testResult = await testConnection(item.id);
    const testedConnection = normalizeConnection(testResult.connection || { ...item, hardware: testResult.hardware });
    await loadSavedConnections();
    openSession(testedConnection);
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
    workMode: normalizeWorkMode(connection.workMode),
    hardware: normalizeHardware(connection.hardware),
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
    workMode: normalizeWorkMode(item.workMode),
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
    workMode: normalizeWorkMode(connection?.workMode),
    hardware: normalizeHardware(connection?.hardware),
  };
}

function normalizeHardware(value) {
  return {
    cpuThreads: Math.max(1, Number(value?.cpuThreads) || 1),
    cpuCores: Math.max(0, Number(value?.cpuCores) || 0),
    cpuModel: String(value?.cpuModel || ''),
    memoryTotalBytes: Math.max(0, Number(value?.memoryTotalBytes) || 0),
    readAt: String(value?.readAt || ''),
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
    workMode: normalizeWorkMode(payload.workMode),
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
    workMode: 'ops',
  };
}

function authLabel(authMethod) {
  return authMethod === 'id_rsa' ? '~/.ssh/id_rsa' : '密码';
}

function normalizeWorkMode(value) {
  return ['frontend', 'backend', 'ops'].includes(value) ? value : 'ops';
}

function workModeLabel(workMode) {
  switch (normalizeWorkMode(workMode)) {
    case 'frontend':
      return '前端模式';
    case 'backend':
      return '后端模式';
    default:
      return '运维模式';
  }
}

async function loadAppInfo() {
  try {
    const result = await getAppInfo();
    appInfo.value = result.app || {};
  } catch (error) {
    console.warn('load app info failed', error);
  }
}

function showAboutDialog() {
  aboutDialog.value.visible = true;
}

function hideAboutDialog() {
  aboutDialog.value.visible = false;
}

function defaultUpdateDialog() {
  return {
    visible: false,
    status: 'idle',
    available: false,
    showProgress: false,
    progress: 0,
    progressLabel: '准备更新',
    detail: '',
    transferText: '',
    logs: [],
    title: '检查更新',
    subtitle: '',
    message: '',
    notes: '',
    error: '',
    releaseUrl: '',
  };
}

async function checkUpdatesFromAbout() {
  updateDialog.value = {
    ...defaultUpdateDialog(),
    visible: true,
    status: 'checking',
    title: '正在检查更新',
    subtitle: `当前版本 ${appInfo.value.version || '0.0.1'}`,
    message: '正在连接 GitHub Release...',
  };

  try {
    const result = await checkForUpdate();
    const update = result.update || {};
    const currentVersion = update.currentVersion || appInfo.value.version || '0.0.1';
    const latestVersion = update.latestVersion || currentVersion;
    if (update.available) {
      updateDialog.value = {
        visible: true,
        status: 'ready',
        available: true,
        title: '发现新版本',
        subtitle: `${currentVersion} -> ${latestVersion}`,
        message: '是否确认更新？确认后会下载新版本、替换当前程序并自动重启。',
        notes: update.notes || '',
        error: '',
        releaseUrl: update.releaseUrl || releaseLatestURL,
      };
      return;
    }

    updateDialog.value = {
      visible: true,
      status: 'done',
      available: false,
      title: '已经是最新版本',
      subtitle: `当前版本 ${currentVersion}`,
      message: '没有发现可用更新。',
      notes: '',
      error: '',
      releaseUrl: update.releaseUrl || releaseLatestURL,
    };
  } catch (error) {
    updateDialog.value = {
      ...defaultUpdateDialog(),
      visible: true,
      status: 'error',
      title: '检查更新失败',
      subtitle: `当前版本 ${appInfo.value.version || '0.0.1'}`,
      message: '无法完成更新检查。',
      error: error instanceof Error ? error.message : '检查更新失败',
      releaseUrl: releaseLatestURL,
    };
  }
}

async function confirmApplyUpdate() {
  updateDialog.value = {
    ...updateDialog.value,
    status: 'applying',
    showProgress: true,
    progress: 2,
    progressLabel: '准备更新',
    detail: '正在初始化更新任务',
    transferText: '',
    logs: [updateLogItem('开始应用更新')],
    message: '正在下载并应用更新，完成后应用会自动重启。',
    error: '',
  };
  updateProgressLogKey = '';

  try {
    const result = await applyUpdate(handleUpdateProgress);
    const update = result.update || {};
    if (update.message === '当前已经是最新版本') {
      updateDialog.value = {
        ...updateDialog.value,
        status: 'done',
        available: false,
        title: '已经是最新版本',
        progress: 100,
        progressLabel: '无需更新',
        detail: `当前版本 ${update.currentVersion || appInfo.value.version || '0.0.1'}`,
        transferText: '',
        message: '当前已经是最新版本。',
      };
      pushUpdateLog('当前已经是最新版本', 'already-latest');
      return;
    }
    updateDialog.value = {
      ...updateDialog.value,
      status: 'applying',
      available: false,
      title: '正在重启',
      progress: 100,
      progressLabel: '准备重启',
      detail: '替换脚本已启动',
      transferText: '',
      message: '更新已准备完成，zShell 即将重启。',
    };
    pushUpdateLog('更新准备完成，即将重启', 'done');
  } catch (error) {
    const message = error instanceof Error ? error.message : '更新失败';
    pushUpdateLog(`更新失败：${message}`, `error:${message}`);
    updateDialog.value = {
      ...updateDialog.value,
      status: 'error',
      error: message,
      message: '更新未完成。',
      releaseUrl: updateDialog.value.releaseUrl || releaseLatestURL,
    };
  }
}

function handleUpdateProgress(progress) {
  const percent = clampProgress(progress.percent);
  const transferText = updateProgressTransferText(progress);
  const detailParts = [];
  if (progress.detail) {
    detailParts.push(progress.detail);
  }
  if (progress.assetName) {
    detailParts.push(progress.assetName);
  }
  if (progress.attempt && progress.maxAttempts) {
    detailParts.push(`第 ${progress.attempt}/${progress.maxAttempts} 次`);
  }

  updateDialog.value = {
    ...updateDialog.value,
    showProgress: true,
    progress: percent,
    progressLabel: progress.message || updateDialog.value.progressLabel || '正在更新',
    detail: detailParts.join(' · '),
    transferText,
    message: progress.message || updateDialog.value.message,
    releaseUrl: progress.releaseUrl || updateDialog.value.releaseUrl || releaseLatestURL,
  };

  const key = updateProgressLogKeyFor(progress);
  if (key && key !== updateProgressLogKey) {
    updateProgressLogKey = key;
    pushUpdateLog(progressLogText(progress, percent), key);
  }
}

function updateProgressLogKeyFor(progress) {
  const percent = clampProgress(progress.percent);
  if (progress.stage === 'downloading') {
    return `${progress.stage}:${progress.attempt || 0}:${Math.floor(percent / 5) * 5}`;
  }
  return `${progress.stage || 'progress'}:${progress.message || ''}:${progress.attempt || 0}`;
}

function progressLogText(progress, percent) {
  const parts = [`${percent}%`, progress.message || '正在更新'];
  if (progress.loadedBytes || progress.totalBytes) {
    parts.push(updateProgressTransferText(progress));
  }
  if (progress.detail && progress.stage !== 'downloading') {
    parts.push(progress.detail);
  }
  return parts.filter(Boolean).join(' · ');
}

function pushUpdateLog(text, key = '') {
  const logs = Array.isArray(updateDialog.value.logs) ? updateDialog.value.logs : [];
  const next = [...logs, updateLogItem(text, key)].slice(-48);
  updateDialog.value = {
    ...updateDialog.value,
    logs: next,
  };
}

function updateLogItem(text, key = '') {
  return {
    id: `${Date.now()}-${key || text}-${Math.random().toString(16).slice(2)}`,
    time: new Date().toLocaleTimeString(),
    text,
  };
}

function updateProgressTransferText(progress) {
  const loaded = Number(progress.loadedBytes) || 0;
  const total = Number(progress.totalBytes) || 0;
  if (loaded > 0 && total > 0) {
    return `${formatUpdateBytes(loaded)} / ${formatUpdateBytes(total)}`;
  }
  if (loaded > 0) {
    return `${formatUpdateBytes(loaded)} 已下载`;
  }
  if (total > 0) {
    return `总大小 ${formatUpdateBytes(total)}`;
  }
  return '';
}

function clampProgress(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed)) {
    return 0;
  }
  return Math.min(100, Math.max(0, Math.round(parsed)));
}

function formatUpdateBytes(size) {
  const value = Number(size) || 0;
  if (value < 1024) {
    return `${Math.max(0, Math.round(value))} B`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB`;
  }
  if (value < 1024 * 1024 * 1024) {
    return `${(value / (1024 * 1024)).toFixed(1)} MB`;
  }
  return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function openReleasePage() {
  const url = updateDialog.value.releaseUrl || releaseLatestURL;
  window.open(url, '_blank', 'noopener');
}

function scheduleStartupUpdateCheck() {
  startupUpdateTimer = window.setTimeout(runStartupUpdateCheck, 1500);
}

async function runStartupUpdateCheck() {
  startupUpdateTimer = null;
  try {
    const result = await checkForUpdate();
    const update = result.update || {};
    if (!update.available || updateDialog.value.visible) {
      return;
    }
    const currentVersion = update.currentVersion || appInfo.value.version || '0.0.1';
    const latestVersion = update.latestVersion || currentVersion;
    updateDialog.value = {
      visible: true,
      status: 'ready',
      available: true,
      title: '发现新版本',
      subtitle: `${currentVersion} -> ${latestVersion}`,
      message: '是否确认更新？确认后会下载新版本、替换当前程序并自动重启。',
      notes: update.notes || '',
      error: '',
      releaseUrl: update.releaseUrl || releaseLatestURL,
    };
  } catch (error) {
    console.warn('startup update check failed', error);
  }
}

function closeUpdateDialog() {
  if (updateDialog.value.status === 'applying') {
    return;
  }
  updateProgressLogKey = '';
  updateDialog.value = defaultUpdateDialog();
}

onMounted(() => {
  loadSavedConnections();
  loadAppInfo();
  applyUiScale();
  loadUIPreferences();
  scheduleStartupUpdateCheck();
  scheduleEditorWarmup();
  window.addEventListener('keydown', handleGlobalKeydown, true);
});

let moveHandler = null;
let upHandler = null;
let saveUiScaleTimer = null;
let startupUpdateTimer = null;
let updateProgressLogKey = '';

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
  const scale = uiScale.value;
  document.documentElement.style.setProperty('--ui-scale', scale.toFixed(2));
  document.documentElement.style.setProperty('--ui-scale-inverse', (1 / scale).toFixed(6));
}

async function loadUIPreferences() {
  try {
    const result = await getUIPreferences();
    setUiScale(result?.preferences?.uiScale || 1, false);
    terminalFontSize.value = clampTerminalFontSize(result?.preferences?.terminalFontSize || 14);
  } catch (error) {
    console.warn('load ui preferences failed', error);
  }
}

function scheduleSaveUiScale() {
  scheduleSavePreferences();
}

function handleTerminalFontSizeChange(value) {
  terminalFontSize.value = clampTerminalFontSize(value);
  scheduleSavePreferences();
}

function clampTerminalFontSize(value) {
  const parsed = Number(value);
  if (!Number.isFinite(parsed) || parsed <= 0) {
    return 14;
  }
  return Math.min(28, Math.max(10, Math.round(parsed)));
}

function scheduleSavePreferences() {
  if (saveUiScaleTimer) {
    window.clearTimeout(saveUiScaleTimer);
  }
  saveUiScaleTimer = window.setTimeout(async () => {
    saveUiScaleTimer = null;
    try {
      await saveUIPreferences({
        uiScale: uiScale.value,
        terminalFontSize: terminalFontSize.value,
      });
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
  if (startupUpdateTimer) {
    window.clearTimeout(startupUpdateTimer);
  }
  cancelEditorWarmup();
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
