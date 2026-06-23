<template>
  <div class="file-manager-shell" @click="hideContextMenu">
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

    <div class="file-error">{{ errorMessage || '\u00A0' }}</div>

    <div v-if="connectionId" class="file-split" :class="{ 'nav-collapsed': navCollapsed }">
      <aside v-if="!navCollapsed" class="path-navigator" @contextmenu.prevent.stop="openBlankContextMenu">
        <div class="path-nav-head">
          <span>路径</span>
          <button class="path-nav-toggle" title="折叠" @click.stop="navCollapsed = true">‹</button>
        </div>

        <div
          v-for="node in treeNodes"
          :key="node.path"
          class="path-node-row"
          :style="{ paddingLeft: `${node.depth * 14 + 8}px` }"
        >
          <button
            class="node-arrow"
            :class="{ hidden: !node.hasChildren }"
            :disabled="!node.hasChildren"
            @click.stop="togglePathNode(node.path)"
          >
            {{ node.collapsed ? '›' : '⌄' }}
          </button>
          <button
            class="path-node-main"
            :class="{ active: node.path === currentPath, opened: node.opened }"
            @click.stop="openDir(node.path)"
            @contextmenu.prevent.stop="openBlankContextMenu"
          >
            <span class="node-label">{{ displayPath(node.path) }}</span>
            <span class="node-state" :class="{ opened: node.opened }">{{ node.opened ? '开' : '未' }}</span>
          </button>
        </div>
      </aside>

      <button v-else class="path-nav-rail" title="展开路径" @click.stop="navCollapsed = false">›</button>

      <section
        class="file-list-pane"
        @click="onPaneClick"
        @contextmenu.prevent="openBlankContextMenu"
      >
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
          <div class="file-row file-row-head">
            <span>名称</span>
            <span>大小</span>
            <span>修改时间</span>
          </div>
          <div
            v-for="(entry, index) in orderedEntries"
            :key="entry.path"
            class="file-row"
            :class="{ selected: isSelected(entry) }"
            @click.stop="selectEntry(entry, index, $event)"
            @dblclick.stop="openEntry(entry)"
            @contextmenu.prevent.stop="openEntryContextMenu($event, entry, index)"
          >
            <span class="name-cell" :class="{ dir: entry.isDir }">
              <span class="type-badge">{{ entry.isDir ? 'DIR' : 'FILE' }}</span>
              {{ entry.name }}
            </span>
            <span>{{ entry.isDir ? '-' : formatSize(entry.size) }}</span>
            <span>{{ formatTime(entry.modTime) }}</span>
          </div>
          <div v-if="orderedEntries.length === 0" class="empty-tip">当前目录为空。</div>
        </div>
      </section>
    </div>
    <div v-else class="empty-tip">连接后可浏览远程文件</div>

    <div
      v-if="contextMenu.visible"
      class="context-menu"
      :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
      @click.stop
    >
      <button :disabled="!connectionId || loading" @click="refreshFromMenu">刷新</button>
    </div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, onMounted, reactive, ref, watch } from 'vue';
import { downloadRemoteFile, listRemoteFiles } from '../services/apiClient';

const ROOT_PATH = '/';
const HOME_PATH = '~';

const props = defineProps({
  connectionId: {
    type: String,
    default: '',
  },
});

const entries = ref([]);
const currentPath = ref(HOME_PATH);
const loading = ref(false);
const errorMessage = ref('');
const navCollapsed = ref(false);
const selectedPaths = ref(new Set());
const pathMeta = ref(
  new Map([
    [ROOT_PATH, { opened: false, collapsed: false }],
    [HOME_PATH, { opened: false, collapsed: false }],
  ]),
);

const contextMenu = reactive({
  visible: false,
  x: 0,
  y: 0,
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

const statusText = computed(() => {
  if (!props.connectionId) {
    return '未连接';
  }
  if (loading.value) {
    return '读取中...';
  }
  return `${orderedEntries.value.length} 项`;
});

const breadcrumbs = computed(() => buildBreadcrumbs(currentPath.value));

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
  window.addEventListener('keydown', onKeydown);
});

onBeforeUnmount(() => {
  window.removeEventListener('keydown', onKeydown);
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
    rememberPath(HOME_PATH, { opened: requestedPath === HOME_PATH || resolvedPath === HOME_PATH });
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
  download(entry.path, entry.name);
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

function selectEntry(entry, index, event) {
  if (event.ctrlKey || event.metaKey) {
    const next = new Set(selectedPaths.value);
    if (next.has(entry.path)) {
      next.delete(entry.path);
    } else {
      next.add(entry.path);
    }
    selectedPaths.value = next;
    return;
  }

  selectedPaths.value = new Set([entry.path]);
}

function isSelected(entry) {
  return selectedPaths.value.has(entry.path);
}

function clearSelection() {
  selectedPaths.value = new Set();
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

function openEntryContextMenu(event, entry) {
  if (!selectedPaths.value.has(entry.path)) {
    selectedPaths.value = new Set([entry.path]);
  }
  openContextMenu(event);
}

function openBlankContextMenu(event) {
  openContextMenu(event);
}

function openContextMenu(event) {
  contextMenu.visible = true;
  contextMenu.x = Math.min(event.clientX, window.innerWidth - 150);
  contextMenu.y = Math.min(event.clientY, window.innerHeight - 70);
}

function hideContextMenu() {
  contextMenu.visible = false;
}

async function refreshFromMenu() {
  hideContextMenu();
  await refresh(currentPath.value || HOME_PATH);
}

function onKeydown(event) {
  if (event.key === 'Escape') {
    hideContextMenu();
  }
  if (event.key === 'F5' && props.connectionId) {
    event.preventDefault();
    refresh(currentPath.value || HOME_PATH);
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
    return `${HOME_PATH}/${raw.split('/').filter(Boolean).slice(1).join('/')}`;
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
  return itemPath;
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
