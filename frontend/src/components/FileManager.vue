<template>
  <div class="file-manager-shell" @click="onShellClick" @contextmenu.prevent.stop="openBlankContextMenu($event)">
    <div class="file-path-row">
      <button
        ref="pathHistoryButton"
        class="path-history-button"
        type="button"
        title="路径历史"
        :disabled="!connectionId || pathHistory.length === 0"
        @click.stop="togglePathHistory"
      >
        ◷
      </button>
      <input
        class="path-input"
        :value="pathDraft"
        :disabled="!connectionId || loading"
        @input="onPathInput"
        @change="commitPathDraft"
        @keydown.tab.prevent.stop="completePathDraft"
        @click.stop
      />
      <span class="hint">{{ statusText }}</span>
    </div>

    <input ref="fileInput" class="hidden-file-input" type="file" multiple @change="onFilePickerUpload" />
    <input ref="folderInput" class="hidden-file-input" type="file" multiple webkitdirectory directory @change="onFolderPickerUpload" />

    <div v-if="errorMessage" class="file-error">{{ errorMessage }}</div>

    <button
      v-if="uploadProgress.visible && !uploadProgress.expanded"
      class="upload-progress-chip"
      :class="{ done: uploadProgress.status === 'done', error: uploadProgress.status === 'error' }"
      type="button"
      @click.stop="expandUploadPanel"
      @contextmenu.prevent.stop
    >
      <strong>{{ uploadTitle }}</strong>
      <span>{{ uploadPercent }}%</span>
    </button>

    <div
      v-if="uploadProgress.visible && uploadProgress.expanded"
      class="upload-progress-panel"
      :class="{ done: uploadProgress.status === 'done', error: uploadProgress.status === 'error' }"
      @click.stop
      @contextmenu.prevent.stop
    >
      <div class="upload-progress-head">
        <strong>{{ uploadTitle }}</strong>
        <span>{{ uploadPercent }}%</span>
      </div>
      <div class="upload-progress-meta">
        <span>{{ uploadDetail }}</span>
        <span>{{ formatUploadSpeed(uploadProgress.speed) }}</span>
      </div>
      <div class="progress-track">
        <span :style="{ width: `${uploadPercent}%` }"></span>
      </div>
      <div class="upload-progress-meta">
        <span>{{ formatSize(uploadProgress.loadedBytes) }} / {{ formatSize(uploadProgress.totalBytes) }}</span>
        <span>{{ uploadProgress.targetPath }}</span>
      </div>

      <div v-if="uploadProgress.files.length" class="upload-file-list">
        <div v-for="item in uploadProgress.files" :key="item.id" class="upload-file-row">
          <div class="upload-file-main">
            <span>{{ item.relativePath }}</span>
            <span>{{ formatSize(item.loaded) }} / {{ formatSize(item.size) }}</span>
          </div>
          <div class="progress-track small">
            <span :style="{ width: `${uploadFilePercent(item)}%` }"></span>
          </div>
        </div>
      </div>

      <div v-if="uploadProgress.message" class="upload-progress-message">{{ uploadProgress.message }}</div>
    </div>

    <div v-if="connectionId" class="file-split" :class="{ 'nav-collapsed': navCollapsed }">
      <aside v-if="!navCollapsed" class="path-navigator" @contextmenu.prevent.stop="openBlankContextMenu($event)">
        <div class="path-nav-head">
          <span>路径</span>
          <button class="path-nav-toggle" title="折叠导航" @click.stop="navCollapsed = true">‹</button>
        </div>

        <div
          v-for="node in treeNodes"
          :key="node.path"
          class="path-node-row"
          :class="{ active: node.path === currentPath, opened: node.opened }"
          :style="{ paddingLeft: `${node.depth * 14 + 8}px` }"
        >
          <button
            class="path-node-main"
            :class="{ active: node.path === currentPath, opened: node.opened }"
            @click.stop="openDir(node.path)"
            @contextmenu.prevent.stop="openPathContextMenu($event, node.path)"
          >
            <span class="node-label">{{ displayPath(node.path) }}</span>
          </button>
          <button
            v-if="node.hasChildren"
            class="node-toggle"
            :title="node.collapsed ? '展开' : '折叠'"
            @click.stop="togglePathNode(node.path)"
            @contextmenu.prevent.stop="openPathContextMenu($event, node.path)"
          >
            {{ node.collapsed ? '▸' : '▾' }}
          </button>
          <span v-else class="node-toggle-spacer"></span>
        </div>
      </aside>

      <button v-else class="path-nav-rail" title="展开路径" @click.stop="navCollapsed = false">›</button>

      <section
        class="file-list-pane"
        :class="{ 'drag-over': dragOver }"
        @click="onPaneClick"
        @contextmenu.prevent.stop="openBlankContextMenu($event)"
        @dragover.prevent="onDragOver"
        @dragleave="onDragLeave"
        @drop.prevent="onDropUpload"
      >
        <div v-if="dragOver" class="drop-hint">释放后上传到当前目录</div>

        <div class="breadcrumb-row">
          <button
            v-for="crumb in breadcrumbs"
            :key="crumb.path"
            class="crumb"
            :class="{ active: crumb.path === currentPath }"
            :disabled="crumb.path === currentPath || loading"
            @click.stop="openDir(crumb.path)"
          >
            {{ crumb.label }}
          </button>
        </div>

        <div class="file-list">
          <div class="file-row file-row-head" :style="{ gridTemplateColumns }">
            <span v-for="column in columns" :key="column.key" class="file-head-cell">
              {{ column.label }}
              <span class="column-resizer" @mousedown.prevent.stop="startColumnResize($event, column.key)"></span>
            </span>
          </div>
          <div
            v-for="(entry, index) in orderedEntries"
            :key="entry.path"
            class="file-row"
            :class="{ selected: isSelected(entry) }"
            :style="{ gridTemplateColumns }"
            draggable="true"
            @click.stop="selectEntry(entry, index, $event)"
            @dblclick.stop="openEntry(entry)"
            @contextmenu.prevent.stop="openEntryContextMenu($event, entry, index)"
            @dragstart="onRemoteDragStart($event, entry, index)"
          >
            <span class="name-cell" :class="{ dir: entry.isDir }">{{ entry.name }}</span>
            <span>{{ entry.isDir ? '文件夹' : '文件' }}</span>
            <span>{{ entry.isDir ? '-' : formatSize(entry.size) }}</span>
            <span>{{ formatTime(entry.modTime) }}</span>
            <span>{{ entry.mode || '-' }}</span>
            <span>{{ entry.owner || '-' }}</span>
          </div>
          <div v-if="orderedEntries.length === 0" class="empty-tip">当前目录为空。</div>
        </div>
      </section>
    </div>
    <div v-else class="empty-tip">连接后可浏览远程文件</div>

    <Teleport to="body">
      <div
        v-if="pathHistoryOpen"
        class="path-history-popover"
        :style="pathHistoryStyle"
        @click.stop
        @contextmenu.prevent.stop
      >
        <div class="path-history-head">
          <strong>路径历史</strong>
          <span>{{ pathHistory.length }} 项</span>
        </div>
        <div class="path-history-list">
          <button
            v-for="itemPath in pathHistory"
            :key="itemPath"
            type="button"
            :class="{ active: itemPath === currentPath }"
            @click="openHistoryPath(itemPath)"
          >
            {{ itemPath }}
          </button>
        </div>
      </div>

      <div
        v-for="editorWindow in editors"
        :key="editorWindow.id"
        class="remote-editor-window"
        :class="{
          active: editorWindow.id === activeEditorId,
          minimized: editorWindow.windowState === 'minimized',
          maximized: editorWindow.windowState === 'maximized',
        }"
        :style="editorWindowStyle(editorWindow)"
        @mousedown.stop="activateEditor(editorWindow.id)"
        @click.stop
        @contextmenu.prevent.stop
      >
        <header class="remote-editor-head" @mousedown.prevent.stop="startEditorDrag($event, editorWindow)" @dblclick.stop="toggleMaximizeEditor(editorWindow)">
          <div class="remote-editor-title">
            <strong>{{ editorTitle(editorWindow) }}{{ editorDirty(editorWindow) ? ' *' : '' }}</strong>
            <span>{{ editorWindow.path }}</span>
          </div>
          <div class="remote-editor-actions" @mousedown.stop>
            <span>{{ editorStatus(editorWindow) }}</span>
            <button class="small-btn" :disabled="!editorDirty(editorWindow) || editorWindow.loading || editorWindow.saving" @click="saveEditor(editorWindow)">保存</button>
            <button class="editor-window-control" type="button" title="最小化" :disabled="editorWindow.loading || editorWindow.saving" @click="minimizeEditor(editorWindow)">_</button>
            <button class="editor-window-control" type="button" :title="editorWindow.windowState === 'normal' ? '最大化' : '还原'" :disabled="editorWindow.loading || editorWindow.saving" @click="toggleMaximizeEditor(editorWindow)">
              {{ editorWindow.windowState === 'normal' ? '□' : '❐' }}
            </button>
            <button class="editor-window-control danger" type="button" title="关闭" :disabled="editorWindow.loading || editorWindow.saving" @click="requestEditorClose(editorWindow)">×</button>
          </div>
        </header>

        <textarea
          v-if="editorWindow.windowState !== 'minimized'"
          class="remote-editor-textarea"
          v-model="editorWindow.content"
          :disabled="editorWindow.loading || editorWindow.saving"
          :spellcheck="false"
          @focus="activateEditor(editorWindow.id)"
        ></textarea>

        <footer v-if="editorWindow.windowState !== 'minimized'" class="remote-editor-foot">
          <span>{{ editorMeta(editorWindow) }}</span>
          <span class="remote-editor-error">{{ editorWindow.error || '\u00A0' }}</span>
        </footer>

        <div v-if="editorWindow.closePrompt.visible && editorWindow.windowState !== 'minimized'" class="editor-close-backdrop">
          <section class="editor-close-dialog">
            <strong>文件已修改</strong>
            <span>{{ editorWindow.path }}</span>
            <div class="editor-close-actions">
              <button class="small-btn" :disabled="editorWindow.saving" @click="saveEditorFromPrompt(editorWindow)">保存</button>
              <button class="small-btn" :disabled="editorWindow.saving" @click="saveAndCloseEditorFromPrompt(editorWindow)">保存并关闭</button>
              <button class="small-btn danger" :disabled="editorWindow.saving" @click="discardEditorFromPrompt(editorWindow)">不保存并关闭</button>
              <button class="small-btn" :disabled="editorWindow.saving" @click="cancelEditorClosePrompt(editorWindow)">取消</button>
            </div>
          </section>
        </div>
      </div>

      <div
        v-if="contextMenu.visible"
        class="context-menu file-context-menu"
        :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
        @click.stop
        @contextmenu.prevent.stop
      >
        <button :disabled="!connectionId || loading" @click="refreshFromMenu">刷新</button>

        <div class="context-menu-separator"></div>
        <button v-if="contextCanOpenDirectory" :disabled="loading" @click="openContextDirectory">打开</button>
        <template v-if="contextEntry && !contextEntry.isDir">
          <button
            v-for="action in fileOpenActions"
            :key="action.key"
            :disabled="loading"
            @click="runContextFileOpenAction(action.key)"
          >
            {{ action.label }}
          </button>
          <button :disabled="loading" @click="downloadContextEntry">下载</button>
        </template>
        <button :disabled="!hasSelection || loading" @click="downloadSelectionFromMenu">下载选中</button>
        <button class="danger" :disabled="!canDeleteFromMenu || loading" @click="deleteFromMenu">删除</button>

        <div class="context-menu-separator"></div>
        <button :disabled="!connectionId || uploading" @click="chooseFilesFromMenu">上传文件到此处...</button>
        <button :disabled="!connectionId || uploading" @click="chooseFolderFromMenu">上传文件夹到此处...</button>

        <div class="context-menu-separator"></div>
        <button :disabled="!hasSelection" @click="copySelectionFromMenu">复制</button>
        <button :disabled="!hasSelection" @click="cutSelectionFromMenu">剪切</button>
        <button :disabled="!canPaste || loading" @click="pasteClipboardFromMenu">粘贴到此处</button>

        <div class="context-menu-separator"></div>
        <button :disabled="!contextPath" @click="copyPathFromMenu">复制路径</button>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import {
  archiveRemoteItemsUrl,
  backendDownloadUrl,
  deleteRemoteItems,
  downloadRemoteFile,
  downloadRemoteItems,
  listRemoteFiles,
  readRemoteTextFile,
  saveRemoteTextFile,
  transferRemoteItems,
  uploadRemoteItems,
} from '../services/apiClient';
import { viewportContextMenuPosition } from '../utils/contextMenuPosition';

const ROOT_PATH = '/';
const HOME_PATH = '~';
const WORK_MODE_START_PATHS = {
  frontend: '/var',
  backend: '/opt',
  ops: ROOT_PATH,
};
const CLIPBOARD_KEY = 'zshell.remote-file.clipboard.v1';
const DIRECTORY_CACHE_LIMIT = 1200;
const MAX_PRELOAD_CONCURRENCY = 32;
const MIN_COLUMN_WIDTH = 70;
const UPLOAD_CLOSE_DELAY_MS = 1200;
const DEFAULT_FILE_OPEN_ACTION = 'textEdit';
const DEFAULT_EDITOR_WIDTH = 980;
const DEFAULT_EDITOR_HEIGHT = 660;
const MIN_EDITOR_TOP = 48;
const CLIPBOARD_ACTIONS = new Set(['copy', 'move']);
const directoryCache = new Map();

const columns = [
  { key: 'name', label: '名称', width: 280 },
  { key: 'type', label: '类型', width: 82 },
  { key: 'size', label: '大小', width: 110 },
  { key: 'modTime', label: '修改时间', width: 175 },
  { key: 'mode', label: '权限', width: 125 },
  { key: 'owner', label: '所属用户', width: 110 },
];

const fileOpenActions = [
  { key: 'textEdit', label: '在线编辑' },
];

const props = defineProps({
  connectionId: {
    type: String,
    default: '',
  },
  workMode: {
    type: String,
    default: 'ops',
  },
  hardware: {
    type: Object,
    default: () => ({}),
  },
});

const entries = ref([]);
const currentPath = ref(initialPathForMode(props.workMode));
const pathDraft = ref(currentPath.value);
const loading = ref(false);
const refreshingCached = ref(false);
const uploading = ref(false);
const dragOver = ref(false);
const errorMessage = ref('');
const navCollapsed = ref(false);
const selectedPaths = ref(new Set());
const lastSelectedIndex = ref(-1);
const clipboard = ref(readClipboard());
const fileInput = ref(null);
const folderInput = ref(null);
const pathHistoryButton = ref(null);
const pendingUploadPath = ref('');
const pathHistory = ref([]);
const pathHistoryOpen = ref(false);
const columnWidths = reactive(loadColumnWidths());
const uploadProgress = reactive({
  visible: false,
  expanded: false,
  status: 'idle',
  targetPath: '',
  files: [],
  directoryCount: 0,
  totalBytes: 0,
  loadedBytes: 0,
  speed: 0,
  startedAt: 0,
  message: '',
});
const editors = ref([]);
const activeEditorId = ref('');
const pathMeta = ref(
  initialPathMeta(currentPath.value),
);

let resizeState = null;
let editorDragState = null;
let uploadCloseTimer = null;
let editorZIndex = 900;
let refreshSerial = 0;
let preloadSerial = 0;
let preloadController = null;

const contextMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  entry: null,
  targetPath: '',
  targetKind: 'blank',
});
const pathHistoryPanel = reactive({
  x: 0,
  y: 0,
  width: 320,
});

watch(
  () => [props.connectionId, props.workMode],
  async ([value, workMode]) => {
    clearSelection();
    cancelDirectoryPreload();
    const startPath = initialPathForMode(workMode);
    resetPathState(startPath);
    if (!value) {
      entries.value = [];
      currentPath.value = startPath;
      errorMessage.value = '';
      return;
    }

    await refresh(startPath);
  },
  { immediate: true },
);

watch(
  currentPath,
  (value) => {
    pathDraft.value = value;
  },
);

const orderedEntries = computed(() =>
  [...entries.value].sort((a, b) => {
    if (a.isDir !== b.isDir) {
      return a.isDir ? -1 : 1;
    }
    return String(a.name || '').localeCompare(String(b.name || ''), undefined, { sensitivity: 'base' });
  }),
);

const selectedEntries = computed(() => orderedEntries.value.filter((entry) => selectedPaths.value.has(entry.path)));
const hasSelection = computed(() => selectedEntries.value.length > 0);
const canPaste = computed(() => Boolean(props.connectionId && clipboard.value?.items?.length));
const contextEntry = computed(() => contextMenu.entry);
const contextCanOpenDirectory = computed(() => Boolean(contextMenu.entry?.isDir || contextMenu.targetKind === 'directory'));
const contextPath = computed(() => contextMenu.entry?.path || contextMenu.targetPath || defaultDirectoryPath());
const contextTargetDir = computed(() => (contextMenu.entry?.isDir ? contextMenu.entry.path : contextMenu.targetPath || defaultDirectoryPath()));
const contextDeleteItem = computed(() => {
  if (hasSelection.value || contextMenu.entry) {
    return null;
  }
  const targetPath = normalizePath(contextMenu.targetPath);
  if (contextMenu.targetKind !== 'directory' || !isDeletableRemotePath(targetPath)) {
    return null;
  }
  return { path: targetPath, isDir: true };
});
const canDeleteFromMenu = computed(() => hasSelection.value || Boolean(contextDeleteItem.value));
const gridTemplateColumns = computed(() => columns.map((column) => `${columnWidths[column.key]}px`).join(' '));
const activeEditor = computed(() => editors.value.find((item) => item.id === activeEditorId.value) || null);
const preloadConcurrency = computed(() => Math.min(MAX_PRELOAD_CONCURRENCY, Math.max(1, Math.round(Number(props.hardware?.cpuThreads) || 1))));

const statusText = computed(() => {
  if (!props.connectionId) {
    return '未连接';
  }
  if (uploading.value) {
    return `上传 ${uploadPercent.value}% · ${formatUploadSpeed(uploadProgress.speed)}`;
  }
  if (refreshingCached.value) {
    return `缓存 ${orderedEntries.value.length} 项 · 正在更新`;
  }
  if (loading.value) {
    return '读取中...';
  }
  if (clipboard.value?.items?.length) {
    return `${clipboard.value.action === 'move' ? '剪切' : '复制'}了 ${clipboard.value.items.length} 项`;
  }
  return `${orderedEntries.value.length} 项`;
});

const breadcrumbs = computed(() => buildBreadcrumbs(currentPath.value));
const pathHistoryStyle = computed(() => ({
  left: `${pathHistoryPanel.x}px`,
  top: `${pathHistoryPanel.y}px`,
  width: `${pathHistoryPanel.width}px`,
}));
const uploadPercent = computed(() => {
  if (uploadProgress.status === 'done') {
    return 100;
  }
  if (uploadProgress.totalBytes <= 0) {
    return 0;
  }
  return Math.min(100, Math.max(0, Math.round((uploadProgress.loadedBytes / uploadProgress.totalBytes) * 100)));
});
const uploadTitle = computed(() => {
  if (uploadProgress.status === 'done') {
    return '上传完成';
  }
  if (uploadProgress.status === 'error') {
    return '上传失败';
  }
  return `上传 ${uploadProgress.files.length + uploadProgress.directoryCount} 项`;
});
const uploadDetail = computed(() => {
  if (uploadProgress.status === 'done') {
    return '已完成，稍后自动折叠';
  }
  if (uploadProgress.status === 'error') {
    return '请检查连接或远程目录权限';
  }
  return `总进度 ${uploadPercent.value}%`;
});

const treeNodes = computed(() => {
  const paths = Array.from(pathMeta.value.keys()).filter(Boolean).sort(comparePaths);
  return paths
    .filter((itemPath) => isPathVisible(itemPath))
    .map((itemPath) => {
      const meta = pathMeta.value.get(itemPath) || {};
      return {
        path: itemPath,
        depth: pathDepth(itemPath),
        opened: Boolean(meta.opened),
        collapsed: Boolean(meta.collapsed),
        hasChildren: paths.some((candidate) => candidate !== itemPath && parentPath(candidate) === itemPath),
      };
    });
});

onMounted(() => {
  applyColumnWidths();
  window.addEventListener('storage', onStorage);
  window.addEventListener('keydown', onKeydown);
  window.addEventListener('pointerdown', onGlobalPointerDown, true);
});

onBeforeUnmount(() => {
  window.removeEventListener('storage', onStorage);
  window.removeEventListener('keydown', onKeydown);
  window.removeEventListener('pointerdown', onGlobalPointerDown, true);
  clearUploadCloseTimer();
  cancelDirectoryPreload();
  stopColumnResize();
  stopEditorDrag();
});

async function refresh(targetPath = currentPath.value || initialPathForMode(props.workMode), options = {}) {
  if (!props.connectionId) {
    return;
  }

  if (options.cancelPreload !== false) {
    cancelDirectoryPreload();
  }
  const requestedPath = targetPath || initialPathForMode(props.workMode);
  const serial = (refreshSerial += 1);
  const cached = options.useCache === false ? null : getCachedDirectory(props.connectionId, requestedPath);
  if (cached) {
    applyDirectoryListing(cached.path, cached.entries, cached.requestedPath || requestedPath);
    loading.value = false;
    refreshingCached.value = true;
  } else {
    loading.value = true;
    refreshingCached.value = false;
  }
  errorMessage.value = '';

  try {
    const result = await listRemoteFiles(props.connectionId, requestedPath);
    if (serial !== refreshSerial) {
      return;
    }
    const resolvedPath = result.path || requestedPath;
    const nextEntries = Array.isArray(result.entries) ? result.entries : [];
    applyDirectoryListing(resolvedPath, nextEntries, requestedPath);
    setCachedDirectory(props.connectionId, requestedPath, resolvedPath, nextEntries, { rememberTree: true });
    if (resolvedPath !== requestedPath) {
      setCachedDirectory(props.connectionId, resolvedPath, resolvedPath, nextEntries, { rememberTree: true });
    }
    scheduleDirectoryPreload();
  } catch (error) {
    if (serial === refreshSerial) {
      errorMessage.value = error instanceof Error ? error.message : '读取目录失败';
    }
  } finally {
    if (serial === refreshSerial) {
      loading.value = false;
      refreshingCached.value = false;
    }
  }
}

function applyDirectoryListing(resolvedPath, nextEntries, requestedPath = resolvedPath) {
  currentPath.value = resolvedPath;
  entries.value = cloneEntries(nextEntries);
  rememberPathHistory(resolvedPath);

  rememberPath(ROOT_PATH);
  rememberPath(HOME_PATH);
  if (requestedPath === HOME_PATH || requestedPath.startsWith('~/')) {
    rememberPath(HOME_PATH, { opened: true });
  }
  rememberDirectoryTree(resolvedPath, entries.value);
  keepExistingSelection();
}

function resetPathState(startPath = initialPathForMode(props.workMode)) {
  pathMeta.value = initialPathMeta(startPath);
  pathHistory.value = [];
  pathHistoryOpen.value = false;
}

function initialPathForMode(workMode) {
  return WORK_MODE_START_PATHS[normalizeWorkMode(workMode)] || ROOT_PATH;
}

function defaultDirectoryPath() {
  return currentPath.value || initialPathForMode(props.workMode);
}

function normalizeWorkMode(workMode) {
  return ['frontend', 'backend', 'ops'].includes(workMode) ? workMode : 'ops';
}

function initialPathMeta(startPath) {
  const meta = new Map([
    [ROOT_PATH, { opened: false, collapsed: false }],
    [HOME_PATH, { opened: false, collapsed: false }],
  ]);
  const normalized = normalizePath(startPath);
  if (normalized && !meta.has(normalized)) {
    meta.set(normalized, { opened: false, collapsed: false });
  }
  return meta;
}

function getCachedDirectory(connectionId, remotePath) {
  const cached = directoryCache.get(directoryCacheKey(connectionId, remotePath));
  if (!cached) {
    return null;
  }
  return {
    ...cached,
    entries: cloneEntries(cached.entries),
  };
}

function setCachedDirectory(connectionId, requestedPath, resolvedPath, nextEntries, options = {}) {
  const key = directoryCacheKey(connectionId, requestedPath);
  if (!key) {
    return;
  }
  const resolved = normalizePath(resolvedPath || requestedPath);
  const entriesForCache = cloneEntries(nextEntries);
  if (directoryCache.has(key)) {
    directoryCache.delete(key);
  }
  directoryCache.set(key, {
    requestedPath,
    path: resolved,
    entries: entriesForCache,
    cachedAt: Date.now(),
  });
  if (options.rememberTree !== false) {
    rememberDirectoryTree(resolved, entriesForCache);
  }
  while (directoryCache.size > DIRECTORY_CACHE_LIMIT) {
    const oldestKey = directoryCache.keys().next().value;
    directoryCache.delete(oldestKey);
  }
}

function directoryCacheKey(connectionId, remotePath) {
  const normalized = normalizePath(remotePath || initialPathForMode(props.workMode));
  if (!connectionId || !normalized) {
    return '';
  }
  return `${connectionId}\u0000${normalized}`;
}

function cloneEntries(value) {
  return Array.isArray(value) ? value.map((entry) => ({ ...entry })) : [];
}

function rememberDirectoryTree(resolvedPath, nextEntries) {
  rememberParentChain(resolvedPath);
  rememberPath(resolvedPath, { opened: true, collapsed: false });
  for (const entry of nextEntries) {
    if (entry.isDir) {
      rememberParentChain(entry.path);
      rememberPath(entry.path, { opened: false });
    }
  }
}

function scheduleDirectoryPreload() {
  cancelDirectoryPreload();
  const targets = directoryPreloadTargets();
  if (targets.length === 0) {
    return;
  }

  const serial = (preloadSerial += 1);
  const controller = new AbortController();
  preloadController = controller;
  const queue = [...targets];
  const workerCount = Math.min(preloadConcurrency.value, queue.length);

  const workers = Array.from({ length: workerCount }, async () => {
    while (queue.length > 0 && preloadSerial === serial && !controller.signal.aborted) {
      const targetPath = queue.shift();
      if (!targetPath || getCachedDirectory(props.connectionId, targetPath)) {
        continue;
      }
      try {
        const result = await listRemoteFiles(props.connectionId, targetPath, { signal: controller.signal });
        if (preloadSerial !== serial || controller.signal.aborted) {
          return;
        }
        const resolvedPath = result.path || targetPath;
        const nextEntries = Array.isArray(result.entries) ? result.entries : [];
        setCachedDirectory(props.connectionId, targetPath, resolvedPath, nextEntries, { rememberTree: false });
        if (resolvedPath !== targetPath) {
          setCachedDirectory(props.connectionId, resolvedPath, resolvedPath, nextEntries, { rememberTree: false });
        }
      } catch (error) {
        if (error?.name === 'AbortError' || controller.signal.aborted) {
          return;
        }
      }
    }
  });

  Promise.allSettled(workers).finally(() => {
    if (preloadSerial === serial) {
      preloadController = null;
    }
  });
}

function cancelDirectoryPreload() {
  preloadSerial += 1;
  if (preloadController) {
    preloadController.abort();
    preloadController = null;
  }
}

function directoryPreloadTargets() {
  const targets = [];
  const seen = new Set();
  for (const itemPath of Array.from(pathMeta.value.keys()).sort(comparePaths)) {
    const meta = pathMeta.value.get(itemPath) || {};
    if (!isPathVisible(itemPath) || (!meta.opened && itemPath !== currentPath.value)) {
      continue;
    }
    const cached = getCachedDirectory(props.connectionId, itemPath);
    if (!cached) {
      continue;
    }
    for (const entry of cached.entries) {
      if (!entry.isDir) {
        continue;
      }
      const normalized = normalizePath(entry.path);
      if (!normalized || seen.has(normalized) || getCachedDirectory(props.connectionId, normalized)) {
        continue;
      }
      seen.add(normalized);
      targets.push(normalized);
    }
  }
  return targets;
}

function rememberPath(itemPath, patch = {}) {
  const normalized = normalizePath(itemPath);
  if (!normalized) {
    return;
  }

  const next = new Map(pathMeta.value);
  const existing = next.get(normalized) || { opened: false, collapsed: false };
  next.set(normalized, { ...existing, ...patch });
  pathMeta.value = next;
}

function rememberParentChain(itemPath) {
  const normalized = normalizePath(itemPath);
  if (!normalized) {
    return;
  }
  const chain = [];
  let cursor = normalized;
  while (cursor) {
    chain.unshift(cursor);
    cursor = parentPath(cursor);
  }
  for (const segment of chain) {
    rememberPath(segment);
  }
}

function rememberPathHistory(itemPath) {
  const normalized = normalizePath(itemPath);
  if (!normalized) {
    return;
  }
  pathHistory.value = [
    normalized,
    ...pathHistory.value.filter((existing) => existing !== normalized),
  ].slice(0, 25);
}

function togglePathHistory() {
  if (!props.connectionId || pathHistory.value.length === 0) {
    pathHistoryOpen.value = false;
    return;
  }
  if (pathHistoryOpen.value) {
    pathHistoryOpen.value = false;
    return;
  }
  positionPathHistoryPanel();
  pathHistoryOpen.value = true;
}

function positionPathHistoryPanel() {
  const button = pathHistoryButton.value;
  if (!button) {
    return;
  }
  const rect = button.getBoundingClientRect();
  const width = Math.min(420, Math.max(300, window.innerWidth - 24));
  const height = Math.min(260, Math.max(120, pathHistory.value.length * 32 + 42));
  pathHistoryPanel.x = Math.min(Math.max(8, rect.left), Math.max(8, window.innerWidth - width - 8));
  pathHistoryPanel.y = Math.max(8, rect.top - height - 8);
  pathHistoryPanel.width = width;
}

function openHistoryPath(itemPath) {
  pathHistoryOpen.value = false;
  clearSelection();
  refresh(itemPath);
}

function togglePathNode(itemPath) {
  const next = new Map(pathMeta.value);
  const existing = next.get(itemPath) || { opened: false, collapsed: false };
  next.set(itemPath, { ...existing, collapsed: !existing.collapsed });
  pathMeta.value = next;
}

function openDir(itemPath) {
  if (loading.value) {
    return;
  }
  pathHistoryOpen.value = false;
  clearSelection();
  refresh(itemPath);
}

function openEntry(entry) {
  if (entry.isDir) {
    openDir(entry.path);
    return;
  }
  runFileOpenAction(DEFAULT_FILE_OPEN_ACTION, entry);
}

function onPathInput(event) {
  pathDraft.value = event.target.value || '';
}

function commitPathDraft(event) {
  const value = pathDraft.value.trim();
  if (!value) {
    pathDraft.value = currentPath.value;
    return;
  }
  pathHistoryOpen.value = false;
  clearSelection();
  refresh(value);
}

async function completePathDraft() {
  if (!props.connectionId) {
    return;
  }
  const prefix = normalizePath(pathDraft.value);
  if (!prefix) {
    return;
  }
  await ensureCompletionCaches(prefix);
  const matches = pathCompletionMatches(prefix);
  if (matches.length === 1) {
    pathDraft.value = matches[0];
  }
}

async function ensureCompletionCaches(prefix) {
  const prefixes = [prefix];
  const relativePrefix = currentRelativeCompletionPrefix(prefix);
  if (relativePrefix && relativePrefix !== prefix) {
    prefixes.push(relativePrefix);
  }
  for (const item of prefixes) {
    const parent = completionParentPath(item);
    if (!parent || getCachedDirectory(props.connectionId, parent)) {
      continue;
    }
    try {
      const result = await listRemoteFiles(props.connectionId, parent);
      const resolvedPath = result.path || parent;
      const nextEntries = Array.isArray(result.entries) ? result.entries : [];
      setCachedDirectory(props.connectionId, parent, resolvedPath, nextEntries, { rememberTree: false });
      if (resolvedPath !== parent) {
        setCachedDirectory(props.connectionId, resolvedPath, resolvedPath, nextEntries, { rememberTree: false });
      }
    } catch {
      // Completion should remain silent when a parent cannot be read.
    }
  }
}

function pathCompletionMatches(prefix) {
  const candidates = pathCompletionCandidates();
  let matches = completionMatchesForPrefix(prefix, candidates);
  if (matches.length === 0) {
    const relativePrefix = currentRelativeCompletionPrefix(prefix);
    if (relativePrefix && relativePrefix !== prefix) {
      matches = completionMatchesForPrefix(relativePrefix, candidates);
    }
  }
  return matches;
}

function pathCompletionCandidates() {
  const candidates = new Set([ROOT_PATH, HOME_PATH, ...Object.values(WORK_MODE_START_PATHS), ...pathHistory.value, ...Array.from(pathMeta.value.keys())]);
  for (const entry of entries.value) {
    if (entry.isDir) {
      candidates.add(entry.path);
    }
  }
  for (const cached of cachedDirectoriesForConnection(props.connectionId)) {
    candidates.add(cached.path);
    for (const entry of cached.entries) {
      if (entry.isDir) {
        candidates.add(entry.path);
      }
    }
  }
  return candidates;
}

function completionMatchesForPrefix(prefix, candidates) {
  const matches = new Set();
  for (const candidate of candidates) {
    const normalized = normalizePath(candidate);
    const match = completionTarget(prefix, normalized);
    if (match) {
      matches.add(match);
    }
  }
  return Array.from(matches).sort(comparePaths);
}

function currentRelativeCompletionPrefix(prefix) {
  const current = normalizePath(currentPath.value);
  if (!current || current === ROOT_PATH || current === HOME_PATH) {
    return '';
  }
  const raw = String(prefix || '').trim();
  if (!raw) {
    return '';
  }
  const clean = raw.startsWith(ROOT_PATH) ? raw.slice(1) : raw;
  if (!clean || clean.includes('/')) {
    return '';
  }
  return normalizePath(`${current}/${clean}`);
}

function completionParentPath(prefix) {
  const normalized = normalizePath(prefix);
  if (!normalized) {
    return '';
  }
  if (normalized === ROOT_PATH || normalized === HOME_PATH) {
    return '';
  }
  if (normalized.endsWith('/') && normalized !== ROOT_PATH) {
    return normalizePath(normalized.slice(0, -1)) || ROOT_PATH;
  }
  return parentPath(normalized) || ROOT_PATH;
}

function completionTarget(prefix, candidate) {
  if (!candidate || candidate === prefix || !candidate.startsWith(prefix)) {
    return '';
  }
  const remaining = candidate.slice(prefix.length);
  const slashIndex = remaining.indexOf('/');
  if (slashIndex === -1) {
    return candidate;
  }
  return candidate.slice(0, prefix.length + slashIndex);
}

function cachedDirectoriesForConnection(connectionId) {
  if (!connectionId) {
    return [];
  }
  const prefix = `${connectionId}\u0000`;
  const result = [];
  for (const [key, cached] of directoryCache.entries()) {
    if (key.startsWith(prefix)) {
      result.push({
        ...cached,
        entries: cloneEntries(cached.entries),
      });
    }
  }
  return result;
}

function syncDirectoryCacheAfterDelete(items) {
  const deleted = items
    .map((item) => ({
      path: normalizePath(item.path),
      isDir: Boolean(item.isDir),
    }))
    .filter((item) => item.path);
  if (deleted.length === 0) {
    return;
  }

  const prefix = `${props.connectionId}\u0000`;
  for (const [key, cached] of Array.from(directoryCache.entries())) {
    if (!key.startsWith(prefix)) {
      continue;
    }
    const cachedPath = normalizePath(cached.path || cached.requestedPath);
    if (deleted.some((item) => item.isDir && isSameOrChildPath(cachedPath, item.path))) {
      directoryCache.delete(key);
      continue;
    }
    const nextEntries = cloneEntries(cached.entries).filter((entry) => !deleted.some((item) => entryMatchesDeletedItem(entry, item)));
    if (nextEntries.length !== cached.entries.length) {
      directoryCache.set(key, {
        ...cached,
        entries: nextEntries,
        cachedAt: Date.now(),
      });
    }
  }

  entries.value = entries.value.filter((entry) => !deleted.some((item) => entryMatchesDeletedItem(entry, item)));
  removeDeletedPathMeta(deleted);
}

function invalidateDirectoryCache(remotePath) {
  const normalized = normalizePath(remotePath);
  if (!normalized || !props.connectionId) {
    return;
  }
  const keys = [directoryCacheKey(props.connectionId, normalized)];
  for (const key of keys) {
    if (key) {
      directoryCache.delete(key);
    }
  }
}

function entryMatchesDeletedItem(entry, item) {
  const entryPath = normalizePath(entry.path);
  if (!entryPath || !item.path) {
    return false;
  }
  return item.isDir ? isSameOrChildPath(entryPath, item.path) : entryPath === item.path;
}

function removeDeletedPathMeta(deletedItems) {
  const next = new Map(pathMeta.value);
  for (const key of Array.from(next.keys())) {
    if (deletedItems.some((item) => (item.isDir ? isSameOrChildPath(key, item.path) : key === item.path))) {
      next.delete(key);
    }
  }
  pathMeta.value = next;
}

function pathAfterDeletingItems(items) {
  const current = normalizePath(currentPath.value) || initialPathForMode(props.workMode);
  for (const item of items) {
    const deletedPath = normalizePath(item.path);
    if (item.isDir && isSameOrChildPath(current, deletedPath)) {
      return nearestExistingParentAfterDelete(deletedPath, items);
    }
  }
  return current;
}

function nearestExistingParentAfterDelete(deletedPath, deletedItems) {
  let candidate = parentPath(deletedPath) || ROOT_PATH;
  while (candidate && deletedItems.some((item) => item.isDir && isSameOrChildPath(candidate, normalizePath(item.path)))) {
    candidate = parentPath(candidate);
  }
  return candidate || ROOT_PATH;
}

function isDeletableRemotePath(itemPath) {
  const normalized = normalizePath(itemPath);
  return Boolean(normalized && normalized !== ROOT_PATH && normalized !== HOME_PATH);
}

function chooseFiles(targetPath = contextTargetDir.value) {
  pendingUploadPath.value = targetPath || defaultDirectoryPath();
  fileInput.value?.click();
}

function chooseFolder(targetPath = contextTargetDir.value) {
  pendingUploadPath.value = targetPath || defaultDirectoryPath();
  folderInput.value?.click();
}

function chooseFilesFromMenu() {
  const target = contextTargetDir.value;
  hideContextMenu();
  chooseFiles(target);
}

function chooseFolderFromMenu() {
  const target = contextTargetDir.value;
  hideContextMenu();
  chooseFolder(target);
}

async function onFilePickerUpload(event) {
  const files = Array.from(event.target.files || []);
  await uploadPickedFiles(files, [], pendingUploadPath.value || defaultDirectoryPath());
  event.target.value = '';
}

async function onFolderPickerUpload(event) {
  const files = Array.from(event.target.files || []);
  const directories = deriveDirectoriesFromFiles(files);
  await uploadPickedFiles(files, directories, pendingUploadPath.value || defaultDirectoryPath());
  event.target.value = '';
}

async function uploadPickedFiles(files, directories = [], targetPath = defaultDirectoryPath()) {
  const items = files.map((file) => ({
    file,
    relativePath: file.webkitRelativePath || file.name,
  }));
  await uploadItems(items, directories, targetPath);
}

async function uploadItems(items, directories = [], targetPath = defaultDirectoryPath()) {
  if (!props.connectionId || (items.length === 0 && directories.length === 0)) {
    return;
  }

  uploading.value = true;
  errorMessage.value = '';
  startUploadProgress(items, directories, targetPath || defaultDirectoryPath());
  let succeeded = false;

  try {
    await uploadRemoteItems(props.connectionId, targetPath || defaultDirectoryPath(), items, directories, onUploadProgress);
    succeeded = true;
    markUploadComplete();
    invalidateDirectoryCache(targetPath || defaultDirectoryPath());
    await refresh(currentPath.value || initialPathForMode(props.workMode), { useCache: false });
  } catch (error) {
    const message = error instanceof Error ? error.message : '上传失败';
    errorMessage.value = message;
    markUploadError(message);
  } finally {
    uploading.value = false;
    if (succeeded) {
      scheduleUploadPanelClose();
    }
  }
}

function onDragOver(event) {
  if (!props.connectionId) {
    return;
  }
  event.dataTransfer.dropEffect = 'copy';
  dragOver.value = true;
}

function onDragLeave(event) {
  if (!event.currentTarget.contains(event.relatedTarget)) {
    dragOver.value = false;
  }
}

async function onDropUpload(event) {
  dragOver.value = false;
  if (!props.connectionId) {
    return;
  }

  const { files, directories } = await collectDroppedItems(event.dataTransfer);
  await uploadItems(files, directories, defaultDirectoryPath());
}

async function download(itemPath, name) {
  if (!props.connectionId) {
    return;
  }

  errorMessage.value = '';
  try {
    await downloadRemoteFile(props.connectionId, itemPath, name);
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '下载失败';
  }
}

async function downloadSelection() {
  if (!hasSelection.value) {
    return;
  }

  const selected = selectedEntries.value;
  const paths = selected.map((entry) => entry.path);
  errorMessage.value = '';

  try {
    if (selected.length === 1 && !selected[0].isDir) {
      await downloadRemoteFile(props.connectionId, selected[0].path, selected[0].name);
      return;
    }
    await downloadRemoteItems(props.connectionId, paths, downloadArchiveName(selected));
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '下载失败';
  }
}

async function downloadSelectionFromMenu() {
  hideContextMenu();
  await downloadSelection();
}

async function downloadContextEntry() {
  const entry = contextMenu.entry;
  hideContextMenu();
  if (!entry || entry.isDir) {
    return;
  }
  await download(entry.path, entry.name);
}

function openContextDirectory() {
  const targetPath = contextMenu.entry?.isDir ? contextMenu.entry.path : contextMenu.targetPath;
  hideContextMenu();
  if (targetPath) {
    openDir(targetPath);
  }
}

async function runContextFileOpenAction(actionKey) {
  const entry = contextMenu.entry;
  hideContextMenu();
  if (!entry || entry.isDir) {
    return;
  }
  await runFileOpenAction(actionKey, entry);
}

async function runFileOpenAction(actionKey, entry) {
  if (actionKey === 'textEdit') {
    await openTextEditor(entry);
  }
}

async function openTextEditor(entry) {
  if (!props.connectionId || !entry || entry.isDir) {
    return;
  }

  const existing = editors.value.find((item) => item.path === entry.path);
  if (existing) {
    if (existing.windowState === 'minimized') {
      existing.windowState = 'normal';
    }
    activateEditor(existing.id);
    return;
  }

  const editorWindow = createEditorWindow(entry);

  try {
    const result = await readRemoteTextFile(props.connectionId, entry.path);
    const file = result.file || {};
    const content = String(file.content ?? '');
    editorWindow.path = String(file.path || entry.path);
    editorWindow.name = String(file.name || entry.name || displayPath(editorWindow.path));
    editorWindow.content = content;
    editorWindow.originalContent = content;
    editorWindow.size = Number(file.size) || content.length;
    editorWindow.modTime = String(file.modTime || entry.modTime || '');
    editorWindow.message = '已打开';
  } catch (error) {
    editorWindow.error = error instanceof Error ? error.message : '打开文件失败';
  } finally {
    editorWindow.loading = false;
  }
}

function createEditorWindow(entry) {
  const bounds = defaultEditorBounds();
  const editorWindow = {
    id: crypto.randomUUID(),
    windowState: 'normal',
    loading: true,
    saving: false,
    path: entry.path,
    name: entry.name,
    content: '',
    originalContent: '',
    size: Number(entry.size) || 0,
    modTime: entry.modTime || '',
    error: '',
    message: '',
    x: bounds.x,
    y: bounds.y,
    width: bounds.width,
    height: bounds.height,
    zIndex: nextEditorZIndex(),
    closePrompt: {
      visible: false,
      afterClose: null,
    },
  };
  editors.value.push(editorWindow);
  activeEditorId.value = editorWindow.id;
  return editorWindow;
}

function defaultEditorBounds() {
  const viewportWidth = window.innerWidth || 1200;
  const viewportHeight = window.innerHeight || 780;
  const width = Math.min(DEFAULT_EDITOR_WIDTH, Math.max(520, viewportWidth - 64));
  const height = Math.min(DEFAULT_EDITOR_HEIGHT, Math.max(360, viewportHeight - 126));
  const offset = (editors.value.length % 7) * 26;
  const centeredX = Math.max(16, Math.round((viewportWidth - width) / 2));
  return {
    x: Math.min(Math.max(16, centeredX + offset), Math.max(16, viewportWidth - 120)),
    y: Math.min(MIN_EDITOR_TOP + offset, Math.max(MIN_EDITOR_TOP, viewportHeight - 120)),
    width,
    height,
  };
}

function nextEditorZIndex() {
  editorZIndex += 1;
  return editorZIndex;
}

function activateEditor(id) {
  const editorWindow = editors.value.find((item) => item.id === id);
  if (!editorWindow) {
    return;
  }
  activeEditorId.value = id;
  editorWindow.zIndex = nextEditorZIndex();
}

function editorDirty(editorWindow) {
  return Boolean(editorWindow && editorWindow.content !== editorWindow.originalContent);
}

function editorTitle(editorWindow) {
  return editorWindow?.name || displayPath(editorWindow?.path) || '远程文件';
}

function editorStatus(editorWindow) {
  if (editorWindow.loading) {
    return '读取中...';
  }
  if (editorWindow.saving) {
    return '保存中...';
  }
  if (editorDirty(editorWindow)) {
    return '未保存';
  }
  return editorWindow.message || '已保存';
}

function editorMeta(editorWindow) {
  if (!editorWindow) {
    return '';
  }
  const parts = [formatSize(new Blob([editorWindow.content]).size)];
  if (editorWindow.modTime) {
    parts.push(formatTime(editorWindow.modTime));
  }
  return parts.join(' · ');
}

function editorWindowStyle(editorWindow) {
  if (editorWindow.windowState === 'minimized') {
    const minimizedIndex = editors.value.filter((item) => item.windowState === 'minimized').findIndex((item) => item.id === editorWindow.id);
    return {
      left: `${16 + Math.max(0, minimizedIndex) * 252}px`,
      bottom: '14px',
      width: '240px',
      height: '38px',
      zIndex: editorWindow.zIndex,
    };
  }
  if (editorWindow.windowState === 'maximized') {
    return {
      left: '12px',
      top: '50px',
      width: 'calc(100vw - 24px)',
      height: 'calc(100vh - 62px)',
      zIndex: editorWindow.zIndex,
    };
  }
  return {
    left: `${editorWindow.x}px`,
    top: `${editorWindow.y}px`,
    width: `${editorWindow.width}px`,
    height: `${editorWindow.height}px`,
    zIndex: editorWindow.zIndex,
  };
}

function minimizeEditor(editorWindow) {
  if (!editorWindow) {
    return;
  }
  editorWindow.windowState = 'minimized';
  activateEditor(editorWindow.id);
}

function toggleMaximizeEditor(editorWindow) {
  if (!editorWindow) {
    return;
  }
  editorWindow.windowState = editorWindow.windowState === 'normal' ? 'maximized' : 'normal';
  activateEditor(editorWindow.id);
}

function startEditorDrag(event, editorWindow) {
  if (!editorWindow || editorWindow.windowState !== 'normal' || event.button !== 0) {
    return;
  }

  activateEditor(editorWindow.id);
  editorDragState = {
    id: editorWindow.id,
    startX: event.clientX,
    startY: event.clientY,
    originX: editorWindow.x,
    originY: editorWindow.y,
  };
  document.body.style.cursor = 'move';
  document.body.style.userSelect = 'none';
  window.addEventListener('mousemove', onEditorDrag);
  window.addEventListener('mouseup', stopEditorDrag);
}

function onEditorDrag(event) {
  if (!editorDragState) {
    return;
  }
  const editorWindow = editors.value.find((item) => item.id === editorDragState.id);
  if (!editorWindow) {
    stopEditorDrag();
    return;
  }

  const maxX = Math.max(16, (window.innerWidth || 1200) - 120);
  const maxY = Math.max(MIN_EDITOR_TOP, (window.innerHeight || 780) - 60);
  editorWindow.x = Math.min(maxX, Math.max(16, editorDragState.originX + event.clientX - editorDragState.startX));
  editorWindow.y = Math.min(maxY, Math.max(MIN_EDITOR_TOP, editorDragState.originY + event.clientY - editorDragState.startY));
}

function stopEditorDrag() {
  if (!editorDragState) {
    return;
  }
  editorDragState = null;
  document.body.style.cursor = '';
  document.body.style.userSelect = '';
  window.removeEventListener('mousemove', onEditorDrag);
  window.removeEventListener('mouseup', stopEditorDrag);
}

async function saveEditor(editorWindow) {
  if (!props.connectionId || !editorWindow || !editorWindow.path || editorWindow.loading || editorWindow.saving) {
    return false;
  }

  editorWindow.saving = true;
  editorWindow.error = '';
  editorWindow.message = '';
  const content = editorWindow.content;

  try {
    const result = await saveRemoteTextFile(props.connectionId, editorWindow.path, content);
    const file = result.file || {};
    editorWindow.path = String(file.path || editorWindow.path);
    editorWindow.name = String(file.name || editorWindow.name || displayPath(editorWindow.path));
    editorWindow.originalContent = content;
    editorWindow.size = Number(file.size) || new Blob([content]).size;
    editorWindow.modTime = String(file.modTime || '');
    editorWindow.message = '已保存';
    invalidateDirectoryCache(parentPath(editorWindow.path) || currentPath.value || initialPathForMode(props.workMode));
    await refresh(currentPath.value || initialPathForMode(props.workMode), { useCache: false });
    return true;
  } catch (error) {
    editorWindow.error = error instanceof Error ? error.message : '保存失败';
    return false;
  } finally {
    editorWindow.saving = false;
  }
}

function requestEditorClose(editorWindow, options = {}) {
  const afterClose = typeof options.afterClose === 'function' ? options.afterClose : null;
  if (!editorWindow || !editorDirty(editorWindow)) {
    closeEditorImmediately(editorWindow);
    runAfterEditorClose(afterClose);
    return;
  }

  editorWindow.closePrompt.visible = true;
  editorWindow.closePrompt.afterClose = afterClose;
  if (editorWindow.windowState === 'minimized') {
    editorWindow.windowState = 'normal';
  }
  activateEditor(editorWindow.id);
}

function closeEditorImmediately(editorWindow) {
  if (!editorWindow) {
    return;
  }
  const index = editors.value.findIndex((item) => item.id === editorWindow.id);
  if (index === -1) {
    return;
  }
  editors.value.splice(index, 1);
  if (activeEditorId.value === editorWindow.id) {
    const next = [...editors.value].sort((left, right) => right.zIndex - left.zIndex)[0];
    activeEditorId.value = next?.id || '';
  }
}

async function runAfterEditorClose(afterClose) {
  if (typeof afterClose === 'function') {
    await afterClose();
  }
}

async function saveEditorFromPrompt(editorWindow) {
  const saved = await saveEditor(editorWindow);
  if (saved) {
    editorWindow.closePrompt.visible = false;
    editorWindow.closePrompt.afterClose = null;
  }
}

async function saveAndCloseEditorFromPrompt(editorWindow) {
  const afterClose = editorWindow.closePrompt.afterClose;
  const saved = await saveEditor(editorWindow);
  if (!saved) {
    return;
  }
  closeEditorImmediately(editorWindow);
  await runAfterEditorClose(afterClose);
}

async function discardEditorFromPrompt(editorWindow) {
  const afterClose = editorWindow.closePrompt.afterClose;
  closeEditorImmediately(editorWindow);
  await runAfterEditorClose(afterClose);
}

function cancelEditorClosePrompt(editorWindow) {
  if (!editorWindow) {
    return;
  }
  editorWindow.closePrompt.visible = false;
  editorWindow.closePrompt.afterClose = null;
}

function selectEntry(entry, index, event) {
  if (event.shiftKey && lastSelectedIndex.value >= 0) {
    const start = Math.min(lastSelectedIndex.value, index);
    const end = Math.max(lastSelectedIndex.value, index);
    const next = new Set(selectedPaths.value);
    for (let i = start; i <= end; i += 1) {
      next.add(orderedEntries.value[i].path);
    }
    selectedPaths.value = next;
    return;
  }

  if (event.ctrlKey || event.metaKey) {
    const next = new Set(selectedPaths.value);
    if (next.has(entry.path)) {
      next.delete(entry.path);
    } else {
      next.add(entry.path);
    }
    selectedPaths.value = next;
    lastSelectedIndex.value = index;
    return;
  }

  selectedPaths.value = new Set([entry.path]);
  lastSelectedIndex.value = index;
}

function isSelected(entry) {
  return selectedPaths.value.has(entry.path);
}

function clearSelection() {
  selectedPaths.value = new Set();
  lastSelectedIndex.value = -1;
}

function keepExistingSelection() {
  const available = new Set(entries.value.map((entry) => entry.path));
  selectedPaths.value = new Set(Array.from(selectedPaths.value).filter((itemPath) => available.has(itemPath)));
}

function onPaneClick(event) {
  if (!event.target.closest('.file-row')) {
    clearSelection();
  }
  hideContextMenu();
}

function onShellClick() {
  hideContextMenu();
  collapseUploadPanel();
  pathHistoryOpen.value = false;
}

function openEntryContextMenu(event, entry, index) {
  if (!selectedPaths.value.has(entry.path)) {
    selectedPaths.value = new Set([entry.path]);
    lastSelectedIndex.value = index;
  }
  openContextMenu(event, {
    entry,
    targetPath: entry.isDir ? entry.path : defaultDirectoryPath(),
    targetKind: entry.isDir ? 'directory' : 'file',
  });
}

function openPathContextMenu(event, targetPath) {
  openContextMenu(event, { targetPath, targetKind: 'directory' });
}

function openBlankContextMenu(event) {
  openContextMenu(event, { targetPath: defaultDirectoryPath(), targetKind: 'blank' });
}

function openContextMenu(event, target = {}) {
  const position = viewportContextMenuPosition(event, { width: 220, height: 470 });
  contextMenu.visible = true;
  contextMenu.entry = target.entry || null;
  contextMenu.targetPath = target.targetPath || defaultDirectoryPath();
  contextMenu.targetKind = target.targetKind || 'blank';
  contextMenu.x = position.x;
  contextMenu.y = position.y;
}

function hideContextMenu() {
  contextMenu.visible = false;
}

function onGlobalPointerDown(event) {
  const target = event.target;
  if (pathHistoryOpen.value && target instanceof Element) {
    if (!target.closest('.path-history-popover') && !target.closest('.path-history-button')) {
      pathHistoryOpen.value = false;
    }
  }
  if (contextMenu.visible) {
    if (target instanceof Element && target.closest('.file-context-menu')) {
      return;
    }
    hideContextMenu();
  }
}

async function refreshFromMenu() {
  const target = contextMenu.targetPath || currentPath.value || initialPathForMode(props.workMode);
  hideContextMenu();
  await refresh(target, { useCache: false });
}

function copySelection() {
  writeClipboard('copy');
}

function cutSelection() {
  writeClipboard('move');
}

function copySelectionFromMenu() {
  hideContextMenu();
  copySelection();
}

function cutSelectionFromMenu() {
  hideContextMenu();
  cutSelection();
}

async function deleteFromMenu() {
  hideContextMenu();
  if (hasSelection.value) {
    await deleteSelection();
    return;
  }
  const item = contextDeleteItem.value;
  if (item) {
    await deleteItems([item]);
  }
}

async function deleteSelection() {
  if (!hasSelection.value || !props.connectionId) {
    return;
  }

  await deleteItems(selectedEntries.value);
}

async function deleteItems(items) {
  if (!items.length || !props.connectionId) {
    return;
  }

  const selected = items.map((item) => ({
    path: item.path,
    isDir: Boolean(item.isDir),
  }));
  if (!window.confirm(deleteConfirmMessage(selected))) {
    return;
  }

  loading.value = true;
  errorMessage.value = '';

  try {
    await deleteRemoteItems(
      props.connectionId,
      selected.map((entry) => ({
        path: entry.path,
        isDir: entry.isDir,
      })),
    );
    const nextPath = pathAfterDeletingItems(selected);
    syncDirectoryCacheAfterDelete(selected);
    clearSelection();
    clearClipboardIfDeleted(selected);
    await refresh(nextPath, { useCache: false });
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '删除失败';
  } finally {
    loading.value = false;
  }
}

function deleteConfirmMessage(selected) {
  const preview = selected
    .slice(0, 6)
    .map((entry) => `- ${entry.path}`)
    .join('\n');
  const more = selected.length > 6 ? `\n... 还有 ${selected.length - 6} 项` : '';
  return `确认强制删除选中的 ${selected.length} 项？\n\n${preview}${more}\n\n此操作不可撤销。`;
}

function writeClipboard(action) {
  if (!hasSelection.value) {
    return;
  }
  const normalizedAction = normalizeClipboardAction(action);

  const payload = {
    sourceConnectionId: props.connectionId,
    action: normalizedAction,
    items: selectedEntries.value.map((entry) => ({
      path: entry.path,
      isDir: entry.isDir,
    })),
    createdAt: Date.now(),
  };
  localStorage.setItem(CLIPBOARD_KEY, JSON.stringify(payload));
  clipboard.value = payload;
}

async function pasteClipboard(targetPath = contextTargetDir.value) {
  if (!canPaste.value) {
    return;
  }

  const pendingClipboard = normalizeClipboardPayload(clipboard.value);
  if (!pendingClipboard) {
    clipboard.value = null;
    localStorage.removeItem(CLIPBOARD_KEY);
    return;
  }

  loading.value = true;
  errorMessage.value = '';

  try {
    await transferRemoteItems({
      sourceConnectionId: pendingClipboard.sourceConnectionId,
      targetConnectionId: props.connectionId,
      targetPath: targetPath || defaultDirectoryPath(),
      action: pendingClipboard.action,
      items: pendingClipboard.items,
    });
    invalidateDirectoryCache(targetPath || defaultDirectoryPath());
    if (pendingClipboard.action === 'move' && pendingClipboard.sourceConnectionId === props.connectionId) {
      for (const item of pendingClipboard.items) {
        invalidateDirectoryCache(parentPath(item.path));
      }
    }
    if (pendingClipboard.action === 'move') {
      localStorage.removeItem(CLIPBOARD_KEY);
      clipboard.value = null;
    }
    await refresh(currentPath.value || initialPathForMode(props.workMode), { useCache: false });
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '粘贴失败';
  } finally {
    loading.value = false;
  }
}

async function pasteClipboardFromMenu() {
  const target = contextTargetDir.value;
  hideContextMenu();
  await pasteClipboard(target);
}

function copyPathFromMenu() {
  const value = contextPath.value;
  hideContextMenu();
  if (value) {
    navigator.clipboard?.writeText(value).catch(() => {});
  }
}

function onRemoteDragStart(event, entry, index) {
  if (!selectedPaths.value.has(entry.path)) {
    selectedPaths.value = new Set([entry.path]);
    lastSelectedIndex.value = index;
  }

  const selected = selectedEntries.value.length ? selectedEntries.value : [entry];
  const paths = selected.map((item) => item.path);
  const fileName = selected.length === 1 && !selected[0].isDir ? selected[0].name : downloadArchiveName(selected);
  const url =
    selected.length === 1 && !selected[0].isDir
      ? backendDownloadUrl(`/api/sftp/download?connectionId=${encodeURIComponent(props.connectionId)}&path=${encodeURIComponent(selected[0].path)}`)
      : archiveRemoteItemsUrl(props.connectionId, paths);
  const absoluteUrl = new URL(url, window.location.origin).toString();

  event.dataTransfer.effectAllowed = 'copy';
  event.dataTransfer.setData('DownloadURL', `application/octet-stream:${fileName}:${absoluteUrl}`);
  event.dataTransfer.setData('text/uri-list', absoluteUrl);
  event.dataTransfer.setData('text/plain', absoluteUrl);
}

function deriveDirectoriesFromFiles(files) {
  const directories = new Set();
  for (const file of files) {
    const relativePath = file.webkitRelativePath || '';
    const parts = relativePath.split('/').filter(Boolean);
    parts.pop();
    for (let i = 1; i <= parts.length; i += 1) {
      directories.add(parts.slice(0, i).join('/'));
    }
  }
  return Array.from(directories);
}

async function collectDroppedItems(dataTransfer) {
  const entriesFromItems = Array.from(dataTransfer.items || [])
    .map((item) => (typeof item.webkitGetAsEntry === 'function' ? item.webkitGetAsEntry() : null))
    .filter(Boolean);

  if (entriesFromItems.length === 0) {
    return {
      files: Array.from(dataTransfer.files || []).map((file) => ({ file, relativePath: file.name })),
      directories: [],
    };
  }

  const files = [];
  const directories = new Set();
  for (const entry of entriesFromItems) {
    await walkDroppedEntry(entry, '', files, directories);
  }
  return { files, directories: Array.from(directories) };
}

async function walkDroppedEntry(entry, parentPath, files, directories) {
  const relativePath = parentPath ? `${parentPath}/${entry.name}` : entry.name;
  if (entry.isFile) {
    const file = await fileFromEntry(entry);
    files.push({ file, relativePath });
    return;
  }

  if (!entry.isDirectory) {
    return;
  }

  directories.add(relativePath);
  const children = await readAllDirectoryEntries(entry.createReader());
  for (const child of children) {
    await walkDroppedEntry(child, relativePath, files, directories);
  }
}

function fileFromEntry(entry) {
  return new Promise((resolve, reject) => {
    entry.file(resolve, reject);
  });
}

function readAllDirectoryEntries(reader) {
  return new Promise((resolve, reject) => {
    const entries = [];
    const readBatch = () => {
      reader.readEntries(
        (batch) => {
          if (batch.length === 0) {
            resolve(entries);
            return;
          }
          entries.push(...batch);
          readBatch();
        },
        (error) => reject(error),
      );
    };
    readBatch();
  });
}

function startUploadProgress(items, directories, targetPath) {
  clearUploadCloseTimer();
  const files = items.map((item, index) => ({
    id: `${Date.now()}-${index}-${item.relativePath || item.file?.name || 'file'}`,
    relativePath: item.relativePath || item.file?.webkitRelativePath || item.file?.name || `file-${index + 1}`,
    size: Number(item.file?.size) || 0,
    loaded: 0,
  }));
  uploadProgress.visible = true;
  uploadProgress.expanded = true;
  uploadProgress.status = 'uploading';
  uploadProgress.targetPath = targetPath;
  uploadProgress.files = files;
  uploadProgress.directoryCount = directories.length;
  uploadProgress.totalBytes = files.reduce((total, item) => total + item.size, 0);
  uploadProgress.loadedBytes = 0;
  uploadProgress.speed = 0;
  uploadProgress.startedAt = Date.now();
  uploadProgress.message = '';
}

function onUploadProgress(progress) {
  if (!uploadProgress.visible || uploadProgress.status !== 'uploading') {
    return;
  }

  const loaded = Number(progress?.loadedBytes ?? progress?.loaded) || 0;
  const total = Number(progress?.totalBytes ?? progress?.total) || 0;
  if (total > uploadProgress.totalBytes) {
    uploadProgress.totalBytes = total;
  }
  const cappedLoaded = uploadProgress.totalBytes > 0 ? Math.min(uploadProgress.totalBytes, loaded) : loaded;
  uploadProgress.loadedBytes = Math.max(uploadProgress.loadedBytes, cappedLoaded);
  distributeUploadLoaded(uploadProgress.loadedBytes);

  const elapsedSeconds = Math.max((Date.now() - uploadProgress.startedAt) / 1000, 0.1);
  uploadProgress.speed = uploadProgress.loadedBytes / elapsedSeconds;
}

function distributeUploadLoaded(loadedBytes) {
  let remaining = loadedBytes;
  for (const item of uploadProgress.files) {
    item.loaded = Math.min(item.size, Math.max(0, remaining));
    remaining -= item.size;
  }
}

function markUploadComplete() {
  uploadProgress.status = 'done';
  uploadProgress.loadedBytes = uploadProgress.totalBytes;
  distributeUploadLoaded(uploadProgress.totalBytes);
  uploadProgress.speed = 0;
  uploadProgress.message = '上传已完成';
}

function markUploadError(message) {
  uploadProgress.status = 'error';
  uploadProgress.expanded = true;
  uploadProgress.message = message || '上传失败';
}

function scheduleUploadPanelClose() {
  clearUploadCloseTimer();
  uploadCloseTimer = window.setTimeout(() => {
    collapseUploadPanel();
    uploadCloseTimer = null;
  }, UPLOAD_CLOSE_DELAY_MS);
}

function clearUploadCloseTimer() {
  if (uploadCloseTimer) {
    window.clearTimeout(uploadCloseTimer);
    uploadCloseTimer = null;
  }
}

function collapseUploadPanel() {
  if (uploadProgress.visible) {
    uploadProgress.expanded = false;
  }
}

function expandUploadPanel() {
  if (uploadProgress.visible) {
    clearUploadCloseTimer();
    uploadProgress.expanded = true;
  }
}

function uploadFilePercent(item) {
  if (uploadProgress.status === 'done') {
    return 100;
  }
  if (!item.size) {
    return uploadProgress.status === 'uploading' ? 0 : 100;
  }
  return Math.min(100, Math.max(0, Math.round((item.loaded / item.size) * 100)));
}

function formatUploadSpeed(bytesPerSecond) {
  if (!bytesPerSecond || bytesPerSecond < 1) {
    return '-';
  }
  return `${formatSize(bytesPerSecond)}/s`;
}

function readClipboard() {
  try {
    const raw = localStorage.getItem(CLIPBOARD_KEY);
    if (!raw) {
      return null;
    }
    return normalizeClipboardPayload(JSON.parse(raw));
  } catch {
    return null;
  }
}

function normalizeClipboardPayload(value) {
  if (!value?.sourceConnectionId || !Array.isArray(value.items) || value.items.length === 0) {
    return null;
  }
  const items = value.items
    .filter((item) => item?.path)
    .map((item) => ({
      path: item.path,
      isDir: Boolean(item.isDir),
    }));
  if (items.length === 0) {
    return null;
  }
  return {
    sourceConnectionId: value.sourceConnectionId,
    action: normalizeClipboardAction(value.action),
    items,
    createdAt: Number(value.createdAt) || Date.now(),
  };
}

function normalizeClipboardAction(action) {
  return CLIPBOARD_ACTIONS.has(action) ? action : 'copy';
}

function clearClipboardIfDeleted(deletedItems) {
  const pendingClipboard = normalizeClipboardPayload(clipboard.value);
  if (!pendingClipboard) {
    return;
  }
  const removed = deletedItems.some((deletedItem) =>
    pendingClipboard.items.some((item) => item.path === deletedItem.path || (deletedItem.isDir && isSameOrChildPath(item.path, deletedItem.path))),
  );
  if (!removed) {
    return;
  }
  localStorage.removeItem(CLIPBOARD_KEY);
  clipboard.value = null;
}

function onStorage(event) {
  if (event.key === CLIPBOARD_KEY) {
    clipboard.value = readClipboard();
  }
}

function onKeydown(event) {
  const key = event.key.toLowerCase();
  const currentEditor = activeEditor.value;
  if ((event.ctrlKey || event.metaKey) && key === 's' && currentEditor) {
    event.preventDefault();
    saveEditor(currentEditor);
    return;
  }
  if (event.key === 'Escape') {
    if (currentEditor?.closePrompt.visible) {
      cancelEditorClosePrompt(currentEditor);
      return;
    }
    hideContextMenu();
  }
  if (event.key === 'F5' && props.connectionId) {
    event.preventDefault();
    refresh(currentPath.value || initialPathForMode(props.workMode), { useCache: false });
  }
}

function startColumnResize(event, key) {
  resizeState = {
    key,
    startX: event.clientX,
    startWidth: columnWidths[key],
  };
  document.body.style.cursor = 'col-resize';
  document.body.style.userSelect = 'none';
  window.addEventListener('mousemove', onColumnResize);
  window.addEventListener('mouseup', stopColumnResize);
}

function onColumnResize(event) {
  if (!resizeState) {
    return;
  }
  const nextWidth = Math.max(MIN_COLUMN_WIDTH, resizeState.startWidth + event.clientX - resizeState.startX);
  columnWidths[resizeState.key] = nextWidth;
  applyColumnWidths();
}

function stopColumnResize() {
  if (!resizeState) {
    return;
  }
  resizeState = null;
  document.body.style.cursor = '';
  document.body.style.userSelect = '';
  window.removeEventListener('mousemove', onColumnResize);
  window.removeEventListener('mouseup', stopColumnResize);
}

function loadColumnWidths() {
  const values = {};
  for (const column of columns) {
    const raw = document.documentElement.style.getPropertyValue(`--file-col-${column.key}`);
    const parsed = Number.parseInt(raw, 10);
    values[column.key] = Number.isFinite(parsed) ? parsed : column.width;
  }
  return values;
}

function applyColumnWidths() {
  for (const column of columns) {
    document.documentElement.style.setProperty(`--file-col-${column.key}`, `${columnWidths[column.key]}px`);
  }
}

function normalizePath(value) {
  const raw = String(value || '').trim();
  if (!raw) {
    return '';
  }
  if (raw === ROOT_PATH || raw === HOME_PATH) {
    return raw;
  }
  if (raw.startsWith('~/')) {
    const parts = raw.slice(2).split('/').filter(Boolean);
    return parts.length ? `${HOME_PATH}/${parts.join('/')}` : HOME_PATH;
  }
  if (raw.startsWith(ROOT_PATH)) {
    const parts = raw.split('/').filter(Boolean);
    return parts.length ? `${ROOT_PATH}${parts.join('/')}` : ROOT_PATH;
  }
  return raw;
}

function parentPath(itemPath) {
  const normalized = normalizePath(itemPath);
  if (!normalized || normalized === ROOT_PATH || normalized === HOME_PATH) {
    return '';
  }
  if (normalized.startsWith('~/')) {
    const parts = normalized.slice(2).split('/').filter(Boolean);
    if (parts.length <= 1) {
      return HOME_PATH;
    }
    return `${HOME_PATH}/${parts.slice(0, -1).join('/')}`;
  }
  if (normalized.startsWith(ROOT_PATH)) {
    const parts = normalized.split('/').filter(Boolean);
    if (parts.length <= 1) {
      return ROOT_PATH;
    }
    return `${ROOT_PATH}${parts.slice(0, -1).join('/')}`;
  }
  return '';
}

function pathDepth(itemPath) {
  if (itemPath === ROOT_PATH || itemPath === HOME_PATH) {
    return 0;
  }
  if (itemPath.startsWith('~/')) {
    return itemPath.slice(2).split('/').filter(Boolean).length;
  }
  return itemPath.split('/').filter(Boolean).length;
}

function isSameOrChildPath(candidate, parent) {
  const cleanCandidate = normalizePath(candidate);
  const cleanParent = normalizePath(parent);
  if (!cleanCandidate || !cleanParent) {
    return false;
  }
  if (cleanCandidate === cleanParent) {
    return true;
  }
  if (cleanParent === ROOT_PATH) {
    return cleanCandidate.startsWith(ROOT_PATH);
  }
  return cleanCandidate.startsWith(`${cleanParent}/`);
}

function isPathVisible(itemPath) {
  if (itemPath === ROOT_PATH || itemPath === HOME_PATH) {
    return true;
  }

  let cursor = parentPath(itemPath);
  while (cursor) {
    if (pathMeta.value.get(cursor)?.collapsed) {
      return false;
    }
    cursor = parentPath(cursor);
  }
  return true;
}

function comparePaths(a, b) {
  if (a === b) {
    return 0;
  }
  if (a === ROOT_PATH) {
    return -1;
  }
  if (b === ROOT_PATH) {
    return 1;
  }
  const aHome = a === HOME_PATH || a.startsWith('~/');
  const bHome = b === HOME_PATH || b.startsWith('~/');
  if (aHome !== bHome) {
    return aHome ? 1 : -1;
  }
  return a.localeCompare(b, undefined, { sensitivity: 'base' });
}

function buildBreadcrumbs(itemPath) {
  const normalized = normalizePath(itemPath);
  if (!normalized) {
    return [];
  }
  if (normalized === HOME_PATH) {
    return [{ label: HOME_PATH, path: HOME_PATH }];
  }
  if (normalized.startsWith('~/')) {
    const parts = normalized.slice(2).split('/').filter(Boolean);
    const crumbs = [{ label: HOME_PATH, path: HOME_PATH }];
    for (let index = 0; index < parts.length; index += 1) {
      crumbs.push({
        label: parts[index],
        path: `${HOME_PATH}/${parts.slice(0, index + 1).join('/')}`,
      });
    }
    return crumbs;
  }
  if (normalized.startsWith(ROOT_PATH)) {
    const parts = normalized.split('/').filter(Boolean);
    const crumbs = [{ label: ROOT_PATH, path: ROOT_PATH }];
    for (let index = 0; index < parts.length; index += 1) {
      crumbs.push({
        label: parts[index],
        path: `${ROOT_PATH}${parts.slice(0, index + 1).join('/')}`,
      });
    }
    return crumbs;
  }
  return [{ label: normalized, path: normalized }];
}

function displayPath(itemPath) {
  const normalized = normalizePath(itemPath);
  if (normalized === ROOT_PATH || normalized === HOME_PATH) {
    return normalized;
  }
  if (normalized.startsWith('~/')) {
    const parts = normalized.slice(2).split('/').filter(Boolean);
    return parts.at(-1) || HOME_PATH;
  }
  const parts = normalized.split('/').filter(Boolean);
  return parts.at(-1) || normalized;
}

function downloadArchiveName(selected) {
  if (selected.length === 1) {
    return `${selected[0].name}.zip`;
  }
  return 'zshell-selected.zip';
}

function formatSize(size) {
  if (typeof size !== 'number') {
    return '-';
  }
  if (size < 1024) {
    return `${Math.max(0, Math.round(size))} B`;
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
