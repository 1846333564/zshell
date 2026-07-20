function backendBase() {
  return window.__WISHELL_BACKEND_BASE__ || '';
}

function apiUrl(path) {
  return `${backendBase()}${path}`;
}

async function requestJson(url, options) {
  const response = await fetch(apiUrl(url), {
    headers: {
      'Content-Type': 'application/json',
    },
    ...options,
  });

  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(body.error || `request failed: ${response.status}`);
  }

  return body;
}

export function createConnection(payload) {
  return requestJson('/api/connections', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function listConnectionConfigs() {
  return requestJson('/api/config/connections', {
    method: 'GET',
  });
}

export function saveConnectionConfig(payload) {
  return requestJson('/api/config/connections', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function updateConnectionConfig(payload) {
  return requestJson('/api/config/connections', {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export function deleteConnectionConfig(id) {
  return requestJson(`/api/config/connections?id=${encodeURIComponent(id)}`, {
    method: 'DELETE',
  });
}

export function getUIPreferences() {
  return requestJson('/api/config/preferences', {
    method: 'GET',
  });
}

export function saveUIPreferences(payload) {
  return requestJson('/api/config/preferences', {
    method: 'PUT',
    body: JSON.stringify(payload),
  });
}

export function getAppInfo() {
  return requestJson('/api/app/info', {
    method: 'GET',
  });
}

export function checkForUpdate() {
  return requestJson('/api/update/check', {
    method: 'POST',
    body: JSON.stringify({}),
  });
}

export async function applyUpdate(onProgress, options = {}) {
  if (typeof onProgress === 'function') {
    return applyUpdateWithProgress(onProgress, options);
  }

  return requestJson('/api/update/apply', {
    method: 'POST',
    body: JSON.stringify({}),
    signal: options.signal,
  });
}

async function applyUpdateWithProgress(onProgress, options = {}) {
  const response = await fetch(apiUrl('/api/update/apply/stream'), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({}),
    signal: options.signal,
  });

  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || `update failed: ${response.status}`);
  }
  if (!response.body) {
    return response.json();
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let result = null;

  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';

    for (const line of lines) {
      const event = parseJson(line.trim());
      if (!event.type) {
        continue;
      }
      if (event.type === 'progress') {
        onProgress(event.progress || {});
        continue;
      }
      if (event.type === 'error') {
        throw new Error(event.error || '更新失败');
      }
      if (event.type === 'stopped') {
        throw updateStoppedError(event.message);
      }
      if (event.type === 'result') {
        result = event.update || {};
      }
    }

    if (done) {
      break;
    }
  }

  if (buffer.trim()) {
    const event = parseJson(buffer.trim());
    if (event.type === 'progress') {
      onProgress(event.progress || {});
    } else if (event.type === 'error') {
      throw new Error(event.error || '更新失败');
    } else if (event.type === 'stopped') {
      throw updateStoppedError(event.message);
    } else if (event.type === 'result') {
      result = event.update || {};
    }
  }

  return { update: result || {} };
}

export function testConnection(connectionId) {
  return requestJson('/api/ssh/test', {
    method: 'POST',
    body: JSON.stringify({ connectionId }),
  });
}

export function listRemoteFiles(connectionId, path, options = {}) {
  return requestJson('/api/sftp/list', {
    method: 'POST',
    body: JSON.stringify({ connectionId, path }),
    signal: options.signal,
  });
}

export function readRemoteTextFile(connectionId, path, options = {}) {
  return requestJson('/api/sftp/file/read', {
    method: 'POST',
    body: JSON.stringify({ connectionId, path }),
    signal: options.signal,
  });
}

export function readRemoteTextFileWithProgress(connectionId, path, onProgress, options = {}) {
  const canReportProgress = typeof onProgress === 'function';
  const canReportChunk = typeof options.onChunk === 'function';
  if (!canReportProgress && !canReportChunk) {
    return readRemoteTextFile(connectionId, path, options);
  }

  return readRemoteTextFileRaw(connectionId, path, onProgress, options);
}

function readRemoteTextFileRaw(connectionId, path, onProgress, options = {}) {
  const collectContent = options.collectContent !== false;
  const signal = options.signal;

  return new Promise((resolve, reject) => {
    const request = new XMLHttpRequest();
    let settled = false;
    let lastLoadedBytes = 0;
    let deliveredCharacters = 0;

    const cleanup = () => signal?.removeEventListener('abort', abortRequest);
    const resolveOnce = (value) => {
      if (settled) return;
      settled = true;
      cleanup();
      resolve(value);
    };
    const rejectOnce = (error) => {
      if (settled) return;
      settled = true;
      cleanup();
      reject(error);
    };
    const abortRequest = () => request.abort();
    const emitProgress = (progress) => {
      if (settled || typeof onProgress !== 'function') return true;
      try {
        onProgress(progress);
        return true;
      } catch (error) {
        rejectOnce(error instanceof Error ? error : new Error('更新远程文件进度失败'));
        request.abort();
        return false;
      }
    };
    const responseTotal = () => Number(request.getResponseHeader('X-WiShell-File-Size')) || 0;
    const responseMeta = () => ({
      path: decodeTextResponseHeader(request.getResponseHeader('X-WiShell-File-Path')) || path,
      name: decodeTextResponseHeader(request.getResponseHeader('X-WiShell-File-Name')) || remoteBaseName(path),
      modTime: decodeTextResponseHeader(request.getResponseHeader('X-WiShell-File-Mod-Time')),
    });
    const emitAvailableText = (loadedBytes, totalBytes) => {
      if (settled || typeof options.onChunk !== 'function') return;
      if (request.status < 200 || request.status >= 300) return;
      const responseText = request.responseText || '';
      if (responseText.length <= deliveredCharacters) return;
      const text = responseText.slice(deliveredCharacters);
      deliveredCharacters = responseText.length;
      const meta = responseMeta();
      try {
        options.onChunk({
          ...meta,
          fileName: meta.name,
          loadedBytes,
          totalBytes,
          text,
        });
      } catch (error) {
        rejectOnce(error instanceof Error ? error : new Error('增量显示远程文件失败'));
        request.abort();
      }
    };

    request.open('POST', apiUrl('/api/sftp/file/read/raw'), true);
    request.responseType = 'text';
    request.setRequestHeader('Content-Type', 'application/json');

    request.onreadystatechange = () => {
      if (request.readyState !== XMLHttpRequest.HEADERS_RECEIVED || settled) return;
      const meta = responseMeta();
      emitProgress({
        stage: 'downloading',
        path: meta.path,
        fileName: meta.name,
        loadedBytes: 0,
        totalBytes: responseTotal(),
        message: '正在下载远程文件内容',
      });
    };

    request.onprogress = (event) => {
      if (settled) return;
      const meta = responseMeta();
      lastLoadedBytes = Math.max(lastLoadedBytes, Number(event.loaded) || 0);
      const totalBytes = event.lengthComputable ? Number(event.total) : responseTotal();
      emitAvailableText(lastLoadedBytes, totalBytes);
      if (settled) return;
      emitProgress({
        stage: 'downloading',
        path: meta.path,
        fileName: meta.name,
        loadedBytes: lastLoadedBytes,
        totalBytes,
        message: '正在下载远程文件内容',
      });
    };

    request.onload = (event) => {
      if (request.status < 200 || request.status >= 300) {
        const body = parseJson(request.responseText);
        rejectOnce(remoteReadError(body.error || `read failed: ${request.status}`));
        return;
      }

      const meta = responseMeta();
      const expectedBytes = responseTotal();
      const loadedBytes = Math.max(lastLoadedBytes, Number(event.loaded) || 0);
      emitAvailableText(loadedBytes, expectedBytes);
      if (settled) return;
      if (loadedBytes !== expectedBytes) {
        rejectOnce(remoteReadError(`远程文件读取不完整：应读取 ${expectedBytes} 字节，实际读取 ${loadedBytes} 字节`));
        return;
      }
      const totalBytes = expectedBytes;
      const content = request.responseText || '';
      if (!emitProgress({
        stage: 'done',
        path: meta.path,
        fileName: meta.name,
        loadedBytes: totalBytes,
        totalBytes,
        message: '远程文件下载完成',
      })) return;
      resolveOnce({
        file: {
          ...meta,
          size: totalBytes,
          ...(collectContent ? { content } : {}),
        },
      });
    };

    request.onerror = () => {
      rejectOnce(remoteReadError('读取远程文件时网络连接中断'));
    };
    request.ontimeout = () => {
      rejectOnce(remoteReadError('读取远程文件超时'));
    };
    request.onabort = () => {
      rejectOnce(abortError('已取消读取远程文件'));
    };

    if (signal?.aborted) {
      rejectOnce(abortError('已取消读取远程文件'));
      return;
    }
    signal?.addEventListener('abort', abortRequest, { once: true });
    request.send(JSON.stringify({ connectionId, path }));
  });
}

export function saveRemoteTextFile(connectionId, path, content) {
  return requestJson('/api/sftp/file/write', {
    method: 'PUT',
    body: JSON.stringify({ connectionId, path, content }),
  });
}

export async function uploadRemoteFile(connectionId, path, file, onProgress) {
  return uploadRemoteItems(connectionId, path, [{ file, relativePath: file.name }], [], onProgress);
}

export async function uploadRemoteItems(connectionId, path, items, directories = [], onProgress) {
  const formData = buildUploadFormData(connectionId, path, items, directories);
  if (typeof onProgress === 'function') {
    return uploadRemoteItemsWithProgress(formData, onProgress);
  }

  const response = await fetch(apiUrl('/api/sftp/upload'), {
    method: 'POST',
    body: formData,
  });
  const body = await response.json().catch(() => ({}));
  if (!response.ok) {
    throw new Error(body.error || `upload failed: ${response.status}`);
  }
  return body;
}

function buildUploadFormData(connectionId, path, items, directories = []) {
  const formData = new FormData();
  formData.append('connectionId', connectionId);
  formData.append('path', path);

  for (const item of items) {
    formData.append('files', item.file);
    formData.append('relativePaths', item.relativePath || item.file.webkitRelativePath || item.file.name);
  }

  for (const directory of directories) {
    formData.append('directories', directory);
  }

  return formData;
}

async function uploadRemoteItemsWithProgress(formData, onProgress) {
  const response = await fetch(apiUrl('/api/sftp/upload/stream'), {
    method: 'POST',
    body: formData,
  });

  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || `upload failed: ${response.status}`);
  }
  if (!response.body) {
    return response.json();
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  let buffer = '';
  let result = null;

  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';

    for (const line of lines) {
      const event = parseJson(line.trim());
      if (!event.type) {
        continue;
      }
      if (event.type === 'progress') {
        onProgress(event.progress || {});
        continue;
      }
      if (event.type === 'error') {
        throw new Error(event.error || '上传失败');
      }
      if (event.type === 'result') {
        result = event.upload || {};
      }
    }

    if (done) {
      break;
    }
  }

  if (buffer.trim()) {
    const event = parseJson(buffer.trim());
    if (event.type === 'progress') {
      onProgress(event.progress || {});
    } else if (event.type === 'error') {
      throw new Error(event.error || '上传失败');
    } else if (event.type === 'result') {
      result = event.upload || {};
    }
  }

  return result || {};
}

export function getMonitorSnapshot(connectionId, processSort) {
  return requestJson('/api/monitor/snapshot', {
    method: 'POST',
    body: JSON.stringify({ connectionId, processSort }),
  });
}

export function backendDownloadUrl(path) {
  return apiUrl(path);
}

export function archiveRemoteItemsUrl(connectionId, remotePaths) {
  const params = new URLSearchParams();
  params.set('connectionId', connectionId);
  for (const remotePath of remotePaths) {
    params.append('path', remotePath);
  }
  return backendDownloadUrl(`/api/sftp/archive?${params.toString()}`);
}

export async function downloadRemoteItems(connectionId, remotePaths, fileName = 'wiShell-download.zip') {
  const response = await fetch(archiveRemoteItemsUrl(connectionId, remotePaths));
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || `download failed: ${response.status}`);
  }

  const blob = await response.blob();
  const objectUrl = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = objectUrl;
  anchor.download = fileName;
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(objectUrl);
}

export async function downloadRemoteFile(connectionId, remotePath, fileName) {
  const url = backendDownloadUrl(`/api/sftp/download?connectionId=${encodeURIComponent(connectionId)}&path=${encodeURIComponent(remotePath)}`);
  const response = await fetch(url);
  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || `download failed: ${response.status}`);
  }

  const blob = await response.blob();
  const objectUrl = URL.createObjectURL(blob);
  const anchor = document.createElement('a');
  anchor.href = objectUrl;
  anchor.download = fileName || 'download.bin';
  document.body.appendChild(anchor);
  anchor.click();
  anchor.remove();
  URL.revokeObjectURL(objectUrl);
}

export function transferRemoteItems(payload) {
  return requestJson('/api/sftp/transfer', {
    method: 'POST',
    body: JSON.stringify(payload),
  });
}

export function renameRemoteItem(connectionId, path, newName) {
  return requestJson('/api/sftp/rename', {
    method: 'POST',
    body: JSON.stringify({ connectionId, path, newName }),
  });
}

export function deleteRemoteItems(connectionId, items) {
  return requestJson('/api/sftp/delete', {
    method: 'POST',
    body: JSON.stringify({ connectionId, items }),
  });
}

function parseJson(value) {
  try {
    return JSON.parse(value || '{}');
  } catch {
    return {};
  }
}

function decodeTextResponseHeader(value) {
  if (!value) return '';
  try {
    const base64 = value.replace(/-/g, '+').replace(/_/g, '/');
    const padded = base64.padEnd(Math.ceil(base64.length / 4) * 4, '=');
    const binary = atob(padded);
    const bytes = Uint8Array.from(binary, (character) => character.charCodeAt(0));
    return new TextDecoder().decode(bytes);
  } catch {
    return '';
  }
}

function remoteBaseName(path) {
  const parts = String(path || '').split('/').filter(Boolean);
  return parts[parts.length - 1] || String(path || '');
}

function updateStoppedError(message) {
  const error = new Error(message || '更新已停止');
  error.name = 'AbortError';
  return error;
}

function remoteReadError(message) {
  const error = new Error(message || '读取远程文件失败');
  error.remoteReadError = true;
  return error;
}

function abortError(message) {
  const error = new Error(message || '操作已取消');
  error.name = 'AbortError';
  return error;
}
