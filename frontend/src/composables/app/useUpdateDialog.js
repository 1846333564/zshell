import { ref } from 'vue';
import { applyUpdate, checkForUpdate } from '../../services/apiClient';

const releaseLatestURL = 'https://github.com/1846333564/zshell/releases/latest';

export function useUpdateDialog(appInfo) {
  const updateDialog = ref(defaultUpdateDialog());
  let startupUpdateTimer = null;
  let updateAbortController = null;
  let stopUpdateRequested = false;
  let updateProgressLogKey = '';

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
    if (updateDialog.value.status === 'applying' || updateDialog.value.status === 'stopping') {
      return;
    }
    updateAbortController = new AbortController();
    stopUpdateRequested = false;
    updateDialog.value = {
      ...updateDialog.value,
      status: 'applying',
      showProgress: true,
      canStop: true,
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
      const result = await applyUpdate(handleUpdateProgress, { signal: updateAbortController.signal });
      const update = result.update || {};
      if (update.message === '当前已经是最新版本') {
        updateDialog.value = {
          ...updateDialog.value,
          status: 'done',
          available: false,
          title: '已经是最新版本',
          canStop: false,
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
        canStop: false,
        progress: 100,
        progressLabel: '准备重启',
        detail: '替换脚本已启动',
        transferText: '',
        message: '更新已准备完成，zShell 即将重启。',
      };
      pushUpdateLog('更新准备完成，即将重启', 'done');
    } catch (error) {
      if (stopUpdateRequested || isAbortError(error)) {
        markUpdateStopped();
        return;
      }
      const message = error instanceof Error ? error.message : '更新失败';
      pushUpdateLog(`更新失败：${message}`, `error:${message}`);
      updateDialog.value = {
        ...updateDialog.value,
        status: 'error',
        canStop: false,
        error: message,
        message: '更新未完成。',
        releaseUrl: updateDialog.value.releaseUrl || releaseLatestURL,
      };
    } finally {
      updateAbortController = null;
      stopUpdateRequested = false;
    }
  }

  function stopApplyUpdate() {
    if (!updateDialog.value.canStop) {
      return;
    }
    stopUpdateRequested = true;
    updateAbortController?.abort();
    pushUpdateLog('已请求停止更新', 'stop-requested');
    updateDialog.value = {
      ...updateDialog.value,
      status: 'stopping',
      canStop: false,
      progressLabel: '正在停止',
      detail: '正在中断更新请求',
      transferText: '',
      message: '正在停止更新，请稍候。',
      error: '',
    };
  }

  function markUpdateStopped() {
    pushUpdateLog('更新已停止', 'stopped');
    updateDialog.value = {
      ...updateDialog.value,
      status: 'stopped',
      available: true,
      title: '更新已停止',
      canStop: false,
      progressLabel: '已停止',
      detail: '可以重新确认更新',
      transferText: '',
      message: '更新已停止，临时下载文件会被清理。',
      error: '',
    };
  }

  function handleUpdateProgress(progress) {
    if (stopUpdateRequested) {
      return;
    }
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
      canStop: !isFinalUpdateStage(progress.stage),
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

  function pushUpdateLog(text, key = '') {
    const logs = Array.isArray(updateDialog.value.logs) ? updateDialog.value.logs : [];
    const next = [...logs, updateLogItem(text, key)].slice(-48);
    updateDialog.value = {
      ...updateDialog.value,
      logs: next,
    };
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
    if (updateDialog.value.status === 'applying' || updateDialog.value.status === 'stopping') {
      return;
    }
    updateProgressLogKey = '';
    updateDialog.value = defaultUpdateDialog();
  }

  function openReleasePage() {
    const url = updateDialog.value.releaseUrl || releaseLatestURL;
    window.open(url, '_blank', 'noopener');
  }

  function cleanupUpdateDialog() {
    if (startupUpdateTimer) {
      window.clearTimeout(startupUpdateTimer);
    }
    updateAbortController?.abort();
  }

  return {
    updateDialog,
    checkUpdatesFromAbout,
    confirmApplyUpdate,
    stopApplyUpdate,
    scheduleStartupUpdateCheck,
    closeUpdateDialog,
    openReleasePage,
    cleanupUpdateDialog,
  };
}

function defaultUpdateDialog() {
  return {
    visible: false,
    status: 'idle',
    available: false,
    showProgress: false,
    canStop: false,
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

function updateProgressLogKeyFor(progress) {
  const percent = clampProgress(progress.percent);
  if (progress.stage === 'downloading') {
    return `${progress.stage}:${progress.attempt || 0}:${Math.floor(percent / 5) * 5}`;
  }
  return `${progress.stage || 'progress'}:${progress.message || ''}:${progress.attempt || 0}`;
}

function isFinalUpdateStage(stage) {
  return stage === 'scheduling' || stage === 'restarting' || stage === 'done' || stage === 'stopped';
}

function isAbortError(error) {
  return error && error.name === 'AbortError';
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
