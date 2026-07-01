import { onBeforeUnmount, onMounted, ref } from 'vue';
import { getAppInfo } from '../../services/apiClient';
import { cancelEditorWarmup, scheduleEditorWarmup } from '../../services/editorWarmup';
import { useConnectionSessions } from './useConnectionSessions';
import { useUiPreferences } from './useUiPreferences';
import { useUpdateDialog } from './useUpdateDialog';

export function useAppController() {
  const connections = useConnectionSessions();
  const ui = useUiPreferences();
  const appInfo = ref({});
  const aboutDialog = ref({ visible: false });
  const updates = useUpdateDialog(appInfo);
  const isDragging = ref(false);
  const appReady = ref(false);
  let moveHandler = null;
  let upHandler = null;
  let readyTimer = null;

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
      connections.consoleHeightPercent.value = Math.min(82, Math.max(28, percent));
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

  onMounted(() => {
    connections.loadSavedConnections();
    loadAppInfo();
    ui.applyUiScale();
    ui.applyCurrentTheme();
    ui.loadUIPreferences();
    updates.scheduleStartupUpdateCheck();
    scheduleEditorWarmup();
    readyTimer = window.setTimeout(() => {
      appReady.value = true;
      readyTimer = null;
    }, 900);
    window.addEventListener('keydown', ui.handleGlobalKeydown, true);
  });

  onBeforeUnmount(() => {
    document.body.style.userSelect = '';
    if (readyTimer) {
      window.clearTimeout(readyTimer);
      readyTimer = null;
    }
    window.removeEventListener('keydown', ui.handleGlobalKeydown, true);
    updates.cleanupUpdateDialog();
    cancelEditorWarmup();
    ui.cleanupUiPreferences();
    if (moveHandler) {
      window.removeEventListener('mousemove', moveHandler);
    }
    if (upHandler) {
      window.removeEventListener('mouseup', upHandler);
    }
  });

  return {
    ...connections,
    ...ui,
    ...updates,
    appInfo,
    aboutDialog,
    appReady,
    isDragging,
    showAboutDialog,
    hideAboutDialog,
    startDrag,
    handleAppContextMenu,
    minimizeWindow,
    toggleMaximizeWindow,
    closeWindow,
  };
}
