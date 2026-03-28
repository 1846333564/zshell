<template>
  <div class="terminal-tabs-shell">
    <div class="tab-bar">
      <div class="tab-items">
        <button
          v-for="tab in tabs"
          :key="tab.id"
          class="tab-btn"
          :class="{ active: tab.id === activeTabId }"
          @click="activateTab(tab.id)"
        >
          {{ tab.title }}
        </button>
      </div>
      <div class="tab-actions">
        <button class="small-btn" @click="createTab">+ 新标签</button>
        <button class="small-btn danger" @click="closeActiveTab" :disabled="tabs.length <= 1">关闭当前</button>
      </div>
    </div>

    <div class="tab-panels">
      <TerminalPanel
        v-for="tab in tabs"
        :key="tab.id"
        v-show="tab.id === activeTabId"
        :connection-id="connectionId"
        :connection-name="connectionName"
        :tab-title="tab.title"
        :active="tab.id === activeTabId"
      />
    </div>
  </div>
</template>

<script setup>
import { ref } from 'vue';
import TerminalPanel from './TerminalPanel.vue';

const props = defineProps({
  connectionId: {
    type: String,
    required: true,
  },
  connectionName: {
    type: String,
    default: '默认连接',
  },
});

const tabs = ref([{ id: crypto.randomUUID(), title: '终端 1' }]);
const activeTabId = ref(tabs.value[0].id);

function activateTab(id) {
  activeTabId.value = id;
}

function createTab() {
  const nextIndex = tabs.value.length + 1;
  const tab = { id: crypto.randomUUID(), title: `终端 ${nextIndex}` };
  tabs.value.push(tab);
  activeTabId.value = tab.id;
}

function closeActiveTab() {
  if (tabs.value.length <= 1) {
    return;
  }

  const index = tabs.value.findIndex((tab) => tab.id === activeTabId.value);
  if (index === -1) {
    return;
  }

  tabs.value.splice(index, 1);
  const next = tabs.value[Math.max(0, index - 1)];
  activeTabId.value = next.id;
}
</script>
