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
      <div class="path-input-completion" @click.stop @contextmenu.stop>
        <input
          class="path-input"
          :value="pathDraft"
          :disabled="!connectionId || loading"
          autocomplete="off"
          spellcheck="false"
          @input="onPathInput"
          @change="commitPathDraft"
          @blur="dismissPathCompletion"
          @keydown.enter.prevent.stop="commitPathDraft"
          @keydown.tab.prevent.stop="completePathDraft"
          @keydown.escape.prevent.stop="dismissPathCompletion"
        />
        <div
          v-if="pathCompletionVisible"
          class="path-completion-popover"
          role="listbox"
          aria-label="目录补全候选"
        >
          <button
            v-for="item in pathCompletion.items"
            :key="item.path"
            type="button"
            role="option"
            :title="item.path"
            @mousedown.prevent
            @click="openPathCompletionItem(item)"
          >
            <span class="path-completion-folder">◇</span>
            <span class="path-completion-name">{{ pathCompletionItemLabel(item) }}</span>
          </button>
          <div class="path-completion-footer">
            <span>{{ pathCompletionSummary }}</span>
            <span v-if="pathCompletion.total > pathCompletion.items.length">
              显示前 {{ pathCompletion.items.length }} 项
            </span>
          </div>
        </div>
      </div>
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
      <aside v-if="!navCollapsed" class="path-navigator">
        <div class="path-nav-head">
          <span>路径</span>
          <button class="path-nav-toggle" title="折叠导航" @click.stop="navCollapsed = true">‹</button>
        </div>

        <div
          ref="pathTreeViewport"
          class="path-tree-viewport"
          @scroll.passive="onTreeScroll"
          @contextmenu.prevent.stop="openBlankContextMenu($event)"
        >
          <div
            class="path-tree-virtual-space"
            role="tree"
            :style="{ height: `${treeVirtualHeight}px`, minWidth: `${treeContentWidth}px` }"
          >
            <div
              v-for="node in visibleTreeNodes"
              :key="node.path"
              class="path-node-row"
              :class="{
                active: node.path === treeSelectedPath,
                opened: node.opened,
              }"
              :style="{
                paddingLeft: `${node.depth * 14 + 8}px`,
                transform: `translateY(${node.virtualIndex * treeRowHeight}px)`,
              }"
              role="treeitem"
              tabindex="0"
              :aria-expanded="node.hasChildren ? String(!node.collapsed) : undefined"
              @click="activateTreeNode(node)"
              @keydown.enter.prevent.stop="activateTreeNode(node)"
              @keydown.space.prevent.stop="activateTreeNode(node)"
              @contextmenu.prevent.stop="openPathContextMenu($event, node.path)"
            >
              <span class="node-toggle-slot" aria-hidden="true">
                <ChevronIcon
                  v-if="node.hasChildren"
                  class="tree-chevron"
                  :direction="node.collapsed ? 'right' : 'down'"
                />
              </span>
              <div class="path-node-main" :class="{ opened: node.opened }">
                <input
                  v-if="renameState.surface === 'tree' && renameState.path === node.path"
                  :ref="(element) => setRenameInputRef(element, node.path)"
                  class="inline-rename-input tree-rename-input"
                  :value="renameState.name"
                  :disabled="renameState.saving"
                  @input="updateRenameName"
                  @click.stop
                  @dblclick.stop
                  @contextmenu.stop
                  @keydown.enter.prevent.stop="commitRename"
                  @keydown.escape.prevent.stop="cancelRename"
                  @blur="commitRename"
                />
                <span
                  v-else
                  class="node-label"
                  :data-tree-path="node.path"
                  :class="{ active: node.path === treeSelectedPath, current: node.path === currentPath }"
                  :title="node.path"
                >
                  {{ node.label }}
                </span>
              </div>
            </div>
          </div>
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

        <div ref="fileListViewport" class="file-list" @scroll.passive="onFileListScroll">
          <div class="file-row file-row-head" :style="{ gridTemplateColumns }">
            <span
              v-for="column in columns"
              :key="column.key"
              class="file-head-cell"
              :class="{ sorted: sortState.key === column.key, clicked: sortPulseKey === column.key }"
            >
              <button class="file-head-button" type="button" @click.stop="changeSort(column.key)">
                <span>{{ column.label }}</span>
                <ChevronIcon
                  v-if="sortState.key === column.key"
                  class="sort-arrow"
                  :direction="sortState.direction === 'asc' ? 'up' : 'down'"
                />
              </button>
              <span class="column-resizer" @mousedown.prevent.stop="startColumnResize($event, column.key)"></span>
            </span>
          </div>
          <div
            v-if="orderedEntries.length"
            class="file-list-virtual-space"
            :style="{ height: `${fileVirtualHeight}px`, minWidth: `${fileContentWidth}px` }"
          >
            <div
              v-for="item in visibleEntries"
              :key="item.entry.path"
              class="file-row file-row-virtual"
              :data-entry-path="item.entry.path"
              :class="{ selected: isSelected(item.entry), 'is-last': item.index === orderedEntries.length - 1 }"
              :style="{
                gridTemplateColumns,
                transform: `translateY(${item.index * fileRowHeight}px)`,
              }"
              draggable="true"
              @click.stop="selectEntry(item.entry, item.index, $event)"
              @dblclick.stop="openEntry(item.entry)"
              @contextmenu.prevent.stop="openEntryContextMenu($event, item.entry, item.index)"
              @dragstart="onRemoteDragStart($event, item.entry, item.index)"
            >
              <span class="name-cell" :class="{ dir: item.entry.isDir }">
                <input
                  v-if="renameState.surface === 'list' && renameState.path === item.entry.path"
                  :ref="(element) => setRenameInputRef(element, item.entry.path)"
                  class="inline-rename-input"
                  :value="renameState.name"
                  :disabled="renameState.saving"
                  @input="updateRenameName"
                  @click.stop
                  @dblclick.stop
                  @contextmenu.stop
                  @keydown.enter.prevent.stop="commitRename"
                  @keydown.escape.prevent.stop="cancelRename"
                  @blur="commitRename"
                />
                <template v-else>{{ item.entry.name }}</template>
              </span>
              <span>{{ item.entry.isDir ? '文件夹' : '文件' }}</span>
              <span>{{ item.entry.isDir ? '-' : formatSize(item.entry.size) }}</span>
              <span>{{ formatTime(item.entry.modTime) }}</span>
              <span>{{ item.entry.mode || '-' }}</span>
              <span>{{ item.entry.owner || '-' }}</span>
            </div>
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
            <button class="small-btn" :disabled="!editorDirty(editorWindow) || editorWindow.contentLoading || editorWindow.editorRuntimeState === 'rendering' || editorWindow.saving" @click="saveEditor(editorWindow)">保存</button>
            <button class="editor-window-control" type="button" title="最小化" :disabled="editorWindow.saving" @click="minimizeEditor(editorWindow)">_</button>
            <button class="editor-window-control" type="button" :title="editorWindow.windowState === 'normal' ? '最大化' : '还原'" :disabled="editorWindow.saving" @click="toggleMaximizeEditor(editorWindow)">
              {{ editorWindow.windowState === 'normal' ? '□' : '❐' }}
            </button>
            <button class="editor-window-control danger" type="button" title="关闭" :disabled="editorWindow.saving" @click="requestEditorClose(editorWindow)">×</button>
          </div>
        </header>

        <RemoteCodeEditor
          v-if="editorWindow.windowState !== 'minimized' && editorWindow.openProgress?.stage !== 'error'"
          v-model="editorWindow.content"
          :path="editorWindow.path"
          :append-chunks="editorWindow.contentChunks"
          :append-version="editorWindow.appendVersion"
          :streaming="editorWindow.contentLoading"
          :disabled="editorWindow.contentLoading || editorWindow.editorRuntimeState === 'rendering' || editorWindow.saving"
          :active="editorWindow.id === activeEditorId"
          @focus="activateEditor(editorWindow.id)"
          @save="saveEditor(editorWindow)"
          @state="setEditorRuntimeState(editorWindow, $event)"
        />

        <div
          v-if="
            editorWindow.contentLoading &&
            editorWindow.windowState !== 'minimized'
          "
          class="remote-editor-open-progress"
        >
          <section class="remote-editor-open-card">
            <div class="remote-editor-open-head">
              <strong>{{ editorOpenStatus(editorWindow) }}</strong>
              <span>
                {{
                  editorWindow.openProgress?.totalBytes > 0
                    ? `${Math.min(100, Math.max(0, Math.round(((editorWindow.openProgress?.loadedBytes || 0) / editorWindow.openProgress.totalBytes) * 100)))}%`
                    : '读取中'
                }}
              </span>
            </div>
            <div class="progress-track">
              <span
                :style="{
                  width: `${
                    editorWindow.openProgress?.stage === 'done'
                      ? 100
                      : editorWindow.openProgress?.totalBytes > 0
                        ? Math.min(100, Math.max(0, Math.round(((editorWindow.openProgress?.loadedBytes || 0) / editorWindow.openProgress.totalBytes) * 100)))
                        : editorWindow.openProgress?.stage === 'preparing'
                          ? 6
                          : 12
                  }%`,
                }"
              ></span>
            </div>
            <div class="remote-editor-open-meta">
              <span>{{ editorWindow.openProgress?.message || '正在读取远程文件' }}</span>
              <span>
                {{
                  formatEditorBytes(
                    editorWindow.openProgress?.loadedBytes || 0,
                    editorWindow.openProgress?.totalBytes || editorWindow.size || 0,
                  )
                }}
              </span>
            </div>
          </section>
        </div>

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
        ref="contextMenuElement"
        class="context-menu file-context-menu"
        :data-context-kind="contextMenu.targetKind"
        :style="{ left: `${contextMenu.x}px`, top: `${contextMenu.y}px` }"
        @click.stop
        @contextmenu.prevent.stop
      >
        <template v-if="contextMenu.targetKind === 'directory'">
          <button :disabled="loading || contextPath === currentPath" @click="openContextDirectory">打开</button>
          <button :disabled="loading" @click="expandContextDirectory">展开目录树</button>
          <button :disabled="!connectionId || loading" @click="refreshFromMenu">刷新</button>

          <div class="context-menu-separator"></div>
          <button :disabled="loading" @click="downloadContextItem">下载文件夹</button>

          <div class="context-menu-separator"></div>
          <button :disabled="!connectionId || uploading" @click="chooseFilesFromMenu">上传文件到此处...</button>
          <button :disabled="!connectionId || uploading" @click="chooseFolderFromMenu">上传文件夹到此处...</button>
          <button :disabled="!canPaste || loading" @click="pasteClipboardFromMenu">粘贴到此处</button>

          <div class="context-menu-separator"></div>
          <button :disabled="!contextCanMutate || loading" @click="copySelectionFromMenu">复制</button>
          <button class="danger" :disabled="!contextCanMutate || loading" @click="cutSelectionFromMenu">剪切</button>

          <div class="context-menu-separator"></div>
          <button :disabled="!contextCanMutate || loading" @click="startRenameFromMenu">重命名</button>
          <button class="danger" :disabled="!canDeleteFromMenu || loading" @click="deleteFromMenu">删除</button>

          <div class="context-menu-separator"></div>
          <button :disabled="!contextPath" @click="copyPathFromMenu">复制路径</button>
        </template>

        <template v-else>
          <button :disabled="loading" @click="runContextFileOpenAction('textEdit')">编辑</button>
          <button disabled title="后续版本提供">打开方式</button>

          <div class="context-menu-separator"></div>
          <button :disabled="loading" @click="downloadContextItem">下载</button>

          <div class="context-menu-separator"></div>
          <button :disabled="loading" @click="copySelectionFromMenu">复制</button>
          <button class="danger" :disabled="loading" @click="cutSelectionFromMenu">剪切</button>

          <div class="context-menu-separator"></div>
          <button :disabled="loading" @click="startRenameFromMenu">重命名</button>
          <button class="danger" :disabled="loading" @click="deleteFromMenu">删除</button>

          <div class="context-menu-separator"></div>
          <button :disabled="!contextPath" @click="copyPathFromMenu">复制路径</button>
        </template>
      </div>
    </Teleport>
  </div>
</template>

<script setup>
import ChevronIcon from './ChevronIcon.vue';
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
  activateTreeNode,
  activateEditor,
  activeEditorId,
  breadcrumbs,
  cancelRename,
  cancelEditorClosePrompt,
  canDeleteFromMenu,
  canPaste,
  changeSort,
  chooseFilesFromMenu,
  chooseFolderFromMenu,
  columns,
  commitPathDraft,
  completePathDraft,
  contextCanOpenDirectory,
  contextCanMutate,
  contextEntry,
  contextMenu,
  contextMenuElement,
  contextPath,
  copyPathFromMenu,
  copySelectionFromMenu,
  currentPath,
  commitRename,
  cutSelectionFromMenu,
  deleteFromMenu,
  discardEditorFromPrompt,
  dismissPathCompletion,
  displayPath,
  downloadContextEntry,
  downloadContextItem,
  downloadSelectionFromMenu,
  dragOver,
  editorDirty,
  editorMeta,
  editorOpenStatus,
  editors,
  editorStatus,
  editorTitle,
  editorWindowStyle,
  errorMessage,
  expandContextDirectory,
  expandUploadPanel,
  fileInput,
  fileContentWidth,
  fileListViewport,
  fileOpenActions,
  fileRowHeight,
  fileVirtualHeight,
  folderInput,
  formatSize,
  formatTime,
  formatEditorBytes,
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
  onFileListScroll,
  onFolderPickerUpload,
  onPaneClick,
  onPathInput,
  onRemoteDragStart,
  onShellClick,
  onTreeScroll,
  openBlankContextMenu,
  openContextDirectory,
  openDir,
  openEntry,
  openEntryContextMenu,
  openHistoryPath,
  openPathCompletionItem,
  openPathContextMenu,
  orderedEntries,
  pasteClipboardFromMenu,
  pathDraft,
  pathCompletion,
  pathCompletionItemLabel,
  pathCompletionSummary,
  pathCompletionVisible,
  pathHistory,
  pathHistoryButton,
  pathHistoryListRef,
  pathHistoryOpen,
  pathHistoryStyle,
  pathTreeViewport,
  refreshFromMenu,
  renameState,
  requestEditorClose,
  runContextFileOpenAction,
  saveAndCloseEditorFromPrompt,
  saveEditor,
  saveEditorFromPrompt,
  selectEntry,
  setRenameInputRef,
  setEditorRuntimeState,
  sortPulseKey,
  sortState,
  startColumnResize,
  startEditorDrag,
  startRenameFromMenu,
  statusText,
  toggleMaximizeEditor,
  togglePathHistory,
  treeRowHeight,
  treeSelectedPath,
  treeNodes,
  treeContentWidth,
  treeVirtualHeight,
  uploadDetail,
  uploadFilePercent,
  uploadPercent,
  uploadProgress,
  uploadTitle,
  uploading,
  updateRenameName,
  visibleEntries,
  visibleTreeNodes,
} = useFileManager(props);
</script>
