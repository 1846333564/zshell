<template>
  <main class="app-shell" @contextmenu="handleAppContextMenu">
    <header class="app-topbar">
      <div class="topbar-drag-region">
        <div class="brand-lockup">
          <span class="brand-glyph">z</span>
          <strong>zShell</strong>
        </div>

        <nav class="app-menu-strip" aria-label="应用菜单">
          <div class="app-menu-item">
            <button class="app-menu-button" type="button">zShell</button>
            <div class="app-menu-dropdown">
              <button type="button" @click="showConnectHome">连接首页</button>
              <button type="button" @click="showAboutDialog">关于 zShell</button>
              <button type="button" @click="closeWindow">退出</button>
            </div>
          </div>

          <div class="app-menu-item">
            <button class="app-menu-button" type="button">配置管理</button>
            <div class="app-menu-dropdown">
              <button type="button" @click="showConnectHome">连接配置</button>
              <button type="button" disabled>导入配置</button>
              <button type="button" disabled>导出配置</button>
            </div>
          </div>

          <div class="app-menu-item">
            <button class="app-menu-button" type="button">UI管理</button>
            <div class="app-menu-dropdown">
              <button type="button" @click="resetUiScale">重置缩放</button>
              <button type="button" @click="showThemeDialog">主题设置</button>
              <button type="button" disabled>布局设置</button>
            </div>
          </div>
        </nav>
      </div>

      <div class="window-controls" aria-label="窗口控制">
        <button type="button" title="最小化" @click="minimizeWindow">-</button>
        <button type="button" title="最大化/还原" @click="toggleMaximizeWindow">□</button>
        <button type="button" class="close" title="关闭" @click="closeWindow">×</button>
      </div>
    </header>

    <section class="desktop-layout" :class="{ 'home-layout': !activeSession }">
      <aside v-if="activeSession" class="monitor-sidebar panel">
        <MonitorPanel :session="activeSession" />
      </aside>

      <section class="main-workspace">
        <header class="connection-tabbar panel">
          <button
            v-for="item in sessions"
            :key="item.connectionId"
            class="connection-tab"
            :class="{ active: item.connectionId === activeSessionId }"
            @click="activateSession(item.connectionId)"
          >
            <span class="tab-title">{{ item.connectionName }}</span>
            <button class="tab-close" @click.stop="closeSession(item.connectionId)">x</button>
          </button>
          <button class="connection-tab add" title="新建连接" @click="showConnectHome">+</button>
        </header>

        <section v-if="!activeSession" class="connect-workspace panel">
          <div class="connect-columns">
            <section class="history-panel flat">
              <div class="history-head">
                <h3>已保存连接</h3>
                <span>{{ configLoading ? '加载中' : `${savedConnections.length} 条` }}</span>
              </div>

              <div v-if="savedConnections.length === 0" class="empty-tip">
                暂无保存的连接。
              </div>

              <div v-else class="history-list">
                <article
                  v-for="item in savedConnections"
                  :key="item.id"
                  class="history-item"
                  :class="{ editing: item.id === editingConnectionId }"
                >
                  <div class="history-meta">
                    <strong>{{ item.name }}</strong>
                    <span>{{ item.host }}:{{ item.port }} · {{ item.username }} · {{ authLabel(item.authMethod) }} · {{ workModeLabel(item.workMode) }}</span>
                  </div>
                  <div class="history-actions">
                    <button class="mini-btn" :disabled="busy || configLoading || !appReady" @click="connectFromSaved(item)">连接</button>
                    <button class="mini-btn" :disabled="busy || configLoading || !appReady" @click="editSavedConnection(item)">编辑</button>
                    <button class="mini-btn danger" :disabled="busy || configLoading || !appReady" @click="removeSavedConnection(item)">删除</button>
                  </div>
                </article>
              </div>
            </section>

            <section class="connect-form-pane">
              <ConnectionForm
                :busy="busy || configLoading || !appReady"
                :error="connectError || configError"
                :initial-value="draftConnection"
                :mode="editingConnectionId ? 'edit' : 'create'"
                :submit-label="editingConnectionId ? '保存并连接' : '保存并连接'"
                :title="editingConnectionId ? '编辑连接' : '连接配置'"
                @connect="handleConnect"
              />
            </section>
          </div>
        </section>

        <section v-else class="active-workspace">
          <div class="terminal-band panel" :style="{ height: `calc(${consoleHeightPercent}% - 5px)` }">
            <TerminalTabs
              :key="activeSession.connectionId"
              :connection-id="activeSession.connectionId"
              :connection-name="activeSession.connectionName"
              :terminal-font-size="terminalFontSize"
              @terminal-font-size-change="handleTerminalFontSizeChange"
            />
          </div>

          <div class="splitter" title="拖拽调整控制台高度" @mousedown.prevent="startDrag"></div>

          <div class="file-band panel">
            <FileManager
              :key="`${activeSession.connectionId}:${activeSession.workMode}`"
              :connection-id="activeSession.connectionId"
              :work-mode="activeSession.workMode"
              :hardware="activeSession.hardware"
            />
          </div>
        </section>
      </section>
    </section>

    <Teleport to="body">
      <div v-if="aboutDialog.visible" class="modal-backdrop" @click.self="hideAboutDialog">
        <section class="app-dialog about-dialog" @click.stop>
          <header class="dialog-head">
            <div>
              <strong>关于 zShell</strong>
              <span>版本 {{ appInfo.version || '0.0.1' }}</span>
            </div>
            <button type="button" class="dialog-close" @click="hideAboutDialog">×</button>
          </header>

          <div class="dialog-body">
            <p>属于{{ appInfo.company || '重庆创翼科技有限公司' }}，开发者{{ appInfo.developer || 'zly' }}，{{ appInfo.channel || '暂时内测版' }}。</p>
          </div>

          <footer class="dialog-actions">
            <button class="small-btn" type="button" :disabled="updateDialog.status === 'checking' || updateDialog.status === 'applying' || updateDialog.status === 'stopping'" @click="checkUpdatesFromAbout">
              检查更新
            </button>
            <button class="small-btn" type="button" @click="hideAboutDialog">关闭</button>
          </footer>
        </section>
      </div>

      <div v-if="themeDialog.visible" class="modal-backdrop" @click.self="cancelThemeDialog">
        <section class="app-dialog theme-dialog" @click.stop>
          <header class="dialog-head">
            <div>
              <strong>主题设置</strong>
              <span>当前 {{ activeTheme.name }}</span>
            </div>
            <button type="button" class="dialog-close" :disabled="themeDialog.saving" @click="cancelThemeDialog">×</button>
          </header>

          <div class="dialog-body theme-dialog-body">
            <section class="theme-grid" aria-label="主题列表">
              <button
                v-for="theme in themeOptions"
                :key="theme.id"
                type="button"
                class="theme-choice"
                :class="{ active: themeDialog.draftKey === theme.id }"
                @click="selectThemeOption(theme.id)"
              >
                <span class="theme-choice-name">{{ theme.name }}</span>
                <span class="theme-swatches" aria-hidden="true">
                  <i v-for="color in theme.preview" :key="color" :style="{ background: color }"></i>
                </span>
              </button>
            </section>

            <section class="custom-theme-panel">
              <div class="custom-theme-head">
                <strong>自定义颜色</strong>
                <button class="mini-btn" type="button" @click="resetCustomTheme">恢复默认</button>
              </div>

              <div class="theme-color-grid">
                <label v-for="field in themeColorFields" :key="field.key" class="theme-color-row">
                  <span>{{ field.label }}</span>
                  <input
                    type="color"
                    :value="themeDialog.draftCustomTheme[field.key]"
                    @input="setCustomThemeColor(field.key, $event.target.value)"
                  />
                  <code>{{ themeDialog.draftCustomTheme[field.key] }}</code>
                </label>
              </div>
            </section>

            <p v-if="themeDialog.error" class="dialog-error">{{ themeDialog.error }}</p>
          </div>

          <footer class="dialog-actions">
            <button class="small-btn" type="button" :disabled="themeDialog.saving" @click="saveThemeDialog">保存主题</button>
            <button class="small-btn" type="button" :disabled="themeDialog.saving" @click="cancelThemeDialog">取消</button>
          </footer>
        </section>
      </div>

      <div v-if="updateDialog.visible" class="modal-backdrop" @click.self="closeUpdateDialog">
        <section class="app-dialog update-dialog" @click.stop>
          <header class="dialog-head">
            <div>
              <strong>{{ updateDialog.title }}</strong>
              <span>{{ updateDialog.subtitle }}</span>
            </div>
            <button type="button" class="dialog-close" :disabled="updateDialog.status === 'applying' || updateDialog.status === 'stopping'" @click="closeUpdateDialog">×</button>
          </header>

          <div class="dialog-body">
            <p>{{ updateDialog.message }}</p>
            <div v-if="updateDialog.showProgress" class="update-progress-panel">
              <div class="update-progress-head">
                <strong>{{ updateDialog.progressLabel }}</strong>
                <span>{{ updateDialog.progress }}%</span>
              </div>
              <div class="progress-track update-progress-track">
                <span :style="{ width: `${updateDialog.progress}%` }"></span>
              </div>
              <div class="update-progress-detail">
                <span>{{ updateDialog.detail || '等待下一步...' }}</span>
                <span>{{ updateDialog.transferText }}</span>
              </div>
              <div v-if="updateDialog.logs.length" class="update-progress-log">
                <div v-for="item in updateDialog.logs" :key="item.id" class="update-progress-log-row">
                  <span>{{ item.time }}</span>
                  <strong>{{ item.text }}</strong>
                </div>
              </div>
            </div>
            <pre v-if="updateDialog.notes" class="release-notes">{{ updateDialog.notes }}</pre>
            <p v-if="updateDialog.error" class="dialog-error">{{ updateDialog.error }}</p>
          </div>

          <footer class="dialog-actions">
            <button
              v-if="updateDialog.available"
              class="small-btn"
              type="button"
              :disabled="updateDialog.status === 'applying' || updateDialog.status === 'stopping'"
              @click="confirmApplyUpdate"
            >
              确认更新
            </button>
            <button v-if="updateDialog.canStop" class="small-btn danger" type="button" @click="stopApplyUpdate">停止</button>
            <button v-if="updateDialog.releaseUrl" class="small-btn" type="button" @click="openReleasePage">打开下载页</button>
            <button class="small-btn" type="button" :disabled="updateDialog.status === 'applying' || updateDialog.status === 'stopping'" @click="closeUpdateDialog">关闭</button>
          </footer>
        </section>
      </div>
    </Teleport>
  </main>
</template>

<script setup>
import ConnectionForm from './components/ConnectionForm.vue';
import FileManager from './components/FileManager.vue';
import MonitorPanel from './components/MonitorPanel.vue';
import TerminalTabs from './components/TerminalTabs.vue';
import { useAppController } from './composables/app/useAppController';

const {
  activeSession,
  activeSessionId,
  activeTheme,
  appReady,
  appInfo,
  aboutDialog,
  busy,
  cancelThemeDialog,
  checkUpdatesFromAbout,
  closeSession,
  closeUpdateDialog,
  closeWindow,
  configError,
  configLoading,
  connectError,
  connectFromSaved,
  consoleHeightPercent,
  confirmApplyUpdate,
  draftConnection,
  editSavedConnection,
  editingConnectionId,
  handleAppContextMenu,
  handleConnect,
  handleTerminalFontSizeChange,
  hideAboutDialog,
  minimizeWindow,
  openReleasePage,
  removeSavedConnection,
  resetUiScale,
  resetCustomTheme,
  saveThemeDialog,
  savedConnections,
  selectThemeOption,
  sessions,
  setCustomThemeColor,
  showAboutDialog,
  showConnectHome,
  showThemeDialog,
  startDrag,
  stopApplyUpdate,
  terminalFontSize,
  themeColorFields,
  themeDialog,
  themeOptions,
  toggleMaximizeWindow,
  updateDialog,
  activateSession,
  authLabel,
  workModeLabel,
} = useAppController();
</script>
