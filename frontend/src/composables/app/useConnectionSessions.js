import { computed, ref } from 'vue';
import {
  deleteConnectionConfig,
  listConnectionConfigs,
  saveConnectionConfig,
  testConnection,
  updateConnectionConfig,
} from '../../services/apiClient';

export function useConnectionSessions() {
  const busy = ref(false);
  const configLoading = ref(false);
  const configError = ref('');
  const connectError = ref('');
  const consoleHeightPercent = ref(58);
  const savedConnections = ref([]);
  const draftConnection = ref(defaultConnectionDraft());
  const editingConnectionId = ref('');
  const sessions = ref([]);
  const activeSessionId = ref('');
  const activeSession = computed(() => sessions.value.find((item) => item.connectionId === activeSessionId.value) || null);

  async function handleConnect(payload) {
    if (busy.value || configLoading.value) {
      return;
    }

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
    if (busy.value || configLoading.value || !item?.id) {
      return;
    }

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
    if (busy.value || configLoading.value || !item?.id) {
      return;
    }

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

  return {
    busy,
    configLoading,
    configError,
    connectError,
    consoleHeightPercent,
    savedConnections,
    draftConnection,
    editingConnectionId,
    sessions,
    activeSessionId,
    activeSession,
    handleConnect,
    connectFromSaved,
    activateSession,
    closeSession,
    showConnectHome,
    startNewConnection,
    editSavedConnection,
    removeSavedConnection,
    loadSavedConnections,
    authLabel,
    workModeLabel,
  };
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
