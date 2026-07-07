function backendBase() {
  return window.__WISHELL_BACKEND_BASE__ || '';
}

function apiUrl(path) {
  return `${backendBase()}${path}`;
}

const STREAM_PARSE_YIELD_EVENTS = 4;

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

export function readRemoteTextFile(connectionId, path) {
  return requestJson('/api/sftp/file/read', {
    method: 'POST',
    body: JSON.stringify({ connectionId, path }),
  });
}

export async function readRemoteTextFileWithProgress(connectionId, path, onProgress, options = {}) {
  const canReportProgress = typeof onProgress === 'function';
  const canReportChunk = typeof options.onChunk === 'function';
  if (!canReportProgress && !canReportChunk) {
    return readRemoteTextFile(connectionId, path);
  }

  try {
    return await readRemoteTextFileStream(connectionId, path, onProgress, options);
  } catch (error) {
    if (error?.remoteReadError) {
      throw error;
    }
    if (canReportProgress) {
      onProgress({
        stage: 'fallback',
        message: '流式读取不可用，正在切换普通读取',
      });
    }
    return readRemoteTextFile(connectionId, path);
  }
}

async function readRemoteTextFileStream(connectionId, path, onProgress, options = {}) {
  const response = await fetch(apiUrl('/api/sftp/file/read/stream'), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({ connectionId, path }),
  });

  if (!response.ok) {
    const body = await response.json().catch(() => ({}));
    throw new Error(body.error || `read failed: ${response.status}`);
  }
  if (!response.body) {
    return response.json();
  }

  const reader = response.body.getReader();
  const decoder = new TextDecoder();
  const contentDecoder = new TextDecoder();
  const contentParts = [];
  const collectContent = options.collectContent !== false;
  let buffer = '';
  let file = null;
  let lastChunk = null;
  let parsedEvents = 0;

  const appendChunk = (chunk = {}) => {
    lastChunk = chunk;
    const bytes = decodeBase64Bytes(chunk.data);
    if (bytes.length === 0) {
      return;
    }
    const text = contentDecoder.decode(bytes, { stream: true });
    if (!text) {
      return;
    }
    if (collectContent) {
      contentParts.push(text);
    }
    options.onChunk?.({ ...chunk, text });
  };

  const handleEvent = (event) => {
    if (!event.type) {
      return;
    }
    if (event.type === 'progress') {
      onProgress?.(event.progress || {});
      return;
    }
    if (event.type === 'chunk') {
      appendChunk(event.chunk || {});
      return;
    }
    if (event.type === 'error') {
      throw remoteReadError(event.error || '读取远程文件失败');
    }
    if (event.type === 'result') {
      file = event.file || {};
    }
  };

  while (true) {
    const { value, done } = await reader.read();
    buffer += decoder.decode(value || new Uint8Array(), { stream: !done });
    const lines = buffer.split('\n');
    buffer = lines.pop() || '';

    for (const line of lines) {
      const event = parseJson(line.trim());
      handleEvent(event);
      parsedEvents += 1;
      if (parsedEvents % STREAM_PARSE_YIELD_EVENTS === 0) {
        await yieldToBrowser();
      }
    }

    if (done) {
      break;
    }
  }

  if (buffer.trim()) {
    const event = parseJson(buffer.trim());
    handleEvent(event);
  }

  if (!file) {
    throw new Error('远程文件读取没有返回内容');
  }

  const tail = contentDecoder.decode();
  if (tail) {
    if (collectContent) {
      contentParts.push(tail);
    }
    options.onChunk?.({ ...(lastChunk || {}), text: tail });
  }
  if (collectContent && file.content == null) {
    file.content = contentParts.join('');
  }

  return { file };
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

function decodeBase64Bytes(value) {
  const binary = atob(value || '');
  const bytes = new Uint8Array(binary.length);
  for (let index = 0; index < binary.length; index += 1) {
    bytes[index] = binary.charCodeAt(index);
  }
  return bytes;
}

function yieldToBrowser() {
  return new Promise((resolve) => {
    if (typeof window !== 'undefined' && typeof window.requestAnimationFrame === 'function') {
      window.requestAnimationFrame(() => resolve());
      return;
    }
    setTimeout(resolve, 0);
  });
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
