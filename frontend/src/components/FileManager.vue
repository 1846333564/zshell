<template>
  <div class="file-manager-shell" @click="hideContextMenu" @contextmenu.prevent.stop="openBlankContextMenu($event)">
    <div class="file-path-row">
      <input
        class="path-input"
        :value="currentPath"
        :disabled="!connectionId || loading"
        @change="onPathChange"
        @click.stop
      />
      <span class="hint">{{ statusText }}</span>
    </div>

    <input ref="fileInput" class="hidden-file-input" type="file" multiple @change="onFilePickerUpload" />
    <input ref="folderInput" class="hidden-file-input" type="file" multiple webkitdirectory directory @change="onFolderPickerUpload" />

    <div class="file-error">{{ errorMessage || '\u00A0' }}</div>

    <div
      v-if="uploadProgress.visible"
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
const CLIPBOARD_KEY = 'zshell.remote-file.clipboard.v1';
const MIN_COLUMN_WIDTH = 70;
const UPLOAD_CLOSE_DELAY_MS = 1200;
const DEFAULT_FILE_OPEN_ACTION = 'textEdit';
const DEFAULT_EDITOR_WIDTH = 980;
const DEFAULT_EDITOR_HEIGHT = 660;
const MIN_EDITOR_TOP = 48;
const CLIPBOARD_ACTIONS = new Set(['copy', 'move']);

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
});

const entries = ref([]);
const currentPath = ref(HOME_PATH);
const loading = ref(false);
const uploading = ref(false);
const dragOver = ref(false);
const errorMessage = ref('');
const navCollapsed = ref(false);
const selectedPaths = ref(new Set());
const lastSelectedIndex = ref(-1);
const clipboard = ref(readClipboard());
const fileInput = ref(null);
const folderInput = ref(null);
const pendingUploadPath = ref('');
const columnWidths = reactive(loadColumnWidths());
const uploadProgress = reactive({
  visible: false,
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
  new Map([
    [ROOT_PATH, { opened: false, collapsed: false }],
    [HOME_PATH, { opened: false, collapsed: false }],
  ]),
);

let resizeState = null;
let editorDragState = null;
let uploadCloseTimer = null;
let editorZIndex = 900;

const contextMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
  entry: null,
  targetPath: '',
  targetKind: 'blank',
});

watch(
  () => props.connectionId,
  async (value) => {
    clearSelection();
    resetPathState();
    if (!value) {
      entries.value = [];
      currentPath.value = HOME_PATH;
      errorMessage.value = '';
      return;
    }

    await refresh(HOME_PATH);
  },
  { immediate: true },
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
const contextPath = computed(() => contextMenu.entry?.path || contextMenu.targetPath || currentPath.value || HOME_PATH);
const contextTargetDir = computed(() => (contextMenu.entry?.isDir ? contextMenu.entry.path : contextMenu.targetPath || currentPath.value || HOME_PATH));
const gridTemplateColumns = computed(() => columns.map((column) => `${columnWidths[column.key]}px`).join(' '));
const activeEditor = computed(() => editors.value.find((item) => item.id === activeEditorId.value) || null);

const statusText = computed(() => {
  if (!props.connectionId) {
    return '未连接';
  }
  if (uploading.value) {
    return `上传 ${uploadPercent.value}% · ${formatUploadSpeed(uploadProgress.speed)}`;
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
    return '已完成，稍后自动关闭';
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
  stopColumnResize();
  stopEditorDrag();
});

async function refresh(targetPath = currentPath.value || HOME_PATH) {
  if (!props.connectionId) {
    return;
  }

  const requestedPath = targetPath || HOME_PATH;
  loading.value = true;
  errorMessage.value = '';

  try {
    const result = await listRemoteFiles(props.connectionId, requestedPath);
    const resolvedPath = result.path || requestedPath;
    currentPath.value = resolvedPath;
    entries.value = Array.isArray(result.entries) ? result.entries : [];

    rememberPath(ROOT_PATH);
    if (requestedPath === HOME_PATH || requestedPath.startsWith('~/')) {
      rememberPath(HOME_PATH, { opened: true });
    }
    rememberParentChain(resolvedPath);
    rememberPath(resolvedPath, { opened: true, collapsed: false });
    for (const entry of entries.value) {
      if (entry.isDir) {
        rememberParentChain(entry.path);
        rememberPath(entry.path, { opened: false });
      }
    }
    keepExistingSelection();
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '读取目录失败';
  } finally {
    loading.value = false;
  }
}

function resetPathState() {
  pathMeta.value = new Map([
    [ROOT_PATH, { opened: false, collapsed: false }],
    [HOME_PATH, { opened: false, collapsed: false }],
  ]);
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

function onPathChange(event) {
  const value = event.target.value?.trim();
  if (!value) {
    event.target.value = currentPath.value;
    return;
  }
  clearSelection();
  refresh(value);
}

function chooseFiles(targetPath = contextTargetDir.value) {
  pendingUploadPath.value = targetPath || currentPath.value || HOME_PATH;
  fileInput.value?.click();
}

function chooseFolder(targetPath = contextTargetDir.value) {
  pendingUploadPath.value = targetPath || currentPath.value || HOME_PATH;
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
  await uploadPickedFiles(files, [], pendingUploadPath.value || currentPath.value || HOME_PATH);
  event.target.value = '';
}

async function onFolderPickerUpload(event) {
  const files = Array.from(event.target.files || []);
  const directories = deriveDirectoriesFromFiles(files);
  await uploadPickedFiles(files, directories, pendingUploadPath.value || currentPath.value || HOME_PATH);
  event.target.value = '';
}

async function uploadPickedFiles(files, directories = [], targetPath = currentPath.value || HOME_PATH) {
  const items = files.map((file) => ({
    file,
    relativePath: file.webkitRelativePath || file.name,
  }));
  await uploadItems(items, directories, targetPath);
}

async function uploadItems(items, directories = [], targetPath = currentPath.value || HOME_PATH) {
  if (!props.connectionId || (items.length === 0 && directories.length === 0)) {
    return;
  }

  uploading.value = true;
  errorMessage.value = '';
  startUploadProgress(items, directories, targetPath || currentPath.value || HOME_PATH);
  let succeeded = false;

  try {
    await uploadRemoteItems(props.connectionId, targetPath || currentPath.value || HOME_PATH, items, directories, onUploadProgress);
    succeeded = true;
    markUploadComplete();
    await refresh(currentPath.value || HOME_PATH);
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
  await uploadItems(files, directories, currentPath.value || HOME_PATH);
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
    await refresh(currentPath.value || HOME_PATH);
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

function openEntryContextMenu(event, entry, index) {
  if (!selectedPaths.value.has(entry.path)) {
    selectedPaths.value = new Set([entry.path]);
    lastSelectedIndex.value = index;
  }
  openContextMenu(event, {
    entry,
    targetPath: entry.isDir ? entry.path : currentPath.value || HOME_PATH,
    targetKind: entry.isDir ? 'directory' : 'file',
  });
}

function openPathContextMenu(event, targetPath) {
  openContextMenu(event, { targetPath, targetKind: 'directory' });
}

function openBlankContextMenu(event) {
  openContextMenu(event, { targetPath: currentPath.value || HOME_PATH, targetKind: 'blank' });
}

function openContextMenu(event, target = {}) {
  const position = viewportContextMenuPosition(event, { width: 220, height: 430 });
  contextMenu.visible = true;
  contextMenu.entry = target.entry || null;
  contextMenu.targetPath = target.targetPath || currentPath.value || HOME_PATH;
  contextMenu.targetKind = target.targetKind || 'blank';
  contextMenu.x = position.x;
  contextMenu.y = position.y;
}

function hideContextMenu() {
  contextMenu.visible = false;
}

function onGlobalPointerDown(event) {
  if (!contextMenu.visible) {
    return;
  }
  const target = event.target;
  if (target instanceof Element && target.closest('.file-context-menu')) {
    return;
  }
  hideContextMenu();
}

async function refreshFromMenu() {
  const target = contextMenu.targetPath || currentPath.value || HOME_PATH;
  hideContextMenu();
  await refresh(target);
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
      targetPath: targetPath || currentPath.value || HOME_PATH,
      action: pendingClipboard.action,
      items: pendingClipboard.items,
    });
    if (pendingClipboard.action === 'move') {
      localStorage.removeItem(CLIPBOARD_KEY);
      clipboard.value = null;
    }
    await refresh(currentPath.value || HOME_PATH);
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

  const loaded = Number(progress?.loaded) || 0;
  const total = Number(progress?.total) || 0;
  let ratio = 0;
  if (progress?.lengthComputable && total > 0) {
    ratio = loaded / total;
  } else if (uploadProgress.totalBytes > 0) {
    ratio = loaded / uploadProgress.totalBytes;
  }
  ratio = Math.min(1, Math.max(0, ratio));

  const loadedBytes = uploadProgress.totalBytes > 0 ? Math.round(uploadProgress.totalBytes * ratio) : loaded;
  uploadProgress.loadedBytes = Math.min(uploadProgress.totalBytes || loadedBytes, loadedBytes);
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
  uploadProgress.message = message || '上传失败';
}

function scheduleUploadPanelClose() {
  clearUploadCloseTimer();
  uploadCloseTimer = window.setTimeout(() => {
    uploadProgress.visible = false;
    uploadCloseTimer = null;
  }, UPLOAD_CLOSE_DELAY_MS);
}

function clearUploadCloseTimer() {
  if (uploadCloseTimer) {
    window.clearTimeout(uploadCloseTimer);
    uploadCloseTimer = null;
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
    refresh(currentPath.value || HOME_PATH);
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
