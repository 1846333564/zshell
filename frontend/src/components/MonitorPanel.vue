<template>
  <div class="monitor-panel">
    <div class="monitor-identity">
      <span>{{ session?.host || '-' }}</span>
      <span>{{ session?.port || '-' }}</span>
      <span>{{ session?.username || '-' }}</span>
    </div>

    <div class="monitor-actions">
      <button class="small-btn" :disabled="!session || loading" @click="refresh">刷新</button>
      <span>{{ statusText }}</span>
    </div>

    <div class="load-grid">
      <button class="load-cell" :disabled="!session || loading" @click="refresh">
        <span>CPU</span>
        <strong>{{ percent(snapshot?.loads?.cpuPercent) }}</strong>
      </button>
      <button class="load-cell" :disabled="!session || loading" @click="refresh">
        <span>内存</span>
        <strong>{{ percent(snapshot?.loads?.memoryPercent) }}</strong>
      </button>
      <button class="load-cell" :disabled="!session || loading" @click="refresh">
        <span>硬盘</span>
        <strong>{{ percent(snapshot?.loads?.diskPercent) }}</strong>
      </button>
    </div>

    <section class="monitor-section">
      <div class="section-row">
        <span>进程</span>
        <div class="sort-buttons">
          <button :class="{ active: processSort === 'memory' }" @click="setSort('memory')">内存</button>
          <button :class="{ active: processSort === 'cpu' }" @click="setSort('cpu')">CPU</button>
        </div>
      </div>
      <div class="process-table">
        <div class="process-row head">
          <span>内存M</span>
          <span>CPU%</span>
          <span>进程</span>
        </div>
        <div v-for="proc in processes" :key="`${proc.name}-${proc.memoryMB}-${proc.cpuPercent}`" class="process-row">
          <span>{{ proc.memoryMB.toFixed(1) }}</span>
          <span>{{ proc.cpuPercent.toFixed(1) }}</span>
          <span class="process-name">{{ proc.name }}</span>
        </div>
        <div v-if="processes.length === 0" class="empty-tip compact">暂无数据</div>
      </div>
    </section>

    <section class="monitor-section">
      <div class="section-row">
        <span>网速</span>
        <span>{{ updatedAt }}</span>
      </div>
      <div class="net-table">
        <div v-for="net in networks" :key="net.name" class="net-row">
          <strong>{{ net.name }}</strong>
          <span>↓ {{ speed(net.rxBps) }}</span>
          <span>↑ {{ speed(net.txBps) }}</span>
        </div>
        <div v-if="networks.length === 0" class="empty-tip compact">暂无数据</div>
      </div>
    </section>

    <section class="monitor-section partitions">
      <div class="section-row">
        <span>分区</span>
      </div>
      <div v-for="part in partitions" :key="`${part.fileSystem}-${part.mount}`" class="partition-row">
        <div>
          <strong>{{ part.mount }}</strong>
          <span>{{ part.fileSystem }}</span>
        </div>
        <div>{{ size(part.freeBytes) }} / {{ size(part.totalBytes) }}</div>
      </div>
      <div v-if="partitions.length === 0" class="empty-tip compact">暂无数据</div>
    </section>

    <div class="file-error">{{ errorMessage || '\u00A0' }}</div>
  </div>
</template>

<script setup>
import { computed, onBeforeUnmount, ref, watch } from 'vue';
import { getMonitorSnapshot } from '../services/apiClient';

const props = defineProps({
  session: {
    type: Object,
    default: null,
  },
});

const snapshot = ref(null);
const loading = ref(false);
const errorMessage = ref('');
const processSort = ref('memory');
let timer = null;

const processes = computed(() => snapshot.value?.processes || []);
const networks = computed(() => snapshot.value?.networks || []);
const partitions = computed(() => snapshot.value?.partitions || []);
const updatedAt = computed(() => (snapshot.value?.updatedAt ? new Date(snapshot.value.updatedAt).toLocaleTimeString() : '-'));
const statusText = computed(() => {
  if (!props.session) {
    return '未连接';
  }
  if (loading.value) {
    return '刷新中';
  }
  return '5秒自动刷新';
});

watch(
  () => props.session?.connectionId,
  () => {
    snapshot.value = null;
    errorMessage.value = '';
    restartTimer();
  },
  { immediate: true },
);

onBeforeUnmount(() => {
  stopTimer();
});

function restartTimer() {
  stopTimer();
  if (!props.session?.connectionId) {
    return;
  }
  refresh();
  timer = window.setInterval(refresh, 5000);
}

function stopTimer() {
  if (timer) {
    window.clearInterval(timer);
    timer = null;
  }
}

async function refresh() {
  if (!props.session?.connectionId || loading.value) {
    return;
  }

  loading.value = true;
  errorMessage.value = '';
  try {
    const result = await getMonitorSnapshot(props.session.connectionId, processSort.value);
    snapshot.value = result.snapshot;
  } catch (error) {
    errorMessage.value = error instanceof Error ? error.message : '监控刷新失败';
  } finally {
    loading.value = false;
  }
}

function setSort(value) {
  processSort.value = value;
  refresh();
}

function percent(value) {
  if (typeof value !== 'number') {
    return '-';
  }
  return `${value.toFixed(1)}%`;
}

function speed(value) {
  if (!value) {
    return '0 B/s';
  }
  if (value < 1024) {
    return `${value.toFixed(0)} B/s`;
  }
  if (value < 1024 * 1024) {
    return `${(value / 1024).toFixed(1)} KB/s`;
  }
  return `${(value / (1024 * 1024)).toFixed(1)} MB/s`;
}

function size(value) {
  if (!value) {
    return '-';
  }
  if (value < 1024 * 1024 * 1024) {
    return `${(value / (1024 * 1024)).toFixed(1)} MB`;
  }
  return `${(value / (1024 * 1024 * 1024)).toFixed(1)} GB`;
}
</script>
