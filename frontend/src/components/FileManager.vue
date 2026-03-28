<template>
  <div class="file-manager-shell">
    <div class="file-head">
      <h2>文件管理</h2>
      <span>{{ connectionId ? 'SFTP已连接' : '未连接' }}</span>
    </div>

    <div class="file-toolbar">
      <button class="small-btn" :disabled="!connectionId || loading" @click="refreshCurrent">刷新</button>
      <button class="small-btn" :disabled="!connectionId || loading || currentPath === '/'" @click="goParent">上级目录</button>
      <button class="small-btn" :disabled="!connectionId || loading" @click="jumpHome">~ 目录</button>
      <input
        class="path-input"
        :value="currentPath"
        :disabled="!connectionId || loading"
        @change="onPathChange"
      />
    </div>

    <div class="file-toolbar">
      <input ref="uploadRef" type="file" :disabled="!connectionId || uploading" @change="onUpload" />
      <span class="hint">{{ uploading ? '上传中...' : '选择文件后自动上传到当前目录' }}</span>
    </div>

    <div class="file-error">{{ errorMessage || '\u00A0' }}</div>

    <div class="file-split" v-if="connectionId">
      <aside class="dir-tree">
        <div class="tree-title">目录树</div>
        <button
          v-for="node in treeNodes"
          :key="node.path"
          class="tree-node"
          :class="{ active: node.path === currentPath }"
          :style="{ paddingLeft: `${node.depth * 14 + 10}px` }"
          @click="openDir(node.path)"
        >
          {{ node.path === '/' ? '/ (root)' : node.path }}
        </button>
      </aside>

      <section class="file-list-pane">
        <div class="file-list">
          <div class="file-row file-row-head">
            <span>名称</span>
            <span>大小</span>
            <span>修改时间</span>
            <span>操作</span>
          </div>
          <div class="file-row" v-for="entry in entries" :key="entry.path">
            <span class="name-cell" :class="{ dir: entry.isDir }">{{ entry.isDir ? '📁 ' : '📄 ' }}{{ entry.name }}</span>
            <span>{{ entry.isDir ? '-' : formatSize(entry.size) }}</span>
            <span>{{ formatTime(entry.modTime) }}</span>
            <span class="actions-cell">
              <button class="mini-btn" v-if="entry.isDir" :disabled="loading" @click="openDir(entry.path)">进入</button>
              <button class="mini-btn" v-else :disabled="loading" @click="download(entry.path, entry.name)">下载</button>
            </span>
          </div>
          <div v-if="entries.length === 0" class="empty-tip">当前目录为空</div>
        </div>
      </section>
    </div>
    <div class="empty-tip" v-else>连接后可浏览远程文件</div>
  </div>
</template>

<script setup>
import { computed, ref, watch } from 'vue';
import { downloadRemoteFile, listRemoteFiles, uploadRemoteFile } from '../services/apiClient';

const props = defineProps({
  connectionId: {
    type: String,
    default: '',
  },
});

const entries = ref([]);
const currentPath = ref('~');
const loading = ref(false);
const uploading = ref(false);
const errorMessage = ref('');
const uploadRef = ref(null);
const knownDirs = ref(new Set(['/', '~']));

watch(
  () => props.connectionId,
  async (value) => {
    if (!value) {
      entries.value = [];
      currentPath.value = '~';
      knownDirs.value = new Set(['/', '~']);
      errorMessage.value = '';
      return;
    }

    await refresh('~');
  },
  { immediate: true },
);

const treeNodes = computed(() => {
  const list = Array.from(knownDirs.value)
    .filter(Boolean)
    .sort((a, b) => a.localeCompare(b));

  return list.map((path) => ({
    path,
    depth: path === '/' || path === '~' ? 0 : Math.max(1, path.split('/').filter(Boolean).length),
  }));
});

async function refresh(path) {
  if (!props.connectionId) {
    return;
  }

  loading.value = true;
  errorMessage.value = '';

  try {
    const result = await listRemoteFiles(props.connectionId, path);
    currentPath.value = result.path || path;
    entries.value = Array.isArray(result.entries) ? result.entries : [];

    addKnownDir(currentPath.value);
    for (const entry of entries.value) {
      if (entry.isDir) {
        addKnownDir(entry.path);
      }
    }
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '读取目录失败';
  } finally {
    loading.value = false;
  }
}

function addKnownDir(dirPath) {
  if (!dirPath) {
    return;
  }

  const next = new Set(knownDirs.value);
  next.add(dirPath);
  knownDirs.value = next;
}

function refreshCurrent() {
  refresh(currentPath.value || '~');
}

function jumpHome() {
  refresh('~');
}

function goParent() {
  if (currentPath.value === '/' || currentPath.value === '~') {
    return;
  }

  const parts = currentPath.value.split('/').filter(Boolean);
  parts.pop();
  const parent = '/' + parts.join('/');
  refresh(parent === '' ? '/' : parent);
}

function openDir(path) {
  refresh(path);
}

async function onUpload(event) {
  const file = event.target.files?.[0];
  if (!file || !props.connectionId) {
    return;
  }

  uploading.value = true;
  errorMessage.value = '';

  try {
    await uploadRemoteFile(props.connectionId, currentPath.value || '~', file);
    await refresh(currentPath.value || '~');
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '上传失败';
  } finally {
    uploading.value = false;
    if (uploadRef.value) {
      uploadRef.value.value = '';
    }
  }
}

async function download(path, name) {
  if (!props.connectionId) {
    return;
  }

  errorMessage.value = '';
  try {
    await downloadRemoteFile(props.connectionId, path, name);
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '下载失败';
  }
}

function onPathChange(event) {
  const value = event.target.value?.trim();
  if (!value) {
    return;
  }
  refresh(value);
}

function formatSize(size) {
  if (size < 1024) {
    return `${size} B`;
  }
  if (size < 1024 * 1024) {
    return `${(size / 1024).toFixed(1)} KB`;
  }
  if (size < 1024 * 1024 * 1024) {
    return `${(size / (1024 * 1024)).toFixed(1)} MB`;
  }
  return `${(size / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}

function formatTime(iso) {
  if (!iso) {
    return '-';
  }
  const date = new Date(iso);
  if (Number.isNaN(date.getTime())) {
    return iso;
  }
  return date.toLocaleString();
}
</script>
