function backendBase() {
  return window.__ZSHELL_BACKEND_BASE__ || '';
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

export async function applyUpdate(onProgress) {
  if (typeof onProgress === 'function') {
    return applyUpdateWithProgress(onProgress);
  }

  return requestJson('/api/update/apply', {
    method: 'POST',
    body: JSON.stringify({}),
  });
}

async function applyUpdateWithProgress(onProgress) {
  const response = await fetch(apiUrl('/api/update/apply/stream'), {
    method: 'POST',
    headers: {
      'Content-Type': 'application/json',
    },
    body: JSON.stringify({}),
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

export function listRemoteFiles(connectionId, path) {
  return requestJson('/api/sftp/list', {
    method: 'POST',
    body: JSON.stringify({ connectionId, path }),
  });
}

export function readRemoteTextFile(connectionId, path) {
  return requestJson('/api/sftp/file/read', {
    method: 'POST',
    body: JSON.stringify({ connectionId, path }),
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

  return new Promise((resolve, reject) => {
    const xhr = new XMLHttpRequest();
    xhr.open('POST', apiUrl('/api/sftp/upload'), true);

    xhr.upload.onprogress = (event) => {
      if (typeof onProgress === 'function') {
        onProgress({
          loaded: event.loaded,
          total: event.total,
          lengthComputable: event.lengthComputable,
        });
      }
    };

    xhr.onload = () => {
      const body = parseJson(xhr.responseText);
      if (xhr.status < 200 || xhr.status >= 300) {
        reject(new Error(body.error || `upload failed: ${xhr.status}`));
        return;
      }
      resolve(body);
    };

    xhr.onerror = () => reject(new Error('upload failed: network error'));
    xhr.onabort = () => reject(new Error('upload failed: aborted'));
    xhr.send(formData);
  });
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

export async function downloadRemoteItems(connectionId, remotePaths, fileName = 'zshell-download.zip') {
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
