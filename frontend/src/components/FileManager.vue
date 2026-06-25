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
            <span
              v-for="column in columns"
              :key="column.key"
              class="file-head-cell"
              :class="{ sorted: sortState.key === column.key, clicked: sortPulseKey === column.key }"
            >
              <button class="file-head-button" type="button" @click.stop="changeSort(column.key)">
                <span>{{ column.label }}</span>
                <span class="sort-arrow">{{ sortArrow(column.key) }}</span>
              </button>
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
        <div ref="pathHistoryListRef" class="path-history-list">
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

        <RemoteCodeEditor
          v-if="editorWindow.windowState !== 'minimized'"
          v-model="editorWindow.content"
          :path="editorWindow.path"
          :disabled="editorWindow.loading || editorWindow.saving"
          :active="editorWindow.id === activeEditorId"
          @focus="activateEditor(editorWindow.id)"
          @save="saveEditor(editorWindow)"
          @state="setEditorRuntimeState(editorWindow, $event)"
        />

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
import RemoteCodeEditor from './RemoteCodeEditor.vue';
import { useFileManager } from './useFileManager';

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

const {
  activeEditorId,
  breadcrumbs,
  canDeleteFromMenu,
  canPaste,
  changeSort,
  chooseFilesFromMenu,
  chooseFolderFromMenu,
  columns,
  commitPathDraft,
  completePathDraft,
  contextCanOpenDirectory,
  contextEntry,
  contextMenu,
  contextPath,
  copyPathFromMenu,
  copySelectionFromMenu,
  currentPath,
  cutSelectionFromMenu,
  deleteFromMenu,
  discardEditorFromPrompt,
  displayPath,
  downloadContextEntry,
  downloadSelectionFromMenu,
  dragOver,
  editorDirty,
  editorMeta,
  editors,
  editorStatus,
  editorTitle,
  editorWindowStyle,
  errorMessage,
  expandUploadPanel,
  fileInput,
  fileOpenActions,
  folderInput,
  formatSize,
  formatTime,
  formatUploadSpeed,
  gridTemplateColumns,
  hasSelection,
  isSelected,
  loading,
  minimizeEditor,
  navCollapsed,
  onDragLeave,
  onDragOver,
  onDropUpload,
  onFilePickerUpload,
  onFolderPickerUpload,
  onPaneClick,
  onPathInput,
  onRemoteDragStart,
  onShellClick,
  openBlankContextMenu,
  openContextDirectory,
  openDir,
  openEntry,
  openEntryContextMenu,
  openHistoryPath,
  openPathContextMenu,
  orderedEntries,
  pasteClipboardFromMenu,
  pathDraft,
  pathHistory,
  pathHistoryButton,
  pathHistoryListRef,
  pathHistoryOpen,
  pathHistoryStyle,
  refreshFromMenu,
  requestEditorClose,
  runContextFileOpenAction,
  saveAndCloseEditorFromPrompt,
  saveEditor,
  saveEditorFromPrompt,
  selectEntry,
  setEditorRuntimeState,
  sortArrow,
  sortPulseKey,
  sortState,
  startColumnResize,
  startEditorDrag,
  statusText,
  toggleMaximizeEditor,
  togglePathHistory,
  togglePathNode,
  treeNodes,
  uploadDetail,
  uploadFilePercent,
  uploadPercent,
  uploadProgress,
  uploadTitle,
} = useFileManager(props);
</script>
